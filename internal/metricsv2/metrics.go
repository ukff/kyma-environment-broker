package metricsv2

import (
	`context`
	`time`
	
	`github.com/kyma-project/kyma-environment-broker/internal/event`
	`github.com/kyma-project/kyma-environment-broker/internal/process`
	`github.com/kyma-project/kyma-environment-broker/internal/storage`
	`github.com/sirupsen/logrus`
)

const (
	prometheusNamespacev2 = "kcp"
	prometheusSubsystemv2 = "keb_v2"
)

// Exposer gather metrics and keep then in memory and expose them to prometheus for fetching them, it gather them by:
// listening in real time for events by "Handler"
// fetching data from database by "Job"

type Exposer interface {
	Handler(ctx context.Context, event interface{}) error
	Job(ctx context.Context)
}

func Register(ctx context.Context, sub event.Subscriber, operations storage.Operations, logger logrus.FieldLogger) {
	
	operationsCollector := NewOperationInfo(ctx, operations, logger, "operation_result_v2")
	
	// test of metrics for upcoming new implementation
	operationsCounter := NewOperationsStats(operations, 5*time.Second, logger)
	operationsCounter.MustRegister(ctx)
	
	sub.Subscribe(process.OperationCounting{}, operationsCounter.Handler)
	sub.Subscribe(process.DeprovisioningSucceeded{}, operationsCollector.Handler)
}
