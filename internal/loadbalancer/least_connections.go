package loadbalancer

import (
	"errors"
	"net/http"
	"sync"
)

// LeastConnectionsBalancer implements least connections load balancing algorithm
// Routes requests to backend with fewest active connections
// Provides better distribution for long-running connections than round-robin
// Time Complexity: O(n) for finding backend with minimum connections
// Space Complexity: O(n) for storing backend references
type LeastConnectionsBalancer struct {
    backends []Backend    // List of all configured backends
    mutex    sync.RWMutex // Protects backends slice during updates
}

// NewLeastConnectionsBalancer creates new least connections load balancer
// Initializes with provided backends, all assumed healthy initially
// Time Complexity: O(1) - simple initialisation
// Space Complexity: O(n) for storing backend slice
func NewLeastConnectionsBalancer(backends []Backend) *LeastConnectionsBalancer {
    return &LeastConnectionsBalancer{
        backends: backends,
    }
}

// SelectBackend chooses backend with fewest active connections
// Skips unhealthy backends and finds minimum connection count among healthy ones
// In case of tie, returns first backend found with minimum connections
// Time Complexity: O(n) for scanning all backends to find minimum
// Space Complexity: O(1) - no additional allocations during selection
func (lc *LeastConnectionsBalancer) SelectBackend(req *http.Request) (Backend, error) {
    lc.mutex.RLock()
    defer lc.mutex.RUnlock()

    if len(lc.backends) == 0 {
        return nil, errors.New("no backends available")
    }

    var selectedBackend Backend
    minConnections := int64(-1) // Use -1 to handle first backend selection

    // Find healthy backend with minimum active connections
    // Linear scan is acceptable for typical backend counts (< 100)
    for _, backend := range lc.backends {
        if !backend.IsHealthy() {
            continue // Skip unhealthy backends
        }

        connections := backend.GetConnections()
        
        // Select backend if it has fewer connections or is first healthy backend
        if minConnections == -1 || connections < minConnections {
            selectedBackend = backend
            minConnections = connections
        }
    }

    if selectedBackend == nil {
        return nil, errors.New("no healthy backends available")
    }

    return selectedBackend, nil
}

// UpdateBackendHealth updates health status for specified backend URL
// Uses linear search to find backend by URL
// Write lock ensures thread safety during health status updates
// Time Complexity: O(n) for linear search through backend list
// Space Complexity: O(1) - no additional allocations
func (lc *LeastConnectionsBalancer) UpdateBackendHealth(url string, healthy bool) {
    lc.mutex.Lock()
    defer lc.mutex.Unlock()

    for _, backend := range lc.backends {
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
func (lc *LeastConnectionsBalancer) GetBackends() []Backend {
    lc.mutex.RLock()
    defer lc.mutex.RUnlock()

    backends := make([]Backend, len(lc.backends))
    copy(backends, lc.backends)
    return backends
}