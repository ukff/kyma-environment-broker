package metricsv2

import (
	"context"
	"fmt"
	`strings`
	`sync`
	
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
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
	supportedPlans = []broker.PlanID{
		broker.AzurePlanID,
		broker.AzureLitePlanID,
		broker.AWSPlanID,
		broker.GCPPlanID,
		broker.SapConvergedCloudPlanID,
		broker.TrialPlanID,
		broker.FreemiumPlanID,
		broker.PreviewPlanName,
	}
	supportedOperations = []internal.OperationType{
		internal.OperationTypeProvision,
		internal.OperationTypeDeprovision,
		internal.OperationTypeUpdate,
	}
	supportedStates = []domain.LastOperationState{
		domain.Failed,
		domain.InProgress,
		domain.Succeeded,
	}
)

type counterKey string

type operationsCounter struct {
	logger  logrus.FieldLogger
	logMu sync.Mutex
	metrics map[counterKey]prometheus.Counter
}

func formatOpState(state domain.LastOperationState) string {
	return strings.Replace(string(state), " ", "_", -1)
}

func NewOperationsCounters(logger logrus.FieldLogger) *operationsCounter {
	operationsCounter := &operationsCounter{
		logger:  logger,
		metrics: make(map[counterKey]prometheus.Counter, len(supportedPlans)*len(supportedOperations)*len(supportedStates)),
	}
	for _, plan := range supportedPlans {
		for _, operationType := range supportedOperations {
			for _, state := range supportedStates {
				operationsCounter.metrics[operationsCounter.buildKeyFor(operationType, state, plan)] = prometheus.NewCounter(
					prometheus.CounterOpts{
						Name: operationsCounter.buildName(operationType, state),
						ConstLabels: prometheus.Labels{"plan_id": string(plan)},
					},
				)
				operationsCounter.Log(fmt.Sprintf("new metric -> %s", operationsCounter.buildKeyFor(operationType, state, plan)), false)
			}
		}
	}
	
	return operationsCounter
}

func (opCounter *operationsCounter) MustRegister() {
	for _, metric := range opCounter.metrics {
		prometheus.MustRegister(metric)
	}
}

func (opCounter *operationsCounter) Handler(_ context.Context, event interface{}) error {
	defer func() {
		if recovery := recover(); recovery != nil {
			opCounter.Log(fmt.Sprintf("panic recovered while handling operation counter: %v", recovery), true)
			opCounter.logger.Errorf("panic recovered while handling operation counter: %v", recovery)
		}
	}()
	
	fmt.Println("Handler start")
	payload, ok := event.(process.OperationCounting)
	// fmt.Println(fmt.Sprintf("Payload: %+v", payload))
	if !ok {
		opCounter.Log(fmt.Sprintf("expected process.OperationStepProcessed but got %+v", event), true)
		return fmt.Errorf("expected process.OperationStepProcessed but got %+v", event)
	}
	
	counterKey := opCounter.buildKeyFor(payload.OpType, payload.OpState, payload.PlanID)
	if counterKey == "" {
		opCounter.Log(fmt.Sprintf("counter key is empty for operation %+v", payload), true)
		return fmt.Errorf("counter key is empty")
	}
	
	metric, found := opCounter.metrics[counterKey]
	if !found {
		opCounter.Log(fmt.Sprintf("counter not found for key %s", counterKey), true)
		return fmt.Errorf("counter not found for key %s", counterKey)
	}
	if metric == nil {
		opCounter.Log(fmt.Sprintf("counter is nil for key %s", counterKey), true)
		return fmt.Errorf("counter is nil for key %s", counterKey)
	}
	
	opCounter.Log(fmt.Sprintf("incrementing counter %s", counterKey), false)
	metric.Inc()
	opCounter.Log(fmt.Sprintf("counter %s incremented", counterKey), false)
	
	return nil
}

func (opCounter *operationsCounter) buildName(operationType internal.OperationType, state domain.LastOperationState) string {
	return prometheus.BuildFQName(
		prometheusNamespace,
		prometheusSubsystem,
		fmt.Sprintf(metricNamePattern, string(operationType), formatOpState(state)),
	)
}

func (opCounter *operationsCounter) buildKeyFor(operationType internal.OperationType, state domain.LastOperationState, planID broker.PlanID) counterKey {
	if operationType == "" || state == "" || planID == "" {
		opCounter.Log(fmt.Sprintf("cannot build key for operationType: %s, state: %s, planID: %s", operationType, state, planID), true)
		return ""
	}
	
	return counterKey(fmt.Sprintf("%s_%s_%s", operationType, formatOpState(state), planID))
}

func (opCounter *operationsCounter) Log(msg string, err bool) {
	opCounter.logMu.Lock()
	defer opCounter.logMu.Unlock()
	
	if err {
		opCounter.logger.Errorf("@metrics error while handling operation counter %s", msg)
	} else {
		opCounter.logger.Infof("@metrics operation counter handled %s", msg)
	}
}