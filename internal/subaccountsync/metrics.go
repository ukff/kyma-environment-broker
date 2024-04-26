package subaccountsync

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	queue       prometheus.Gauge
	queueOps    *prometheus.CounterVec
	cisRequests *prometheus.CounterVec
	states      *prometheus.GaugeVec
	informer    *prometheus.CounterVec
	timeInQueue prometheus.Gauge
}

func NewMetrics(reg prometheus.Registerer, namespace string) *Metrics {
	m := &Metrics{
		states: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "in_memory_states",
			Help:      "Information about in-memory states.",
		}, []string{"type"}),
		queueOps: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "priority_queue_ops",
			Help:      "Priority queue operations.",
		}, []string{"operation"}),
		cisRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cis_requests",
			Help:      "CIS requests.",
		}, []string{"endpoint"}),
		informer: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "informer",
			Help:      "Informer stats.",
		}, []string{"event"}),
		queue: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "priority_queue_size",
			Help:      "Queue size.",
		}),
		timeInQueue: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "time_in_queue",
			Help:      "Time spent in queue.",
		}),
	}
	reg.MustRegister(m.queue, m.queueOps, m.states, m.informer, m.cisRequests)
	return m
}
