package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type BusinessMetrics struct {
	checkoutsTotal     *prometheus.CounterVec
	checkoutValueTotal prometheus.Counter
}

func NewBusinessMetrics() *BusinessMetrics {
	return &BusinessMetrics{
		checkoutsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:        "checkouts_total",
			Help:        "Total number of checkout attempts",
			ConstLabels: prometheus.Labels{"service": "cart"},
		}, []string{"status", "reason"}),
		checkoutValueTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name:        "checkout_value_total",
			Help:        "Cumulative sum of total_price for successful checkouts",
			ConstLabels: prometheus.Labels{"service": "cart"},
		}),
	}
}

func (m *BusinessMetrics) RecordSuccess(totalPrice float64) {
	m.checkoutsTotal.WithLabelValues("success", "").Inc()
	m.checkoutValueTotal.Add(totalPrice)
}

func (m *BusinessMetrics) RecordFailure(reason string) {
	m.checkoutsTotal.WithLabelValues("failed", reason).Inc()
}
