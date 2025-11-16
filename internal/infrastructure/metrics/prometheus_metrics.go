package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusMetrics struct {
	httpRequests *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec
}

func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		httpRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "pr_reviewer_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		httpDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "pr_reviewer_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
	}
}

func (m *PrometheusMetrics) IncHTTPRequests(method, path string, statusCode int) {
	m.httpRequests.WithLabelValues(method, path, string(rune(statusCode))).Inc()
}

func (m *PrometheusMetrics) ObserveHTTPDuration(method, path string, duration float64) {
	m.httpDuration.WithLabelValues(method, path).Observe(duration)
}
