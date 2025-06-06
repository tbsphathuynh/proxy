package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/WillKirkmanM/proxy/internal/config"
)

// TestCacheHit verifies cache returns stored response on subsequent requests
// Ensures cache reduces backend load by serving from memory
func TestCacheHit(t *testing.T) {
    cache := NewCache(config.CacheConfig{
        MaxSize: 10,
        TTL:     time.Minute,
    })

    // Create test handler that tracks call count
    callCount := 0
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        callCount++
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("test response"))
    })

    cachedHandler := cache.Wrap(handler)

    // First request - should hit backend
    req1 := httptest.NewRequest("GET", "/test", nil)
    w1 := httptest.NewRecorder()
    cachedHandler.ServeHTTP(w1, req1)

    if callCount != 1 {
        t.Errorf("Expected 1 backend call, got %d", callCount)
    }

    // Second request - should hit cache
    req2 := httptest.NewRequest("GET", "/test", nil)
    w2 := httptest.NewRecorder()
    cachedHandler.ServeHTTP(w2, req2)

    if callCount != 1 {
        t.Errorf("Expected 1 backend call after cache hit, got %d", callCount)
    }

    if w2.Header().Get("X-Cache-Status") != "HIT" {
        t.Error("Expected cache hit header")
    }
}

// TestCacheExpiry verifies expired entries are not served
// Ensures stale data is not returned to clients
func TestCacheExpiry(t *testing.T) {
    cache := NewCache(config.CacheConfig{
        MaxSize: 10,
        TTL:     time.Millisecond,
    })

    callCount := 0
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        callCount++
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("test response"))
    })

    cachedHandler := cache.Wrap(handler)

    // First request
    req1 := httptest.NewRequest("GET", "/test", nil)
    w1 := httptest.NewRecorder()
    cachedHandler.ServeHTTP(w1, req1)

    // Wait for expiry
    time.Sleep(time.Millisecond * 2)

    // Second request after expiry - should hit backend
    req2 := httptest.NewRequest("GET", "/test", nil)
    w2 := httptest.NewRecorder()
    cachedHandler.ServeHTTP(w2, req2)

    if callCount != 2 {
        t.Errorf("Expected 2 backend calls after expiry, got %d", callCount)
    }
}

// TestCacheLRUEviction verifies least recently used entries are evicted
// Ensures cache respects size limits and maintains performance
func TestCacheLRUEviction(t *testing.T) {
    cache := NewCache(config.CacheConfig{
        MaxSize: 2,
        TTL:     time.Minute,
    })

    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("response for " + r.URL.Path))
    })

    cachedHandler := cache.Wrap(handler)

    // Fill cache to capacity
    req1 := httptest.NewRequest("GET", "/test1", nil)
    cachedHandler.ServeHTTP(httptest.NewRecorder(), req1)

    req2 := httptest.NewRequest("GET", "/test2", nil)
    cachedHandler.ServeHTTP(httptest.NewRecorder(), req2)

    // Add third entry - should evict first
    req3 := httptest.NewRequest("GET", "/test3", nil)
    cachedHandler.ServeHTTP(httptest.NewRecorder(), req3)

    // Verify first entry was evicted
    if cache.currentSize != 2 {
        t.Errorf("Expected cache size 2, got %d", cache.currentSize)
    }

    // First entry should no longer be cached
    cacheKey1 := cache.generateCacheKey(req1)
    if cache.get(cacheKey1) != nil {
        t.Error("Expected first entry to be evicted")
    }
}

// TestCacheOnlyGET verifies non-GET requests bypass cache
// Ensures idempotency requirements for caching are enforced
func TestCacheOnlyGET(t *testing.T) {
    cache := NewCache(config.CacheConfig{
        MaxSize: 10,
        TTL:     time.Minute,
    })

    callCount := 0
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        callCount++
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("test response"))
    })

    cachedHandler := cache.Wrap(handler)

    // POST request should not be cached
    req := httptest.NewRequest("POST", "/test", nil)
    w1 := httptest.NewRecorder()
    cachedHandler.ServeHTTP(w1, req)

    // Second POST request should hit backend again
    w2 := httptest.NewRecorder()
    cachedHandler.ServeHTTP(w2, req)

    if callCount != 2 {
        t.Errorf("Expected 2 backend calls for POST requests, got %d", callCount)
    }
}