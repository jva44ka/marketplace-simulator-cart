package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type DbMetrics struct {
	requestsTotal *prometheus.CounterVec
}

func NewDbMetrics() *DbMetrics {
	return &DbMetrics{
		requestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "cart_db_requests_total",
			Help: "Total number of cart DB requests",
		}, []string{"method", "status"}),
	}
}

func (m *DbMetrics) ReportRequest(method string, status string) {
	m.requestsTotal.WithLabelValues(method, status).Inc()
}
