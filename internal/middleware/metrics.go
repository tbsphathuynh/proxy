package middleware

import (
	"net/http"

	"github.com/WillKirkmanM/proxy/internal/metrics"
)

// metricsMiddleware adapts Prometheus metrics into Middleware
type metricsMiddleware struct {
    m *metrics.Metrics
}

// NewMetrics constructs the metrics middleware
func NewMetrics() Middleware {
    return &metricsMiddleware{m: metrics.NewMetrics()}
}

// Wrap instruments each request with Prometheus metrics
func (mm *metricsMiddleware) Wrap(next http.Handler) http.Handler {
    // label "proxy" for top-level metrics
    return mm.m.MetricsMiddleware("proxy")(next)
}