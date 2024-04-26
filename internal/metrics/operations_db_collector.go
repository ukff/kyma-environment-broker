package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/metricsv2"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

// Retention is the default time and date for obtaining operations by the database query
// For performance reasons, it is not possible to query entire operations database table,
// so instead KEB queries the database for last 14 days worth of data and then for deltas
// during the ellapsed time
const (
	Retention       = 14 * 24 * time.Hour
	PollingInterval = 30 * time.Second
)

type operationsGetter interface {
	ListOperationsInTimeRange(from, to time.Time) ([]internal.Operation, error)
}

type opsMetricService struct {
	logger     logrus.FieldLogger
	operations *prometheus.GaugeVec
	lastUpdate time.Time
	db         operationsGetter
	cache      map[string]internal.Operation
}

// StartOpsMetricService creates service for exposing prometheus metrics for operations.
//
// This is intended as a replacement for OperationResultCollector to address shortcomings
// of the initial implementation - lack of consistency and non-aggregatable metric desing.
// The underlying data is fetched asynchronously from the KEB SQL database to provide
// consistency and the operation result state is exposed as a label instead of a value to
// enable common gauge aggregation.

// kcp_keb_operation_result

func StartOpsMetricService(ctx context.Context, db operationsGetter, logger logrus.FieldLogger) {
	svc := &opsMetricService{
		db:         db,
		lastUpdate: time.Now().Add(-Retention),
		logger:     logger,
		cache:      make(map[string]internal.Operation),
		operations: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "operation_result",
			Help:      "Results of operations",
		}, []string{"operation_id", "instance_id", "global_account_id", "plan_id", "type", "state", "error_category", "error_reason"}),
	}
	go svc.run(ctx)
}

func (s *opsMetricService) setOperation(op internal.Operation, val float64) {
	labels := make(map[string]string)
	labels["operation_id"] = op.ID
	labels["instance_id"] = op.InstanceID
	labels["global_account_id"] = op.GlobalAccountID
	labels["plan_id"] = op.Plan
	labels["type"] = string(op.Type)
	labels["state"] = string(op.State)
	labels["error_category"] = string(op.LastError.Component())
	labels["error_reason"] = string(op.LastError.Reason())
	s.operations.With(labels).Set(val)
}

func (s *opsMetricService) updateOperation(op internal.Operation) {
	oldOp, found := s.cache[op.ID]
	if found {
		s.setOperation(oldOp, 0)
	}
	s.setOperation(op, 1)
	if op.State == domain.Failed || op.State == domain.Succeeded {
		delete(s.cache, op.ID)
	} else {
		s.cache[op.ID] = op
	}
}

func (s *opsMetricService) updateMetrics() (err error) {
	metricsv2.Debug(s.logger, "@metricsv1", "Job started")
	defer func() {
		metricsv2.Debug(s.logger, "@metricsv1", "Job started")
		if r := recover(); r != nil {
			metricsv2.Debug(s.logger, "@metricsv1", "Panic happen in Job")
			// it's not desirable to panic metrics goroutine, instead it should return and log the error
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()
	now := time.Now()
	operations, err := s.db.ListOperationsInTimeRange(s.lastUpdate, now)
	if err != nil {
		metricsv2.Debug(s.logger, "@Debug", "@metricsv1 failed to list operations")
		return fmt.Errorf("failed to list operations: %v", err)
	}
	metricsv2.Debug(s.logger, "@Debug", fmt.Sprintf("@metricsv1 : %d ops processing start", len(operations)))
	for _, op := range operations {
		s.updateOperation(op)
	}
	metricsv2.Debug(s.logger, "@Debug", fmt.Sprintf("@metricsv1 : %d ops processing end", len(operations)))
	s.lastUpdate = now
	return nil
}

func (s *opsMetricService) run(ctx context.Context) {
	metricsv2.Debug(s.logger, "@metricsv1", "tick tick")
	if err := s.updateMetrics(); err != nil {
		metricsv2.Debug(s.logger, "@metricsv1", "Job started fist time")
		s.logger.Error("failed to update operations metrics", err)
	}
	ticker := time.NewTicker(PollingInterval)
	for {
		select {
		case <-ticker.C:
			metricsv2.Debug(s.logger, "@metricsv1", "tick tick")
			if err := s.updateMetrics(); err != nil {
				metricsv2.Debug(s.logger, "@metricsv1", "ctx done")
				s.logger.Error("failed to update operations metrics", err)
			}
		case <-ctx.Done():
			metricsv2.Debug(s.logger, "@metricsv1", "ctx done")
			s.logger.Error("ctx done")
			return
		}
	}
}
