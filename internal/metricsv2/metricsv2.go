package metricsv2

import (
	"context"

	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	provisionedInstances = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "keb",
		Subsystem: "test",
		Name:      "provisioned",
		Help:      "counter of successfully provisioned instances",
	})
)

// dont fail anything since it is just test function which is used for gathering informations before development
func Handler(ctx context.Context, ev interface{}) error {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("recovered in test metrics: %v", r)
		}
	}()

	switch ev.(type) {
	case process.ProvisioningSucceeded:
		provisionedInstances.Inc()
	default:
		logrus.Error("ev type not supported")
	}

	return nil
}
