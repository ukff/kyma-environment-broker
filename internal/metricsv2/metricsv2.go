package metricsv2

import (
	"context"

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
)

// dont fail anything since it is just test function which is used for gathering informations before development
func Handler(ctx context.Context, ev interface{}) error {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered in test metrics: %v", r)
		}
	}()

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
