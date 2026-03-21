package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type RequestMetrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

func NewRequestMetrics() *RequestMetrics {
	return &RequestMetrics{
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cart_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "code"},
		),
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "cart_http_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method"},
		),
	}
}

func (rm *RequestMetrics) ReportRequestInfo(methodName string, code string, duration time.Duration) {
	rm.requestsTotal.WithLabelValues(methodName, code).Inc()
	rm.requestDuration.WithLabelValues(methodName).Observe(duration.Seconds())
}
