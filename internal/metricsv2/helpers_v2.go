package metricsv2

import (
	`github.com/prometheus/client_golang/prometheus`
	`github.com/sirupsen/logrus`
)

//
// COPY OF THE internal/metrics/operations_collector.go for test porpuses, will be refactored
//

func collect(ch chan<- prometheus.Metric, desc *prometheus.Desc, value int, labelValues ...string) {
	m, err := prometheus.NewConstMetric(
		desc,
		prometheus.GaugeValue,
		float64(value),
		labelValues...)
	
	if err != nil {
		logrus.Errorf("unable to register metric %s", err.Error())
		return
	}
	ch <- m
}