package metricsv2

import (
	"context"
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pkg/errors"
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
	prometheusNamespace = "kcp"
	prometheusSubsystem = "keb_v2"
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
	metrics map[counterKey]prometheus.Counter
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
						Name: prometheus.BuildFQName(
							prometheusNamespace,
							prometheusSubsystem,
							fmt.Sprintf(metricNamePattern, string(operationType), string(state)),
						),
						Help: fmt.Sprintf("The counter of %s operations in %s state", operationType, state),
					},
				)
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

func (opCounter *operationsCounter) handler(_ context.Context, event interface{}) error {
	defer func() {
		if recovery := recover(); recovery != nil {
			opCounter.logger.Errorf("panic recovered while handling operation counter: %v", r)
		}
	}()

	var counterKey counterKey
	switch payload := event.(type) {
	case process.ProvisioningFinished:
		counterKey = opCounter.buildKeyFor(payload.Operation.Type, payload.Operation.State, broker.PlanID(payload.Operation.Plan))
	case process.DeprovisioningFinished:
		counterKey = opCounter.buildKeyFor(payload.Operation.Type, payload.Operation.State, broker.PlanID(payload.Operation.Plan))
	case process.UpdateFinished:
		counterKey = opCounter.buildKeyFor(payload.Operation.Type, payload.Operation.State, broker.PlanID(payload.Operation.Plan))
	default:
		return fmt.Errorf("unexpected event type")
	}

	if _, exists := opCounter.metrics[counterKey]; !exists {
		return errors.Errorf("counter with %s not exists", counterKey)
	}

	opCounter.metrics[counterKey].Inc()

	return nil
}

func (op *operationsCounter) buildKeyFor(operationType internal.OperationType, state domain.LastOperationState, planID broker.PlanID) counterKey {
	return counterKey(fmt.Sprintf("%s_%s_%s", operationType, state, planID))
}
