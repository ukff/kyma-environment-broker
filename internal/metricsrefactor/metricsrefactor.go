package metricsrefactor

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	metric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lj_operations_test",
		Help: "The total number of processed events",
	}, []string{"operation_id", "instance_id", "type", "state"})
)

type operation struct {
	ID         string
	InstanceID string
	Type       string
	State      string
}

var logMu sync.Mutex

func log(msg string, error bool) {
	logMu.Lock()
	defer logMu.Unlock()
	if error {
		logrus.Errorf("@debug (error) -> %s", msg)
		return
	}
	logrus.Infof("@debug (info) -> %s", msg)
}

func getOperationDataFromGenericEvent(event interface{}) *operation {
	op := event.(map[string]interface{})["Operation"].(map[string]interface{})
	return &operation{
		ID:         op["ID"].(string),
		InstanceID: op["InstanceID"].(string),
		Type:       op["Type"].(string),
		State:      op["State"].(string),
	}
}

// Tests
func OperationStepProcessedHandler(ctx context.Context, ev interface{}) error {
	// dont dont anything since this are only test metrics
	defer func() {
		if r := recover(); r != nil {
			log("recovered panic in f", true)
		}
	}()
	op := getOperationDataFromGenericEvent(ev)
	m := metric.WithLabelValues(op.ID, op.InstanceID, op.Type, op.State)
	m.Set(float64(1))
	return nil
}
