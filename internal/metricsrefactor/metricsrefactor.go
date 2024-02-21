package metricsrefactor

import (
	"context"
	"sync"

	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	metric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lj_operations_test",
		Help: "The total number of processed events",
	}, []string{"operation_id", "instance_id", "plan_id", "type", "state"})
)

var m sync.Mutex

func log(msg string, error bool) {
	m.Lock()
	defer m.Unlock()
	if error {
		logrus.Error(msg)
		return
	}
	logrus.Info(msg)
}

func OperationSucceededHandler(ctx context.Context, ev interface{}) error {
	log("OperationSucceededHandler called", false)
	op, ok := ev.(process.OperationSucceeded)
	if !ok {
		log("expected process.OperationSucceeded but got %+v", true)
		return nil
	}

	log("setting metric...", false)
	m := metric.WithLabelValues(op.Operation.ID, op.Operation.InstanceID, string(op.Operation.Type), string(op.Operation.State))
	m.Set(float64(1))
	log("metric set", false)

	return nil
}
