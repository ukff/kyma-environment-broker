package debug

import (
	"testing"

	"github.com/gorilla/mux"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/metrics"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestDebug(t *testing.T) {
	// create server
	router := mux.NewRouter()
	db := storage.NewMemoryStorage()
	prometheus.MustRegister(metrics.NewOperationsCollector(db.Operations()))
	router.Handle("/metrics", promhttp.Handler())

	db.Operations().InsertOperation(internal.Operation{
		ID: "1",
	})
}
