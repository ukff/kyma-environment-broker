package metrics

import (
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/prometheus/client_golang/prometheus"
)

func RegisterAll(sub event.Subscriber, operationStatsGetter OperationsStatsGetter, instanceStatsGetter InstancesStatsGetter) {
	opResultCollector := NewOperationResultCollector()
	opDurationCollector := NewOperationDurationCollector()
	stepResultCollector := NewStepResultCollector()

	prometheus.MustRegister(opResultCollector, opDurationCollector, stepResultCollector)

	// This ones connect to DATABASE
	// Question -> when it is called?
	prometheus.MustRegister(NewOperationsCollector(operationStatsGetter))
	prometheus.MustRegister(NewInstancesCollector(instanceStatsGetter))

	// This one acts per event
	// PubSub - Publisher and Subscriber
	// Publisher Publish -> call event handler for given event
	// Subscriber Subscribe -> It adds event handler to event (only register handlers, does not call them)
	// This way dont lookup to database, but only to current events.
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
}
