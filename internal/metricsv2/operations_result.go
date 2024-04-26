package metricsv2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

type operationsResult struct {
	logger          logrus.FieldLogger
	metrics         *prometheus.GaugeVec
	lastUpdate      time.Time
	operations      storage.Operations
	cache           map[string]internal.Operation
	poolingInterval time.Duration
	sync            sync.Mutex
}

var _ Exposer = (*operationsResult)(nil)

func NewOperationResult(ctx context.Context, db storage.Operations, cfg Config, logger logrus.FieldLogger) *operationsResult {
	opInfo := &operationsResult{
		operations: db,
		lastUpdate: time.Now().Add(-cfg.OperationResultRetentionPeriod),
		logger:     logger,
		cache:      make(map[string]internal.Operation),
		metrics: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespacev2,
			Subsystem: prometheusSubsystemv2,
			Name:      "operation_result",
			Help:      "Results of metrics",
		}, []string{"operation_id", "instance_id", "global_account_id", "plan_id", "type", "state", "error_category", "error_reason", "error"}),
		poolingInterval: cfg.OperationResultPoolingInterval,
	}
	go opInfo.Job(ctx)
	return opInfo
}

func (s *operationsResult) setOperation(op internal.Operation, val float64) {
	labels := make(map[string]string)
	labels["operation_id"] = op.ID
	labels["instance_id"] = op.InstanceID
	labels["global_account_id"] = op.GlobalAccountID
	labels["plan_id"] = op.ProvisioningParameters.PlanID
	labels["type"] = string(op.Type)
	labels["state"] = string(op.State)
	labels["error_category"] = string(op.LastError.Component())
	labels["error_reason"] = string(op.LastError.Reason())
	labels["error"] = op.LastError.Error()
	s.metrics.With(labels).Set(val)
}

// operation_result metrics works on 0/1 system.
// each metric have labels which identify the operation data by Operation ID
// if metrics with OpId is set to 1, then it means that this event happen in KEB system and will be persisted in Prometheus Server
// metrics set to 0 means that this event is outdated, and will be replaced by new one which happen
func (s *operationsResult) updateOperation(op internal.Operation) {
	defer s.sync.Unlock()
	s.sync.Lock()

	oldOp, found := s.cache[op.ID]
	if found {
		s.setOperation(oldOp, 0)
		Debug(s.logger,"@Debug", fmt.Sprintf("@metricsv2 : operation ID %s set to 0 with state %s and type %s", op.ID, op.State, op.Type))
	}
	s.setOperation(op, 1)
	Debug(s.logger,"@Debug", fmt.Sprintf("@metricsv2 : operation ID %s set to 1 with state %s and type %s", op.ID, op.State, op.Type))
	if op.State == domain.Failed || op.State == domain.Succeeded {
		Debug(s.logger,"@Debug", fmt.Sprintf("@metricsv2 : deleting operation ID %s from cache with status %s and type %s", op.ID, op.State, op.Type))
		delete(s.cache, op.ID)
	} else {
		Debug(s.logger,"@Debug", fmt.Sprintf("@metricsv2 : adding operation ID %s to cache with status %s and type %s", op.ID, op.State, op.Type))
		s.cache[op.ID] = op
	}
}

func (s *operationsResult) updateMetrics() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	now := time.Now()
	operations, err := s.operations.ListOperationsInTimeRange(s.lastUpdate, now)
	Debug(s.logger,"@Debug", fmt.Sprintf("@metricsv2 : getting operations from window %s to %s", s.lastUpdate, now))
	if err != nil {
		return fmt.Errorf("failed to list metrics: %v", err)
	}
	Debug(s.logger,"@Debug", fmt.Sprintf("@metricsv2 : %d ops processing start", len(operations)))
	for _, op := range operations {
		Debug(s.logger,"@Debug", fmt.Sprintf("@metricsv2 : processing operation ID %s, created_at %s updated_at %s", op.ID, op.CreatedAt, op.UpdatedAt))
		s.updateOperation(op)
	}
	Debug(s.logger,"@Debug", fmt.Sprintf("@metricsv2 : %d ops processing end", len(operations)))
	s.lastUpdate = now
	return nil
}

func (s *operationsResult) Handler(ctx context.Context, event interface{}) error {
	defer s.sync.Unlock()
	s.sync.Lock()

	defer func() {
		Debug(s.logger, "@metricsv2", "Handler func end")
		if recovery := recover(); recovery != nil {
			Debug(s.logger, "@metricsv2", "Handler func end with defer")
			s.logger.Errorf("panic recovered while handling operation info event: %v", recovery)
		}
	}()

	switch ev := event.(type) {
	case process.DeprovisioningSucceeded:
		s.logger.Infof("dep succeeded")
		Debug(s.logger, "@metricsv2", "DeprovisioningSucceeded event received")
		s.updateOperation(ev.Operation.Operation)
	default:
		Debug(s.logger, "@metricsv2", fmt.Sprintf("unexpected event type: %T", event))
		s.logger.Errorf("unexpected event type: %T", event)
	}
	return nil
}

func (s *operationsResult) Job(ctx context.Context) {
	Debug(s.logger, "@metricsv2", "Job started")
	defer func() {
		Debug(s.logger, "@metricsv2", "Job ended")
		if recovery := recover(); recovery != nil {
			Debug(s.logger, "@metricsv2", "Panic happen in Job")
			s.logger.Errorf("panic recovered while performing operation info job: %v", recovery)
		}
	}()

	s.logger.Errorf("updateMetrics called")
	Debug(s.logger, "@metricsv2", "Job started fist time")
	if err := s.updateMetrics(); err != nil {
		Debug(s.logger, "@metricsv2", "Job started first time failed")
		s.logger.Error("failed to update metrics metrics", err)
	}

	ticker := time.NewTicker(s.poolingInterval)
	for {
		select {
		case <-ticker.C:
			Debug(s.logger, "@metricsv2", "tick tick")
			if err := s.updateMetrics(); err != nil {
				Debug(s.logger, "@metricsv2", "in Job loop failed to update metrics")
				s.logger.Error("failed to update operation info metrics", err)
			}
		case <-ctx.Done():
			Debug(s.logger, "@metricsv2", "ctx done")
			s.logger.Error("ctx done")
			return
		}
	}
}
