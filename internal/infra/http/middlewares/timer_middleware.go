package middlewares

import (
	"net/http"
	"strconv"
	"time"
)

type Metrics interface {
	ReportRequestInfo(methodName string, code string, duration time.Duration)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

type TimerMiddleware struct {
	h       http.Handler
	metrics Metrics
}

func NewTimerMiddleware(h http.Handler, m Metrics) http.Handler {
	return &TimerMiddleware{h: h, metrics: m}
}

func (m *TimerMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

	m.h.ServeHTTP(rec, r)

	pattern := r.Pattern
	if pattern == "" {
		pattern = r.URL.Path
	}

	m.metrics.ReportRequestInfo(pattern, strconv.Itoa(rec.status), time.Since(start))
}