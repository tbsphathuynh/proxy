package proxy

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/WillKirkmanM/proxy/internal/config"
	"github.com/WillKirkmanM/proxy/internal/loadbalancer"
	"github.com/WillKirkmanM/proxy/internal/middleware"
)

// Server represents the main proxy server instance
// This struct encapsulates all server dependencies using dependency injection pattern
// The composition approach allows for easy testing and component substitution
type Server struct {
    httpServer   *http.Server
    loadBalancer loadbalancer.LoadBalancer
    middleware   []middleware.Middleware
    config       *config.Config
}

// NewServer creates a new proxy server instance using factory pattern
// The factory pattern encapsulates complex initialisation logic and dependency wiring
// This approach promotes loose coupling and makes testing easier
// Time Complexity: O(n) where n is number of backends for load balancer initialisation
// Space Complexity: O(n) for storing backend configurations and middleware chain
func NewServer(cfg *config.Config) (*Server, error) {
    // Create load balancer using factory pattern based on configuration
    // This allows runtime selection of load balancing algorithms
    lb, err := loadbalancer.NewLoadBalancer(cfg.LoadBalance.Algorithm, cfg.LoadBalance.Backends)
    if err != nil {
        return nil, fmt.Errorf("failed to create load balancer: %w", err)
    }

    // Build middleware chain using chain of responsibility pattern
    // Order matters: rate limiting before caching to prevent cache pollution
    middlewares := []middleware.Middleware{
        middleware.NewRateLimiter(cfg.RateLimit),
        middleware.NewCache(cfg.Cache),
        middleware.NewMetrics(), // prometheus metrics
    }

    // Create HTTP server with configured timeouts
    // Timeouts are critical for preventing resource exhaustion attacks
    server := &http.Server{
        Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
        ReadTimeout:  cfg.Server.ReadTimeout,
        WriteTimeout: cfg.Server.WriteTimeout,
        IdleTimeout:  cfg.Server.IdleTimeout,
    }

    return &Server{
        httpServer:   server,
        loadBalancer: lb,
        middleware:   middlewares,
        config:       cfg,
    }, nil
}

// Start begins serving HTTP requests with graceful shutdown support
// Uses context for coordinated shutdown across all components
// Time Complexity: O(1) for startup, O(∞) for request serving until context cancellation
// Space Complexity: O(1) for server state, O(n) for concurrent request handling
func (s *Server) Start(ctx context.Context) error {
    // Set up HTTP handler with middleware chain
    // The handler implements the template method pattern
    s.httpServer.Handler = s.buildHandler()

    // Channel for server errors - prevents blocking on error conditions
    errChan := make(chan error, 1)

    // Start HTTP server in separate goroutine
    // This prevents blocking the main goroutine and allows concurrent shutdown handling
    go func() {
        if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            errChan <- fmt.Errorf("HTTP server error: %w", err)
        }
    }()

    // Start health checking in background
    // Health checks run independently to avoid blocking request processing
    go s.startHealthChecks(ctx)

    // Wait for either error or context cancellation
    // This implements the select pattern for concurrent event handling
    select {
    case err := <-errChan:
        return err
    case <-ctx.Done():
        return ctx.Err()
    }
}

// Shutdown gracefully stops the server and all background processes
// Implements graceful shutdown pattern to prevent data loss and connection drops
// Time Complexity: O(1) for shutdown initiation, depends on active connection count
// Space Complexity: O(1) - no additional memory allocation during shutdown
func (s *Server) Shutdown(ctx context.Context) error {
    // Shutdown HTTP server with context timeout
    // This ensures shutdown completes within reasonable time bounds
    if err := s.httpServer.Shutdown(ctx); err != nil {
        return fmt.Errorf("failed to shutdown HTTP server: %w", err)
    }

    // Additional cleanup for load balancer and other components would go here
    // For now, context cancellation handles background goroutine cleanup

    return nil
}

