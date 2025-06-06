package middleware

import "net/http"

// Middleware defines the interface for HTTP middleware components
// This interface implements the decorator pattern for request/response processing
// Middleware can modify requests, responses, or short-circuit the request chain
type Middleware interface {
    // Wrap decorates an HTTP handler with additional functionality
    // Returns a new handler that executes middleware logic before/after the wrapped handler
    // This implements the chain of responsibility pattern for request processing
    // Time Complexity: O(1) for wrapping, varies by middleware implementation
    // Space Complexity: O(1) for handler wrapping, varies by middleware state
    Wrap(next http.Handler) http.Handler
}