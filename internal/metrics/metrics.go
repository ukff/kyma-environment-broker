package metrics

import (
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/metricsv2"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	prometheusNamespace = "kcp"
	prometheusSubsystem = "keb"
)

/*var noAction = func(ctx context.Context, ev interface{}) error {
	return nil
}*/

func RegisterAll(
	sub event.Subscriber, operationStatsGetter OperationsStatsGetter, instanceStatsGetter InstancesStatsGetter,
) {
	opDurationCollector := NewOperationDurationCollector()
	prometheus.MustRegister(opDurationCollector)
	prometheus.MustRegister(NewOperationsCollector(operationStatsGetter))
	prometheus.MustRegister(NewInstancesCollector(instanceStatsGetter))

	/*
		sub.Subscribe(process.ProvisioningStepProcessed{}, noAction)
		sub.Subscribe(process.DeprovisioningStepProcessed{}, noAction)
		sub.Subscribe(process.UpgradeKymaStepProcessed{}, noAction)
		sub.Subscribe(process.UpgradeClusterStepProcessed{}, noAction)
		sub.Subscribe(process.ProvisioningSucceeded{}, noAction)
		sub.Subscribe(process.ProvisioningStepProcessed{}, noAction)
		sub.Subscribe(process.DeprovisioningStepProcessed{}, noAction)
		sub.Subscribe(process.OperationStepProcessed{}, noAction)
		sub.Subscribe(process.OperationStepProcessed{}, noAction)
		sub.Subscribe(process.OperationSucceeded{}, noAction)
	*/

	sub.Subscribe(process.ProvisioningSucceeded{}, opDurationCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opDurationCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.OperationSucceeded{}, opDurationCollector.OnOperationSucceeded)
	sub.Subscribe(process.OperationStepProcessed{}, opDurationCollector.OnOperationStepProcessed)

	// test of metrics for upcoming new implementation
	metricsv2.Register(sub)
}
