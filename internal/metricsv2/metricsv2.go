package metricsv2

// test package for exposing real metrics and analyze on plutono to further develop

import (
	"context"
	"sync"

	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	provisionedInstancesCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "keb",
		Subsystem: "test",
		Name:      "provisioned_counter",
		Help:      "counter of successfully provisioned instances",
	})
	operationsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "keb",
		Subsystem: "test",
		Name:      "operations_total_counter",
		Help:      "Results of operations (total count)",
	}, []string{"type", "state"})
	mutex = sync.Mutex{}
)

func Register(sub event.Subscriber) {
	metricsv2err := prometheus.Register(provisionedInstancesCounter)
	if metricsv2err != nil {
		logrus.Errorf("Error while registering ProvisionedInstancesCounter: %s", metricsv2err.Error())
	} else {
		sub.Subscribe(process.ProvisioningSucceeded{}, Handler)
	}

	metricsv2err = prometheus.Register(operationsCounter)
	if metricsv2err != nil {
		logrus.Errorf("Error while registering OperationsCounter: %s", metricsv2err.Error())
	} else {
		sub.Subscribe(process.OperationStepProcessed{}, Handler)
	}
}

// dont fail anything since it is just test function which is used for gathering informations before development
func Handler(ctx context.Context, ev interface{}) error {
	logrus.Info("metricsv2 test handler called")
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered in test metrics: %v", r)
		}
	}()
	mutex.Lock()
	defer mutex.Unlock()

	switch data := ev.(type) {
	case process.ProvisioningSucceeded:
		// keb_test_provisioned_counter X
		provisionedInstancesCounter.Inc()
	case process.OperationStepProcessed:
		// keb_test_result_operations_total_counter{type="provision", state="in progress"} X
		operationsCounter.WithLabelValues(string(data.Operation.Type), string(data.Operation.State)).Inc()
	default:
		logrus.Error("ev type not supported")
	}

	return nil
}
