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

// Exposer gather metrics and keep then in memory and expose them to prometheus for fetching them, it gather them by:
// listening in real time for events by "Handler"
// fetching data from database by "Job"

type Exposer interface {
	Handler(ctx context.Context, event interface{}) error
	Job(ctx context.Context)
}

func Register(ctx context.Context, sub event.Subscriber, operations storage.Operations, instanceStatsGetter InstancesStatsGetter, logger logrus.FieldLogger) {
	opDurationCollector := NewOperationDurationCollector()
	prometheus.MustRegister(opDurationCollector)
	prometheus.MustRegister(NewInstancesCollector(instanceStatsGetter))

	sub.Subscribe(process.ProvisioningSucceeded{}, opDurationCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opDurationCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.OperationSucceeded{}, opDurationCollector.OnOperationSucceeded)
	sub.Subscribe(process.OperationStepProcessed{}, opDurationCollector.OnOperationStepProcessed)

	opCollector := NewOperationsCollector(ctx, operations, logger)
	
	opStats := NewOperationsCounters(operations, 5*time.Second, logger)
	opStats.MustRegister(ctx)

	sub.Subscribe(process.OperationCounting{}, opStats.Handler)
	sub.Subscribe(process.DeprovisioningSucceeded{}, opCollector.Handler)
}