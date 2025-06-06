package loadbalancer

import (
	"errors"
	"net/http"
	"sync"
)

// RoundRobinBalancer implements round-robin load balancing algorithm
// This algorithm distributes requests evenly across all healthy backends
// Uses atomic counter to ensure thread-safe operation across concurrent requests
// Time Complexity: O(n) worst case for finding healthy backend, O(1) average case
// Space Complexity: O(n) for storing backend references
type RoundRobinBalancer struct {
    backends []Backend    // List of all configured backends
    current  int          // Current position in round-robin cycle
    mutex    sync.RWMutex // Protects current counter and backends slice
}

// NewRoundRobinBalancer creates a new round-robin load balancer
// Initializes with all provided backends in healthy state
// Current position starts at 0 for deterministic behavior
// Time Complexity: O(1) - simple initialisation
// Space Complexity: O(n) for storing backend slice
func NewRoundRobinBalancer(backends []Backend) *RoundRobinBalancer {
    return &RoundRobinBalancer{
        backends: backends,
        current:  0,
    }
}

// SelectBackend chooses next backend using round-robin algorithm
// Skips unhealthy backends and wraps around when reaching end of list
// Thread-safe implementation using mutex for current position protection
// Time Complexity: O(n) worst case if all backends unhealthy, O(1) typical case
// Space Complexity: O(1) - no additional allocations during selection
func (rb *RoundRobinBalancer) SelectBackend(req *http.Request) (Backend, error) {
    rb.mutex.Lock()
    defer rb.mutex.Unlock()

    if len(rb.backends) == 0 {
        return nil, errors.New("no backends available")
    }

    // Try each backend starting from current position
    // This ensures even distribution and handles unhealthy backends gracefully
    start := rb.current
    for {
        backend := rb.backends[rb.current]
        
        // Move to next backend for subsequent requests
        // Modulo operation ensures wrap-around at end of list
        rb.current = (rb.current + 1) % len(rb.backends)

        // Return backend if healthy, otherwise continue searching
        if backend.IsHealthy() {
            return backend, nil
        }

        // If we've checked all backends without finding healthy one
        // This prevents infinite loop when all backends are unhealthy
        if rb.current == start {
            return nil, errors.New("no healthy backends available")
        }
    }
}

// UpdateBackendHealth updates health status for specified backend URL
// Uses linear search to find backend by URL (could be optimized with map)
// Write lock ensures thread safety during health status updates
// Time Complexity: O(n) for linear search through backend list
// Space Complexity: O(1) - no additional allocations
func (rb *RoundRobinBalancer) UpdateBackendHealth(url string, healthy bool) {
    rb.mutex.Lock()
    defer rb.mutex.Unlock()

    // Linear search for backend with matching URL
    // Could be optimized with URL->Backend map for O(1) lookup
    for _, backend := range rb.backends {
        if backend.GetURL() == url {
            backend.SetHealthy(healthy)
            return
        }
    }
}

// GetBackends returns copy of all backends for health checking
// Uses read lock to allow concurrent access during health checks
// Returns slice copy to prevent external modification of internal state
// Time Complexity: O(n) for slice copy
// Space Complexity: O(n) for copied slice
func (rb *RoundRobinBalancer) GetBackends() []Backend {
    rb.mutex.RLock()
    defer rb.mutex.RUnlock()

    // Return copy to prevent external modifications
    backends := make([]Backend, len(rb.backends))
    copy(backends, rb.backends)
    return backends
}