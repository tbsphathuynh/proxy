package metrics

import (
	"net/http"
	"strconv"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics provides Prometheus metrics collection for proxy server
// Tracks request counts, durations, and backend health for monitoring
// Enables observability and performance analysis through metrics
type Metrics struct {
    requestsTotal    *prometheus.CounterVec   // Total requests by method and status
    requestDuration  *prometheus.HistogramVec // Request duration distribution
    backendHealth    *prometheus.GaugeVec     // Backend health status (0/1)
    activeConnections prometheus.Gauge         // Current active connections
}

// NewMetrics creates new metrics collector with Prometheus instruments
// Registers all metrics with default registry for HTTP exposition
// Time Complexity: O(1) - metric registration
// Space Complexity: O(1) - fixed metric storage
func NewMetrics() *Metrics {
    m := &Metrics{
        requestsTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "proxy_requests_total",
                Help: "Total number of HTTP requests processed",
            },
            []string{"method", "status_code", "backend"},
        ),
        requestDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "proxy_request_duration_seconds",
                Help:    "HTTP request duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"method", "backend"},
        ),
        backendHealth: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "proxy_backend_health",
                Help: "Backend health status (1=healthy, 0=unhealthy)",
            },
            []string{"backend_url"},
        ),
        activeConnections: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Name: "proxy_active_connections",
                Help: "Number of active connections",
            },
        ),
    }

    // Register metrics with Prometheus
    prometheus.MustRegister(m.requestsTotal)
    prometheus.MustRegister(m.requestDuration)
    prometheus.MustRegister(m.backendHealth)
    prometheus.MustRegister(m.activeConnections)

    return m
}

// RecordRequest records HTTP request metrics including duration and status
// Called by middleware to track request statistics
// Time Complexity: O(1) - metric recording
// Space Complexity: O(1) - no additional allocations
func (m *Metrics) RecordRequest(method, statusCode, backend string, duration time.Duration) {
    m.requestsTotal.WithLabelValues(method, statusCode, backend).Inc()
    m.requestDuration.WithLabelValues(method, backend).Observe(duration.Seconds())
}

// UpdateBackendHealth updates health metric for specified backend
// Called by health check system to track backend availability
// Time Complexity: O(1) - metric update
// Space Complexity: O(1) - no additional allocations
func (m *Metrics) UpdateBackendHealth(backendURL string, healthy bool) {
    value := 0.0
    if healthy {
        value = 1.0
    }
    m.backendHealth.WithLabelValues(backendURL).Set(value)
}

// IncrementConnections increments active connection count
// Called when new connection is established
// Time Complexity: O(1) - atomic increment
// Space Complexity: O(1) - no allocations
func (m *Metrics) IncrementConnections() {
    m.activeConnections.Inc()
}

// DecrementConnections decrements active connection count
// Called when connection is closed
// Time Complexity: O(1) - atomic decrement
// Space Complexity: O(1) - no allocations
func (m *Metrics) DecrementConnections() {
    m.activeConnections.Dec()
}

// Handler returns HTTP handler for Prometheus metrics exposition
// Enables metrics scraping by monitoring systems
// Time Complexity: O(1) - returns existing handler
// Space Complexity: O(1) - no additional allocations
func (m *Metrics) Handler() http.Handler {
    return promhttp.Handler()
}

// MetricsMiddleware creates middleware for automatic request metrics collection
// Wraps HTTP handlers to collect timing and status metrics
// Time Complexity: O(1) per request for metric recording
// Space Complexity: O(1) - no additional allocations per request
func (m *Metrics) MetricsMiddleware(backend string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // Increment active connections
            m.IncrementConnections()
            defer m.DecrementConnections()

            // Wrap response writer to capture status code
            wrapper := &statusRecorder{ResponseWriter: w, statusCode: 200}
            
            // Process request
            next.ServeHTTP(wrapper, r)
            
            // Record metrics
            duration := time.Since(start)
            m.RecordRequest(
                r.Method,
                strconv.Itoa(wrapper.statusCode),
                backend,
                duration,
            )
        })
    }
}

// statusRecorder wraps ResponseWriter to capture HTTP status codes
// Used by metrics middleware to record response status
type statusRecorder struct {
    http.ResponseWriter
    statusCode int
}

// WriteHeader captures status code for metrics
func (sr *statusRecorder) WriteHeader(code int) {
    sr.statusCode = code
    sr.ResponseWriter.WriteHeader(code)
}