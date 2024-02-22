package metrics

import (
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/metricsv2"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

func RegisterAll(sub event.Subscriber, operationStatsGetter OperationsStatsGetter, instanceStatsGetter InstancesStatsGetter) {
	opResultCollector := NewOperationResultCollector()
	opDurationCollector := NewOperationDurationCollector()
	stepResultCollector := NewStepResultCollector()
	prometheus.MustRegister(opResultCollector, opDurationCollector, stepResultCollector)
	prometheus.MustRegister(NewOperationsCollector(operationStatsGetter))
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

	// test of metrics for upcoming new implementation
	metricsv2err := prometheus.Register(metricsv2.ProvisionedInstancesCounter)
	if metricsv2err != nil {
		logrus.Errorf("Error while registering ProvisionedInstancesCounter: %s", metricsv2err.Error())
	} else {
		sub.Subscribe(process.ProvisioningSucceeded{}, metricsv2.Handler)
	}

	metricsv2err = prometheus.Register(metricsv2.OperationsCounter)
	if metricsv2err != nil {
		logrus.Errorf("Error while registering OperationsCounter: %s", metricsv2err.Error())
	} else {
		sub.Subscribe(process.OperationStepProcessed{}, metricsv2.Handler)
	}
}
