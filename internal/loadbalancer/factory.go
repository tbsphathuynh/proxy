package loadbalancer

import (
	"fmt"
	"strings"

	"github.com/WillKirkmanM/proxy/internal/config"
)

// LoadBalancerType represents different load balancing algorithms
// Enables type-safe selection of load balancing strategies
type LoadBalancerType string

const (
    RoundRobin         LoadBalancerType = "round-robin"
    LeastConnections   LoadBalancerType = "least-connections"
    WeightedRoundRobin LoadBalancerType = "weighted-round-robin"
)

// BackendConfig represents backend server configuration
// Includes URL and optional weight for weighted algorithms
type BackendConfig struct {
    URL    string `yaml:"url" json:"url"`
    Weight int    `yaml:"weight" json:"weight" default:"1"`
}

// NewLoadBalancer creates load balancer instance using factory pattern
// Supports multiple algorithms through strategy pattern implementation
// Factory pattern encapsulates creation logic and enables runtime algorithm selection
// Time Complexity: O(n) where n is number of backends for initialisation
// Space Complexity: O(n) for storing backend configurations
func NewLoadBalancer(algorithm string, backendConfigs []config.BackendConfig) (LoadBalancer, error) {
    if len(backendConfigs) == 0 {
        return nil, fmt.Errorf("no backends configured")
    }

    // Parse backend configurations and create Backend instances
    backends := make([]Backend, len(backendConfigs))
    for i, cfg := range backendConfigs {
        weight := cfg.Weight
        if weight <= 0 {
            weight = 1 // Default weight for invalid values
        }

        backend, err := NewHTTPBackend(cfg.URL, weight)
        if err != nil {
            return nil, fmt.Errorf("failed to create backend %s: %w", cfg.URL, err)
        }
        backends[i] = backend
    }

    // Create load balancer based on algorithm using strategy pattern
    switch LoadBalancerType(strings.ToLower(algorithm)) {
    case RoundRobin:
        return NewRoundRobinBalancer(backends), nil
    case LeastConnections:
        return NewLeastConnectionsBalancer(backends), nil
    case WeightedRoundRobin:
        return NewWeightedRoundRobinBalancer(backends), nil
    default:
        return nil, fmt.Errorf("unsupported load balancing algorithm: %s", algorithm)
    }
}

// GetSupportedAlgorithms returns list of supported load balancing algorithms
// Used for configuration validation and documentation
// Time Complexity: O(1) - returns static slice
// Space Complexity: O(1) - static data
func GetSupportedAlgorithms() []string {
    return []string{
        string(RoundRobin),
        string(LeastConnections),
        string(WeightedRoundRobin),
    }
}