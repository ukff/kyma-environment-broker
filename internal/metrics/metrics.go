package metrics

import (
	debug "github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/prometheus/client_golang/prometheus"
)

func RegisterAll(sub event.Subscriber, operationStatsGetter OperationsStatsGetter, instanceStatsGetter InstancesStatsGetter) {
	debug.Log("register NewOperationsCollector start")
	prometheus.MustRegister(NewOperationsCollector(operationStatsGetter))
	debug.Log("register NewOperationsCollector end")

	debug.Log("register NewInstancesCollector start")
	prometheus.MustRegister(NewInstancesCollector(instanceStatsGetter))
	debug.Log("register NewInstancesCollector start")

	/*opResultCollector := NewOperationResultCollector()
	opDurationCollector := NewOperationDurationCollector()
	stepResultCollector := NewStepResultCollector()
	prometheus.MustRegister(opResultCollector, opDurationCollector, stepResultCollector)

	prometheus.MustRegister(NewInstancesCollector(instanceStatsGetter))

	sub.Subscribe(process.ProvisioningStepProcessed{}, opResultCollector.OnProvisioningStepProcessed)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opResultCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.UpgradeKymaStepProcessed{}, opResultCollector.OnUpgradeKymaStepProcessed)
	sub.Subscribe(process.UpgradeClusterStepProcessed{}, opResultCollector.OnUpgradeClusterStepProcessed)
	sub.Subscribe(process.ProvisioningSucceeded{}, opResultCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.ProvisioningSucceeded{}, opDurationCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opDurationCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.ProvisioningStepProcessed{}, stepResultCollector.OnProvisioningStepProcessed)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, stepResultCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.OperationStepProcessed{}, stepResultCollector.OnOperationStepProcessed)
	sub.Subscribe(process.OperationStepProcessed{}, opResultCollector.OnOperationStepProcessed)
	sub.Subscribe(process.OperationSucceeded{}, opResultCollector.OnOperationSucceeded)
	sub.Subscribe(process.OperationSucceeded{}, opDurationCollector.OnOperationSucceeded)
	sub.Subscribe(process.OperationStepProcessed{}, opDurationCollector.OnOperationStepProcessed)
	*/
}
