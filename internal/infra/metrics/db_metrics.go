package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type DbMetrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

func NewDbMetrics() *DbMetrics {
	return &DbMetrics{
		requestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:        "db_requests_total",
			Help:        "Total number of DB requests",
			ConstLabels: prometheus.Labels{"service": "cart"},
		}, []string{"method", "status"}),
		requestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "db_request_duration_seconds",
			Help:        "Duration of DB requests in seconds",
			ConstLabels: prometheus.Labels{"service": "cart"},
			Buckets:     []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		}, []string{"method", "status"}),
	}
}

func (m *DbMetrics) ReportRequest(method, status string, duration time.Duration) {
	m.requestsTotal.WithLabelValues(method, status).Inc()
	m.requestDuration.WithLabelValues(method, status).Observe(duration.Seconds())
}
