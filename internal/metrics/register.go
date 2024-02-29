package metrics

import (
	"context"
	
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	prometheusNamespace = "kcp"
	prometheusSubsystem = "keb"
)

func Register(ctx context.Context, sub event.Subscriber, db operationsGetter, instanceStatsGetter InstancesStatsGetter, logger logrus.FieldLogger) {
	opDurationCollector := NewOperationDurationCollector()
	opCounters := NewOperationsCounters(logger)
	
	prometheus.MustRegister(opDurationCollector)
	prometheus.MustRegister(NewInstancesCollector(instanceStatsGetter))
	opCounters.MustRegister()
	
	sub.Subscribe(process.ProvisioningSucceeded{}, opDurationCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opDurationCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.OperationSucceeded{}, opDurationCollector.OnOperationSucceeded)
	sub.Subscribe(process.OperationStepProcessed{}, opDurationCollector.OnOperationStepProcessed)
	
	sub.Subscribe(process.ProvisioningFinished{}, opCounters.handler)
	sub.Subscribe(process.DeprovisioningFinished{}, opCounters.handler)
	sub.Subscribe(process.UpdateFinished{}, opCounters.handler)
	
	StartOpsMetricService(ctx, db, logger)
}