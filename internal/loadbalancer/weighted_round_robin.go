package loadbalancer

import (
	"errors"
	"net/http"
	"sync"
)

// WeightedRoundRobinBalancer implements weighted round-robin load balancing
// Distributes requests based on backend weights with smooth weighted round-robin algorithm
// Prevents bursts of requests to high-weight backends by distributing smoothly
// Time Complexity: O(n) for finding next backend with highest current weight
// Space Complexity: O(n) for storing backend references and current weights
type WeightedRoundRobinBalancer struct {
    backends       []Backend // List of all configured backends
    currentWeights []int     // Current weights for smooth distribution
    mutex          sync.RWMutex // Protects backends and weights during updates
}

// NewWeightedRoundRobinBalancer creates new weighted round-robin load balancer
// Initializes current weights to zero for smooth weighted algorithm
// Time Complexity: O(n) for initialising current weights array
// Space Complexity: O(n) for storing backends and current weights
func NewWeightedRoundRobinBalancer(backends []Backend) *WeightedRoundRobinBalancer {
    return &WeightedRoundRobinBalancer{
        backends:       backends,
        currentWeights: make([]int, len(backends)),
    }
}

// SelectBackend chooses next backend using smooth weighted round-robin algorithm
// Algorithm ensures even distribution while respecting weights
// Prevents weight-based clustering by smooth weight adjustment
// Time Complexity: O(n) for finding backend with highest current weight
// Space Complexity: O(1) - no additional allocations during selection
func (wrr *WeightedRoundRobinBalancer) SelectBackend(req *http.Request) (Backend, error) {
    wrr.mutex.Lock()
    defer wrr.mutex.Unlock()

    if len(wrr.backends) == 0 {
        return nil, errors.New("no backends available")
    }

    // Find healthy backend with highest current weight
    selectedIndex := -1
    maxCurrentWeight := -1

    for i, backend := range wrr.backends {
        if !backend.IsHealthy() {
            continue // Skip unhealthy backends
        }

        // Add backend weight to current weight for smooth distribution
        wrr.currentWeights[i] += backend.GetWeight()

        // Select backend with highest current weight
        if wrr.currentWeights[i] > maxCurrentWeight {
            selectedIndex = i
            maxCurrentWeight = wrr.currentWeights[i]
        }
    }

    if selectedIndex == -1 {
        return nil, errors.New("no healthy backends available")
    }

    // Calculate total weight of all healthy backends
    totalWeight := 0
    for _, backend := range wrr.backends {
        if backend.IsHealthy() {
            totalWeight += backend.GetWeight()
        }
    }

    // Subtract total weight from selected backend's current weight
    // This ensures smooth distribution over time
    wrr.currentWeights[selectedIndex] -= totalWeight

    return wrr.backends[selectedIndex], nil
}

// UpdateBackendHealth updates health status for specified backend URL
// Uses linear search to find backend by URL
// Write lock ensures thread safety during health status updates
// Time Complexity: O(n) for linear search through backend list
// Space Complexity: O(1) - no additional allocations
func (wrr *WeightedRoundRobinBalancer) UpdateBackendHealth(url string, healthy bool) {
    wrr.mutex.Lock()
    defer wrr.mutex.Unlock()

    for _, backend := range wrr.backends {
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
func (wrr *WeightedRoundRobinBalancer) GetBackends() []Backend {
    wrr.mutex.RLock()
    defer wrr.mutex.RUnlock()

    backends := make([]Backend, len(wrr.backends))
    copy(backends, wrr.backends)
    return backends
}

// UpdateBackendWeight updates weight for specified backend URL
// Allows dynamic weight adjustment for traffic shaping
// Write lock ensures thread safety during weight updates
// Time Complexity: O(n) for linear search through backend list
// Space Complexity: O(1) - no additional allocations
func (wrr *WeightedRoundRobinBalancer) UpdateBackendWeight(url string, weight int) {
    wrr.mutex.Lock()
    defer wrr.mutex.Unlock()

    for _, backend := range wrr.backends {
        if backend.GetURL() == url {
            backend.SetWeight(weight)
            return
        }
    }
}