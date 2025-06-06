package proxy

import (
    "net/http"
    "net/http/httputil"
    "net/url"

    "github.com/WillKirkmanM/proxy/internal/loadbalancer"
)

// NewReverseProxy creates a new reverse proxy for the specified backend
// This function wraps Go's standard httputil.ReverseProxy with custom logic
// The proxy handles URL rewriting, header modification, and error handling
// Time Complexity: O(1) - constant time proxy creation
// Space Complexity: O(1) - single proxy instance per backend
func NewReverseProxy(backend loadbalancer.Backend) *httputil.ReverseProxy {
    // Parse backend URL for proxy configuration
    // URL parsing is required for proper request forwarding
    target, _ := url.Parse(backend.GetURL())

    // Create reverse proxy with custom director function
    // Director function modifies outgoing requests before forwarding
    proxy := httputil.NewSingleHostReverseProxy(target)

    // Customize request director for additional processing
    // This allows header manipulation, logging, and request modification
    originalDirector := proxy.Director
    proxy.Director = func(req *http.Request) {
        // Apply original director first to set basic proxy headers
        originalDirector(req)
        
        // Add custom headers for backend identification
        // This helps backends identify requests coming through the proxy
        req.Header.Set("X-Forwarded-By", "go-reverse-proxy")
        req.Header.Set("X-Backend-URL", backend.GetURL())
    }

    // Customize error handler for better error reporting
    // Default error handler may not provide sufficient debugging information
    proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
        // Log error for debugging purposes
        // In production, this should use structured logging
        
        // Return appropriate HTTP error status
        // 502 Bad Gateway indicates upstream server error
        http.Error(w, "Backend server error", http.StatusBadGateway)
    }

    return proxy
}