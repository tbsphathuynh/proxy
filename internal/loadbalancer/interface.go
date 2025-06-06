package loadbalancer

import (
	"net/http"
	"net/url"
	"sync/atomic"
)

// Backend represents a backend server interface
// Encapsulates server state and operations for load balancing
// Allows different backend implementations with consistent interface
type Backend interface {
    GetURL() string                                      // Returns backend server URL
    IsHealthy() bool                                     // Returns current health status
    SetHealthy(bool)                                     // Updates health status
    ServeHTTP(http.ResponseWriter, *http.Request)        // Handles HTTP requests
    GetConnections() int64                               // Returns current connection count
    IncrementConnections()                               // Increments active connections
    DecrementConnections()                               // Decrements active connections
    GetWeight() int                                      // Returns backend weight for weighted algorithms
    SetWeight(int)                                       // Sets backend weight
}

// LoadBalancer defines interface for load balancing algorithms
// Abstracts load balancing strategy to support different algorithms
// Enables easy swapping between round-robin, weighted, least-connections, etc.
type LoadBalancer interface {
    SelectBackend(*http.Request) (Backend, error) // Selects backend for request
    UpdateBackendHealth(string, bool)             // Updates backend health status
    GetBackends() []Backend                       // Returns all backends for monitoring
}

// HTTPBackend implements Backend interface for HTTP servers
// Provides concrete implementation for proxying HTTP requests
// Maintains health status, connection count, and weight for load balancing decisions
type HTTPBackend struct {
    url         *url.URL      // Parsed backend server URL
    healthy     bool          // Current health status
    client      *http.Client  // HTTP client for request forwarding
    connections int64         // Active connection count (atomic for thread safety)
    weight      int           // Backend weight for weighted load balancing
}

// NewHTTPBackend creates new HTTP backend with specified URL and weight
// Initializes with healthy status and default HTTP client
// Default weight of 1 provides equal distribution for weighted algorithms
// Time Complexity: O(1) - simple struct initialisation
// Space Complexity: O(1) - fixed size backend structure
func NewHTTPBackend(backendURL string, weight int) (*HTTPBackend, error) {
    url, err := url.Parse(backendURL)
    if err != nil {
        return nil, err
    }

    if weight <= 0 {
        weight = 1 // Default weight for invalid values
    }

    return &HTTPBackend{
        url:         url,
        healthy:     true,
        client:      &http.Client{},
        connections: 0,
        weight:      weight,
    }, nil
}

// GetURL returns backend server URL string
// Used for backend identification and health check routing
// Time Complexity: O(1) - returns cached URL string
// Space Complexity: O(1) - no additional allocations
func (b *HTTPBackend) GetURL() string {
    return b.url.String()
}

// IsHealthy returns current backend health status
// Used by load balancer to determine routing eligibility
// Time Complexity: O(1) - simple boolean access
// Space Complexity: O(1) - no allocations
func (b *HTTPBackend) IsHealthy() bool {
    return b.healthy
}

// SetHealthy updates backend health status
// Called by health check system to mark backends as up/down
// Time Complexity: O(1) - simple boolean assignment
// Space Complexity: O(1) - no allocations
func (b *HTTPBackend) SetHealthy(healthy bool) {
    b.healthy = healthy
}

// GetConnections returns current active connection count
// Used by least connections algorithm for load balancing decisions
// Atomic load ensures thread-safe access in concurrent environment
// Time Complexity: O(1) - atomic memory access
// Space Complexity: O(1) - no allocations
func (b *HTTPBackend) GetConnections() int64 {
    return atomic.LoadInt64(&b.connections)
}

// IncrementConnections atomically increases active connection count
// Called when new request is routed to this backend
// Atomic operation ensures thread safety without mutex overhead
// Time Complexity: O(1) - atomic memory operation
// Space Complexity: O(1) - no allocations
func (b *HTTPBackend) IncrementConnections() {
    atomic.AddInt64(&b.connections, 1)
}

// DecrementConnections atomically decreases active connection count
// Called when request completes or connection closes
// Atomic operation ensures accurate connection tracking
// Time Complexity: O(1) - atomic memory operation
// Space Complexity: O(1) - no allocations
func (b *HTTPBackend) DecrementConnections() {
    atomic.AddInt64(&b.connections, -1)
}

// GetWeight returns backend weight for weighted load balancing
// Higher weights receive proportionally more traffic
// Used by weighted round-robin and weighted least connections algorithms
// Time Complexity: O(1) - simple integer access
// Space Complexity: O(1) - no allocations
func (b *HTTPBackend) GetWeight() int {
    return b.weight
}

// SetWeight updates backend weight for weighted load balancing
// Allows dynamic weight adjustment for traffic shaping
// Weight changes take effect on next load balancing decision
// Time Complexity: O(1) - simple integer assignment
// Space Complexity: O(1) - no allocations
func (b *HTTPBackend) SetWeight(weight int) {
    if weight <= 0 {
        weight = 1 // Ensure positive weight
    }
    b.weight = weight
}

// ServeHTTP forwards request to backend server with connection tracking
// Implements reverse proxy functionality with error handling
// Updates request URL to point to backend server and tracks connections
// Time Complexity: O(1) for setup, O(n) for request/response transfer
// Space Complexity: O(n) for request/response buffering
func (b *HTTPBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Increment connection count for load balancing
    b.IncrementConnections()
    defer b.DecrementConnections()

    // Create new request with backend URL
    r.URL.Scheme = b.url.Scheme
    r.URL.Host = b.url.Host
    r.Host = b.url.Host

    // Forward request to backend
    resp, err := b.client.Do(r)
    if err != nil {
        http.Error(w, "Backend unavailable", http.StatusBadGateway)
        return
    }
    defer resp.Body.Close()

    // Copy response headers
    for key, values := range resp.Header {
        for _, value := range values {
            w.Header().Add(key, value)
        }
    }

    // Set status code and copy body
    w.WriteHeader(resp.StatusCode)
    
    // Stream response body to client
    buffer := make([]byte, 32*1024) // 32KB buffer for streaming
    for {
        n, err := resp.Body.Read(buffer)
        if n > 0 {
            w.Write(buffer[:n])
        }
        if err != nil {
            break
        }
    }
}