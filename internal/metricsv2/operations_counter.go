package metricsv2

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

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

func NewOperationsCounters(ctx context.Context, operations storage.Operations, loopInterval time.Duration, logger logrus.FieldLogger) (*operationStats, error) {
	os := &operationStats{
		logger:       logger,
		gauges:       make(map[counterKey]prometheus.Gauge, len(plans)*len(opTypes)*1),
		counters:     make(map[counterKey]prometheus.Counter, len(plans)*len(opTypes)*2),
		operations:   operations,
		loopInterval: loopInterval,
	}
	for _, plan := range plans {
		for _, opType := range opTypes {
			for _, opState := range opStates {
				key, err := os.buildKeyFor(opType, opState, plan)
				if err != nil {
					return nil, err
				}
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

	go os.getLoop(ctx)

	return os, nil
}

func (os *operationStats) MustRegister() {
	for _, metric := range os.counters {
		prometheus.MustRegister(metric)
	}
	for _, metric := range os.gauges {
		prometheus.MustRegister(metric)
	}
}

func (os *operationStats) Handler(_ context.Context, event interface{}) error {
	defer func() {
		if recovery := recover(); recovery != nil {
			os.Log(fmt.Sprintf("panic recovered while handling operation counter: %v", recovery), true)
		}
	}()

	payload, ok := event.(process.OperationCounting)
	if !ok {
		os.Log(fmt.Sprintf("expected process.OperationStepProcessed but got %+v", event), true)
		return fmt.Errorf("expected process.OperationStepProcessed but got %+v", event)
	}

	if domain.LastOperationState(payload.OpState) != domain.Failed && domain.LastOperationState(payload.OpState) != domain.Succeeded {
		return fmt.Errorf("operation state is in progress, but operation counter support events only from failed or succeded operations")
	}

	key, err := os.buildKeyFor(internal.OperationType(payload.OpType), domain.LastOperationState(payload.OpState), broker.PlanID(payload.PlanID))
	if err != nil {
		os.Log(err.Error(), true)
		return err
	}

	metric, found := os.counters[key]
	if !found {
		os.Log(fmt.Sprintf("counter not found for key %s", key), true)
		return fmt.Errorf("counter not found for key %s", key)
	}

	if metric == nil {
		os.Log(fmt.Sprintf("counter is nil for key %s", key), true)
		return fmt.Errorf("counter is nil for key %s", key)
	}

	os.counters[key].Inc()
	os.Log(fmt.Sprintf("counter %s incremented", key), false)

	return nil
}

func (os *operationStats) getLoop(ctx context.Context) {
	defer func() {
		if recovery := recover(); recovery != nil {
			os.Log(fmt.Sprintf("panic recovered while handling in progress operation counter: %v", recovery), true)
		}
	}()

	ticker := time.NewTicker(os.loopInterval)
	os.Log(fmt.Sprintf("start in_progress operation metrics gathering by every %v", ticker), false)
	for {
		select {
		case <-ticker.C:
			stats, err := os.operations.GetOperationStatsByPlanV2()
			if err != nil {
				os.Log(fmt.Sprintf("cannot fetch in progress operations: %s", err.Error()), true)
				continue
			}

			updatedStats := make(map[counterKey]struct{}, 0)

			for _, stat := range stats {
				os.Log(fmt.Sprintf("stat: %d %s %s %s", stat.Count, stat.Type, stat.State, stat.PlanID.String), false)

				key, err := os.buildKeyFor(internal.OperationType(stat.Type), domain.LastOperationState(stat.State), broker.PlanID(stat.PlanID.String))
				if err != nil {
					os.Log(err.Error(), true)
					continue
				}

				metric, found := os.gauges[key]
				if !found {
					os.Log(fmt.Sprintf("gauge not found for key %s", key), true)
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
			os.Log("in_progress operations metrics updated", false)
		case <-ctx.Done():
			os.Log("in_progress operations metrics stop. ctx done", false)
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

func (os *operationStats) buildKeyFor(operationType internal.OperationType, state domain.LastOperationState, planID broker.PlanID) (counterKey, error) {
	if operationType == "" || state == "" || planID == "" {
		os.Log(fmt.Sprintf("cannot build key for operationType: %s, state: %s, planID: %s", operationType, state, planID), true)
		return counterKey(""), fmt.Errorf("cannot build key for operationType: %s, state: %s, planID: %s", operationType, state, planID)
	}

	return counterKey(fmt.Sprintf("%s_%s_%s", operationType, formatOpState(state), planID)), nil
}

func (os *operationStats) Log(msg string, err bool) {
	os.logMu.Lock()
	defer os.logMu.Unlock()

	if err {
		os.logger.Errorf("@metricsv2: error while handling operation counter -> %s", msg)
	} else {
		os.logger.Infof("@metricsv2: operation counter handled -> %s", msg)
	}
}

func formatOpState(state domain.LastOperationState) string {
	return strings.ReplaceAll(string(state), " ", "_")
}