// buildHandler constructs the HTTP handler with middleware chain
// Implements chain of responsibility pattern for request processing
// Each middleware can modify request/response or short-circuit the chain
// Time Complexity: O(m) where m is number of middleware for chain construction
// Space Complexity: O(m) for middleware chain storage
func (s *Server) buildHandler() http.Handler {
    // Start with the core proxy handler
    // This is the final handler in the chain that performs actual proxying
    var handler http.Handler = http.HandlerFunc(s.proxyHandler)

    // Apply middleware in reverse order to build chain correctly
    // Middleware wrapping creates nested function calls: middleware1(middleware2(handler))
    for i := len(s.middleware) - 1; i >= 0; i-- {
        handler = s.middleware[i].Wrap(handler)
    }

    return handler
}

// proxyHandler performs the core reverse proxy functionality
// This is where actual request forwarding happens using the selected backend
// Time Complexity: O(log n) for backend selection with balanced algorithms
// Space Complexity: O(1) for request processing, O(k) for request/response buffering
func (s *Server) proxyHandler(w http.ResponseWriter, r *http.Request) {
    // Select backend using configured load balancing algorithm
    // Load balancer handles backend health and availability
    backend, err := s.loadBalancer.SelectBackend(r)
    if err != nil {
        http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
        return
    }

    // Create reverse proxy for selected backend
    // Each request gets a fresh proxy instance to avoid state issues
    proxy := NewReverseProxy(backend)
    
    // Forward request to selected backend
    // The reverse proxy handles URL rewriting, header forwarding, and response copying
    proxy.ServeHTTP(w, r)
}

// startHealthChecks begins background health monitoring for all backends
// Health checks run on configurable intervals to detect backend failures
// Uses observer pattern to notify load balancer of backend status changes
// Time Complexity: O(n) per check interval where n is number of backends
// Space Complexity: O(1) for health check state per backend
func (s *Server) startHealthChecks(ctx context.Context) {
    // Create ticker for periodic health checks
    // Ticker ensures consistent check intervals regardless of check duration
    ticker := time.NewTicker(s.config.Health.Interval)
    defer ticker.Stop()

    // Perform initial health check before starting periodic checks
    // This ensures backend status is known at startup
    s.performHealthChecks()

    // Run health checks on configured intervals until context cancellation
    // This implements the observer pattern for backend health monitoring
    for {
        select {
        case <-ticker.C:
            s.performHealthChecks()
        case <-ctx.Done():
            return
        }
    }
}

// performHealthChecks executes health checks for all configured backends
// Each backend is checked concurrently to minimize total check time
// Results are reported to load balancer using observer pattern
// Time Complexity: O(n) where n is number of backends (concurrent execution)
// Space Complexity: O(n) for goroutine stacks during concurrent health checks
func (s *Server) performHealthChecks() {
    // Get all backends from load balancer for health checking
    // This ensures we check all backends regardless of current health status
    backends := s.loadBalancer.GetBackends()

    // Check each backend concurrently to minimize total check time
    // Concurrent checks prevent one slow backend from delaying others
    for _, backend := range backends {
        go func(b loadbalancer.Backend) {
            // Perform HTTP health check with configured timeout
            // Timeout prevents health checks from hanging indefinitely
            healthy := s.checkBackendHealth(b)
            
            // Notify load balancer of backend health status
            // Load balancer updates routing decisions based on health status
            s.loadBalancer.UpdateBackendHealth(b.GetURL(), healthy)
        }(backend)
    }
}

// checkBackendHealth verifies if a specific backend is healthy and responsive
// Uses HTTP GET request to configured health check endpoint
// Timeout ensures health checks don't block other operations
// Time Complexity: O(1) - single HTTP request with bounded timeout
// Space Complexity: O(1) - minimal request/response buffering
func (s *Server) checkBackendHealth(backend loadbalancer.Backend) bool {
    // Create HTTP client with health check timeout
    // Dedicated client prevents interference with proxy requests
    client := &http.Client{
        Timeout: s.config.Health.Timeout,
    }

    // Construct health check URL by appending health path to backend URL
    // This allows backends to implement custom health check endpoints
    healthURL := backend.GetURL() + s.config.Health.Path

    // Perform GET request to health check endpoint
    // GET is used as it's idempotent and widely supported for health checks
    resp, err := client.Get(healthURL)
    if err != nil {
        return false
    }
    defer resp.Body.Close()

    // Consider backend healthy if HTTP status is 2xx
    // This is a common convention for health check endpoints
    return resp.StatusCode >= 200 && resp.StatusCode < 300
}