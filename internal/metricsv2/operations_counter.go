package metricsv2

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	
	`github.com/kyma-project/kyma-environment-broker/common/setup`
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// exposed metrics:
// - kcp_keb_v2_operations_{plan_name}_provisioning_failed_total
// - kcp_keb_v2_operations_{plan_name}_provisioning_in_progress_total
// - kcp_keb_v2_operations_{plan_name}_provisioning_succeeded_total
// - kcp_keb_v2_operations_{plan_name}_deprovisioning_failed_total
// - kcp_keb_v2_operations_{plan_name}_deprovisioning_in_progress_total
// - kcp_keb_v2_operations_{plan_name}_deprovisioning_succeeded_total
// - kcp_keb_v2_operations_{plan_name}_update_failed_total
// - kcp_keb_v2_operations_{plan_name}_update_in_progress_total
// - kcp_keb_v2_operations_{plan_name}_update_succeeded_total

const (
	prometheusNamespace = "keb"
	prometheusSubsystem = "kcp_v2"
	metricNamePattern   = "operations_%s_%s_total"
)

var (
	plans = []broker.PlanID{
		broker.AzurePlanID,
		broker.AzureLitePlanID,
		broker.AWSPlanID,
		broker.GCPPlanID,
		broker.SapConvergedCloudPlanID,
		broker.TrialPlanID,
		broker.FreemiumPlanID,
		broker.PreviewPlanName,
	}
	opTypes = []internal.OperationType{
		internal.OperationTypeProvision,
		internal.OperationTypeDeprovision,
		internal.OperationTypeUpdate,
	}
	opStates = []domain.LastOperationState{
		domain.Failed,
		domain.InProgress,
		domain.Succeeded,
	}
)

type counterKey string

type operationStats struct {
	logger       logrus.FieldLogger
	logMu        sync.Mutex
	operations   storage.Operations
	gauges       map[counterKey]prometheus.Gauge
	counters     map[counterKey]prometheus.Counter
	loopInterval time.Duration
}

func NewOperationsCounters(operations storage.Operations, loopInterval time.Duration, logger logrus.FieldLogger) *operationStats {
	return &operationStats{
		logger:       logger.WithField("source", "@metricsv2"),
		gauges:       make(map[counterKey]prometheus.Gauge, len(plans)*len(opTypes)*1),
		counters:     make(map[counterKey]prometheus.Counter, len(plans)*len(opTypes)*2),
		operations:   operations,
		loopInterval: loopInterval,
	}
}

func (os *operationStats) MustRegister(ctx context.Context) {
	// while on testing phase, we don't want to panic app, also MustRegister should be called only once on startup
	defer func() {
		if recovery := recover(); recovery != nil {
			os.logger.Errorf("panic recovered while creating and registering operations metrics: %v", recovery)
		}
	}()
	
	for _, plan := range plans {
		for _, opType := range opTypes {
			for _, opState := range opStates {
				key, err := os.buildKeyFor("init", opType, opState, plan)
				setup.FatalOnError(err)
				fqName := os.buildName(opType, opState)
				labels := prometheus.Labels{"plan_id": string(plan)}
				switch opState {
				case domain.InProgress:
					os.gauges[key] = prometheus.NewGauge(
						prometheus.GaugeOpts{
							Name:        fqName,
							ConstLabels: labels,
						},
					)
				case domain.Failed, domain.Succeeded:
					os.counters[key] = prometheus.NewCounter(
						prometheus.CounterOpts{
							Name:        fqName,
							ConstLabels: labels,
						},
					)
				}
			}
		}
	}
	
	for _, metric := range os.counters {
		prometheus.MustRegister(metric)
	}
	
	for _, metric := range os.gauges {
		prometheus.MustRegister(metric)
	}
	
	go os.getLoop(ctx)
}

func (os *operationStats) Handler(_ context.Context, event interface{}) error {
	defer func() {
		if recovery := recover(); recovery != nil {
			os.logger.Error("panic recovered while handling operation counter: %v", recovery)
		}
	}()

	payload, ok := event.(process.OperationCounting)
	if !ok {
		return fmt.Errorf("expected process.OperationStepProcessed but got %+v", event)
	}

	if domain.LastOperationState(payload.OpState) != domain.Failed && domain.LastOperationState(payload.OpState) != domain.Succeeded {
		return fmt.Errorf("operation state is in progress, but operation counter support events only from failed or succeded operations")
	}

	key, err := os.buildKeyFor(payload.OpId, internal.OperationType(payload.OpType), domain.LastOperationState(payload.OpState), broker.PlanID(payload.PlanID))
	if err != nil {
		os.logger.Error(err)
		return err
	}

	metric, found := os.counters[key]
	if !found {
		return fmt.Errorf("counter not found for key %s", key)
	}

	if metric == nil {
		return fmt.Errorf("counter is nil for key %s", key)
	}

	os.counters[key].Inc()
	return nil
}

func (os *operationStats) getLoop(ctx context.Context) {
	defer func() {
		if recovery := recover(); recovery != nil {
			os.logger.Errorf("panic recovered while handling in progress operation counter: %v", recovery)
		}
	}()

	ticker := time.NewTicker(os.loopInterval)
	for {
		select {
		case <-ticker.C:
			stats, err := os.operations.GetOperationStatsByPlanV2()
			if err != nil {
				os.logger.Errorf("cannot fetch in progress operations: %s", err.Error())
				continue
			}
			updatedStats := make(map[counterKey]struct{})
			for _, stat := range stats {
				key, err := os.buildKeyFor("loop", internal.OperationType(stat.Type), domain.LastOperationState(stat.State), broker.PlanID(stat.PlanID.String))
				if err != nil {
					os.logger.Error(err)
					continue
				}

				metric, found := os.gauges[key]
				if !found {
					os.logger.Errorf("gauge not found for key %s", key)
					continue
				}

				metric.Set(float64(stat.Count))
				updatedStats[key] = struct{}{}
			}

			for key, metric := range os.gauges {
				if _, ok := updatedStats[key]; ok {
					continue
				}
				metric.Set(0)
			}
			
		case <-ctx.Done():
			return
		}
	}
}

func (os *operationStats) buildName(operationType internal.OperationType, state domain.LastOperationState) string {
	return prometheus.BuildFQName(
		prometheusNamespace,
		prometheusSubsystem,
		fmt.Sprintf(metricNamePattern, string(operationType), formatOpState(state)),
	)
}

func (os *operationStats) buildKeyFor(caller string, operationType internal.OperationType, state domain.LastOperationState, planID broker.PlanID) (counterKey, error) {
	if operationType == "" || state == "" || planID == "" {
		return "", fmt.Errorf("caller: %s cannot build key for operationType: %s, state: %s, planID: %s", caller, operationType, state, planID)
	}

	return counterKey(fmt.Sprintf("%s_%s_%s", operationType, formatOpState(state), planID)), nil
}

func formatOpState(state domain.LastOperationState) string {
	return strings.ReplaceAll(string(state), " ", "_")
}