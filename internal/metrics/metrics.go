package metrics

import (
	"context"
	"time"
	
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	prometheusNamespace = "kcp"
	prometheusSubsystem = "keb"
)

func Register(ctx context.Context, sub event.Subscriber, operations storage.Operations, instanceStatsGetter InstancesStatsGetter, logger logrus.FieldLogger) {
	opDurationCollector := NewOperationDurationCollector()
	prometheus.MustRegister(opDurationCollector)
	prometheus.MustRegister(NewOperationsCollector(operations))
	prometheus.MustRegister(NewInstancesCollector(instanceStatsGetter))
	
	sub.Subscribe(process.ProvisioningSucceeded{}, opDurationCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opDurationCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.OperationSucceeded{}, opDurationCollector.OnOperationSucceeded)
	sub.Subscribe(process.OperationStepProcessed{}, opDurationCollector.OnOperationStepProcessed)

	StartOpsMetricService(ctx, operations, logger)

	// test of metrics for upcoming new implementation
	operationsCounter := NewOperationsCounters(operations, 5*time.Second, logger)
	operationsCounter.MustRegister(ctx)

	sub.Subscribe(process.OperationCounting{}, operationsCounter.Handler)
}
