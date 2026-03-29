package middlewares

import (
	"log/slog"
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

	duration := time.Since(start)
	m.metrics.ReportRequestInfo(pattern, strconv.Itoa(rec.status), duration)
	logRequest(r, pattern, rec.status, duration)
}

func logRequest(r *http.Request, pattern string, status int, duration time.Duration) {
	args := []any{"method", r.Method, "pattern", pattern, "status", status, "duration", duration}
	switch {
	case status >= 500:
		slog.ErrorContext(r.Context(), "request failed", args...)
	case status >= 400:
		slog.WarnContext(r.Context(), "request failed", args...)
	default:
		slog.InfoContext(r.Context(), "request", args...)
	}
}