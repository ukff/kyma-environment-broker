package metrics

import (
	"context"
	"fmt"
	
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	`github.com/kyma-project/kyma-environment-broker/internal/process`
	"github.com/pivotal-cf/brokerapi/v8/domain"
	`github.com/pkg/errors`
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

const (
	metricNamePattern = "operations_%s_%s_total"
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
	logger   logrus.FieldLogger
	counters map[counterKey]prometheus.Counter
}

func NewOperationsCounters(logger logrus.FieldLogger) *operationsCounter {
	stats := &operationsCounter{
		logger:   logger,
		counters: make(map[counterKey]prometheus.Counter, len(supportedPlans)*len(supportedOperations)*len(supportedStates)),
	}
	for _, plan := range supportedPlans {
		for _, operationType := range supportedOperations {
			for _, state := range supportedStates {
				stats.counters[stats.buildKeyFor(operationType, state, plan)] = prometheus.NewCounter(
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
	return stats
}

func (o *operationsCounter) MustRegister() {
	for _, counter := range o.counters {
		prometheus.MustRegister(counter)
	}
}

func (o *operationsCounter) handler(_ context.Context, event interface{}) error {
	defer func() {
		if r := recover(); r != nil {
			o.logger.Errorf("panic recovered while handling operation counter: %v", r)
		}
	}()
	
	var counterKey counterKey
	switch e := event.(type) {
	case process.ProvisioningFinished:
		counterKey = o.buildKeyFor(e.Operation.Type, e.Operation.State, broker.PlanID(e.Operation.Plan))
	case process.DeprovisioningFinished:
		counterKey = o.buildKeyFor(e.Operation.Type, e.Operation.State, broker.PlanID(e.Operation.Plan))
	case process.UpdateFinished:
		counterKey = o.buildKeyFor(e.Operation.Type, e.Operation.State, broker.PlanID(e.Operation.Plan))
	default:
		return fmt.Errorf("unexpected event type")
	}
	
	return o.increase(counterKey)
}

func (o *operationsCounter) increase(key counterKey) error{
	if _, exists := o.counters[key]; !exists {
		return errors.Errorf("counter with %s not exists", key)
	}
	o.counters[key].Inc()
	return nil
}

func (o *operationsCounter) buildKeyFor(operationType internal.OperationType, state domain.LastOperationState, planID broker.PlanID) counterKey {
	return counterKey(fmt.Sprintf("%s_%s_%s", operationType, state, planID))
}