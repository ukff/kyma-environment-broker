package metrics

import (
	"context"
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
)

// OperationsStatsGetter provides metrics, which shows how many operations were done for the following plans:

// - kcp_keb_operations_{plan_name}_provisioning_failed_total
// - kcp_keb_operations_{plan_name}_provisioning_in_progress_total
// - kcp_keb_operations_{plan_name}_provisioning_succeeded_total
// - kcp_keb_operations_{plan_name}_deprovisioning_failed_total
// - kcp_keb_operations_{plan_name}_deprovisioning_in_progress_total
// - kcp_keb_operations_{plan_name}_deprovisioning_succeeded_total
// - kcp_keb_operations_{plan_name}_update_failed_total
// - kcp_keb_operations_{plan_name}_update_in_progress_total
// - kcp_keb_operations_{plan_name}_update_succeeded_total

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

type (
	counterKey string

	operationStats struct {
		operationsCounters map[counterKey]prometheus.Counter
	}
)

func NewOperationsCounters() *operationStats {
	return &operationStats{
		operationsCounters: make(map[counterKey]prometheus.Counter),
	}
}

func (o *operationStats) Register() {
	for key, counter := range o.createMetrics() {
		prometheus.MustRegister(counter)
		o.operationsCounters[key] = counter
	}
}

func (o *operationStats) increaseCounter(
	operationType internal.OperationType, state domain.LastOperationState, plan broker.PlanID,
) error {
	key := o.buildKeyFor(operationType, state, plan)
	if _, ok := o.operationsCounters[key]; !ok {
		return fmt.Errorf("counter for key %s not found", key)
	}
	o.operationsCounters[key].Inc()
	return nil
}

func (o *operationStats) buildKeyFor(
	operationType internal.OperationType, state domain.LastOperationState, planID broker.PlanID,
) counterKey {
	return counterKey(fmt.Sprintf("%s_%s_%s", operationType, state, planID))
}

func (o *operationStats) buildMetricFor(
	operationType internal.OperationType, state domain.LastOperationState, planID broker.PlanID,
) (counterKey, prometheus.Counter) {
	key := o.buildKeyFor(operationType, state, planID)
	return key, prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name: prometheus.BuildFQName(
				prometheusNamespace, prometheusSubsystem,
				fmt.Sprintf("operations_%s_%s_total", string(operationType), string(state)),
			),
			Help: fmt.Sprintf("The number of %s operations in %s state", operationType, state),
		},
	)
}

func (o *operationStats) createMetrics() map[counterKey]prometheus.Counter {
	counters := make(
		map[counterKey]prometheus.Counter, len(supportedPlans)*len(supportedOperations)*len(supportedStates),
	)
	for _, plan := range supportedPlans {
		for _, operationType := range supportedOperations {
			for _, state := range supportedStates {
				key, metric := o.buildMetricFor(operationType, state, plan)
				counters[key] = metric
			}
		}
	}
	return counters
}

func (o *operationStats) onOperationFinished(_ context.Context, operation interface{}) error {
	switch data := operation.(type) {
	case process.ProvisioningFinished:
	case process.DeprovisioningFinished:
	case process.UpdateFinished:
		return o.increaseCounter(data.Operation.Type, data.Operation.State, broker.PlanID(data.Operation.Plan))
	}
	return fmt.Errorf("unexpected event type")
}
