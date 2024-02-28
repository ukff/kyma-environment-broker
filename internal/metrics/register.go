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
	opCounters := NewOperationsCounters()
	
	prometheus.MustRegister(opDurationCollector)
	prometheus.MustRegister(NewInstancesCollector(instanceStatsGetter))
	opCounters.Register()
	
	sub.Subscribe(process.ProvisioningSucceeded{}, opDurationCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opDurationCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.OperationSucceeded{}, opDurationCollector.OnOperationSucceeded)
	sub.Subscribe(process.OperationStepProcessed{}, opDurationCollector.OnOperationStepProcessed)
	
	sub.Subscribe(process.ProvisioningFinished{}, opCounters.onOperationFinished)
	sub.Subscribe(process.DeprovisioningFinished{}, opCounters.onOperationFinished)
	sub.Subscribe(process.UpdateFinished{}, opCounters.onOperationFinished)
	
	StartOpsMetricService(ctx, db, logger)
}