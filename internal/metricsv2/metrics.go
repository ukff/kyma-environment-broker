package metricsv2

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
	prometheusNamespacev2 = "kcp"
	prometheusSubsystemv2 = "keb_v2"
)

// Exposer gathers metrics and keeps these in memory and exposes to prometheus for fetching, it gathers them by:
// listening in real time for events by "Handler"
// fetching data from database by "Job"

type Exposer interface {
	MustRegister(ctx context.Context)
	Job(ctx context.Context)
	Handler(ctx context.Context, event interface{}) error
}

type Holder struct {
	OperationDurationCollector *OperationDurationCollector
	InstancesCollector         *InstancesCollector
	OperationsResult           *operationsResult
	OperationStats             *OperationStats
}

func Register(ctx context.Context, sub event.Subscriber, operations storage.Operations, instances storage.Instances, logger logrus.FieldLogger) *Holder {

	opDurationCollector := NewOperationDurationCollector()
	instanceCollector := NewInstancesCollector(instances)
	opResult := NewOperationResult(ctx, operations, logger, time.Second*30, time.Hour*24*7)
	opStats := NewOperationsStats(operations, time.Second*30, logger)

	prometheus.MustRegister(opDurationCollector)
	prometheus.MustRegister(instanceCollector)
	opStats.MustRegister(ctx)

	sub.Subscribe(process.ProvisioningSucceeded{}, opDurationCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opDurationCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.OperationSucceeded{}, opDurationCollector.OnOperationSucceeded)
	sub.Subscribe(process.OperationStepProcessed{}, opDurationCollector.OnOperationStepProcessed)
	sub.Subscribe(process.OperationFinished{}, opStats.Handler)
	sub.Subscribe(process.DeprovisioningSucceeded{}, opResult.Handler)

	return &Holder{
		OperationDurationCollector: opDurationCollector,
		InstancesCollector:         instanceCollector,
		OperationsResult:           opResult,
		OperationStats:             opStats,
	}
}
