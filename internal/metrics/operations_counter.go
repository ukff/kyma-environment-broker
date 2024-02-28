package metrics

import (
	"context"
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
	`github.com/sirupsen/logrus`
)

// exposed metrics:
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

type counterKey string

type operationStats struct {
	logger     logrus.FieldLogger
	operationsCounters map[counterKey]prometheus.Counter
}

func NewOperationsCounters(logger logrus.FieldLogger) *operationStats {
	return &operationStats{
		logger: logger,
		operationsCounters: make(map[counterKey]prometheus.Counter),
	}
}

func (o *operationStats) Register() {
	for key, counter := range o.createCounters() {
		prometheus.MustRegister(counter)
		o.operationsCounters[key] = counter
	}
}

func (o *operationStats) increaseCounterByKey(key counterKey) error {
	if _, ok := o.operationsCounters[key]; !ok {
		return fmt.Errorf("counter for key %s not found", key)
	}
	o.operationsCounters[key].Inc()
	return nil
}

func (o *operationStats) buildKeyFor(operationType internal.OperationType, state domain.LastOperationState, planID broker.PlanID) counterKey {
	return counterKey(fmt.Sprintf("%s_%s_%s", operationType, state, planID))
}

func (o *operationStats) buildCounterFor(operationType internal.OperationType, state domain.LastOperationState, planID broker.PlanID) (counterKey, prometheus.Counter) {
	return o.buildKeyFor(operationType, state, planID), prometheus.NewCounter(
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

func (o *operationStats) createCounters() map[counterKey]prometheus.Counter {
	result := make(map[counterKey]prometheus.Counter, len(supportedPlans)*len(supportedOperations)*len(supportedStates),)
	for _, plan := range supportedPlans {
		for _, operationType := range supportedOperations {
			for _, state := range supportedStates {
				key, metric := o.buildCounterFor(operationType, state, plan)
				result[key] = metric
			}
		}
	}
	return result
}

func (o *operationStats) handler(_ context.Context, event interface{}) error {
	defer func() {
		if r := recover(); r != nil {
			o.logger.Errorf("panic recovered while handling operation counter: %v", r)
		}
	}()
	
	switch data := event.(type) {
	case process.ProvisioningFinished:
	case process.DeprovisioningFinished:
	case process.UpdateFinished:
		err := o.increaseCounterByKey(o.buildKeyFor(data.Operation.Type, data.Operation.State, broker.PlanID(data.Operation.Plan)))
		if err != nil {
			o.logger.Errorf("unable to increase counter for operation %s: %s", data.Operation.ID, err)
			return err
		}
		return nil
	}
	return fmt.Errorf("unexpected event type")
}
