package metrics

import (
	"context"

	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/metricsrefactor"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/sirupsen/logrus"
)

const (
	prometheusNamespace = "lj"
	prometheusSubsystem = "keb"
)

func Setup(ctx context.Context, subscriber event.Subscriber, db operationsGetter, operationStatsGetter OperationsStatsGetter, instanceStatsGetter InstancesStatsGetter, logger logrus.FieldLogger) {
	// opDurationCollector := NewOperationDurationCollector()
	// StartOpsMetricService(ctx, db, logger)

	//prometheus.MustRegister(opResultCollector, opDurationCollector)

	// This ones connect to DATABASE
	// Question -> when it is called?
	//prometheus.MustRegister(NewOperationsCollector(operationStatsGetter))
	// prometheus.MustRegister(NewInstancesCollector(instanceStatsGetter))

	// This one acts per event
	// PubSub - Publisher and Subscriber
	// Publisher Publish -> call event handler for given event
	// Subscriber Subscribe -> It adds event handler to event (only register handlers, does not call them)
	// This way dont lookup to database, but only to current events.

	/*subscriber.Subscribe(process.ProvisioningStepProcessed{}, opResultCollector.OnProvisioningStepProcessed)
	subscriber.Subscribe(process.DeprovisioningStepProcessed{}, opResultCollector.OnDeprovisioningStepProcessed)
	subscriber.Subscribe(process.UpgradeKymaStepProcessed{}, opResultCollector.OnUpgradeKymaStepProcessed)
	subscriber.Subscribe(process.UpgradeClusterStepProcessed{}, opResultCollector.OnUpgradeClusterStepProcessed)
	subscriber.Subscribe(process.OperationSucceeded{}, metricsre.TestHandler)*/

	subscriber.Subscribe(process.OperationStepProcessed{}, metricsrefactor.OperationStepProcessedHandler)
	subscriber.Subscribe(process.OperationSucceeded{}, metricsrefactor.OperationSucceededHandler)

	/*subscriber.Subscribe(process.ProvisioningSucceeded{}, opResultCollector.OnProvisioningSucceeded)

	// Measure duration
	subscriber.Subscribe(process.ProvisioningSucceeded{}, opDurationCollector.OnProvisioningSucceeded)
	subscriber.Subscribe(process.DeprovisioningStepProcessed{}, opDurationCollector.OnDeprovisioningStepProcessed)
	subscriber.Subscribe(process.OperationSucceeded{}, opDurationCollector.OnOperationSucceeded)
	subscriber.Subscribe(process.OperationStepProcessed{}, opDurationCollector.OnOperationStepProcessed)
	*/
}
