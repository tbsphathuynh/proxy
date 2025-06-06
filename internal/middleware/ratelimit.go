package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/WillKirkmanM/proxy/internal/config"
)

// TokenBucket implements token bucket algorithm for rate limiting
// Allows burst traffic up to bucket capacity while maintaining sustained rate
// Refills tokens at specified rate to prevent resource exhaustion
// Time Complexity: O(1) for token operations
// Space Complexity: O(1) per bucket instance
type TokenBucket struct {
    capacity     int           // Maximum tokens in bucket
    tokens       int           // Current available tokens
    refillRate   int           // Tokens added per second
    lastRefill   time.Time     // Last time bucket was refilled
    mutex        sync.Mutex    // Protects bucket state
}

// NewTokenBucket creates token bucket with specified capacity and refill rate
// Initializes bucket at full capacity for immediate availability
// Time Complexity: O(1) - constant time initialisation
// Space Complexity: O(1) - fixed size structure
func NewTokenBucket(capacity, refillRate int) *TokenBucket {
    return &TokenBucket{
        capacity:   capacity,
        tokens:     capacity,
        refillRate: refillRate,
        lastRefill: time.Now(),
    }
}

// TryConsume attempts to consume specified number of tokens
// Returns true if tokens available, false if rate limit exceeded
// Refills bucket based on elapsed time since last refill
// Time Complexity: O(1) - constant time operations
// Space Complexity: O(1) - no additional allocations
func (tb *TokenBucket) TryConsume(tokens int) bool {
    tb.mutex.Lock()
    defer tb.mutex.Unlock()

    tb.refill()

    if tb.tokens >= tokens {
        tb.tokens -= tokens
        return true
    }
    return false
}

// refill adds tokens to bucket based on elapsed time
// Calculates tokens to add using time difference and refill rate
// Caps tokens at bucket capacity to prevent overflow
// Time Complexity: O(1) - simple arithmetic operations
// Space Complexity: O(1) - no additional allocations
func (tb *TokenBucket) refill() {
    now := time.Now()
    elapsed := now.Sub(tb.lastRefill)
    
    // Calculate tokens to add based on elapsed time
    tokensToAdd := int(elapsed.Seconds()) * tb.refillRate
    
    if tokensToAdd > 0 {
        tb.tokens += tokensToAdd
        if tb.tokens > tb.capacity {
            tb.tokens = tb.capacity
        }
        tb.lastRefill = now
    }
}

// RateLimiter manages rate limiting for HTTP requests
// Uses token bucket algorithm with client IP-based bucketing
// Prevents abuse while allowing legitimate burst traffic
// Time Complexity: O(1) for rate limit checks
// Space Complexity: O(n) where n is number of unique client IPs
type RateLimiter struct {
    buckets    map[string]*TokenBucket // Per-client token buckets
    mutex      sync.RWMutex            // Protects buckets map
    capacity   int                     // Bucket capacity
    refillRate int                     // Tokens per second
}

// NewRateLimiter creates rate limiter with specified limits
// Initializes empty bucket map for lazy client bucket creation
// Time Complexity: O(1) - constant time initialisation
// Space Complexity: O(1) initial, grows with unique clients
func NewRateLimiter(config config.RateLimitConfig) *RateLimiter {
    return &RateLimiter{
        buckets:    make(map[string]*TokenBucket),
        capacity:   config.Capacity,
        refillRate: config.RefillRate,
    }
}

// Wrap decorates handler with rate limiting functionality
// Extracts client IP and checks against token bucket
// Returns 429 Too Many Requests if rate limit exceeded
// Time Complexity: O(1) for rate limit check
// Space Complexity: O(1) per unique client IP
func (rl *RateLimiter) Wrap(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract client IP for rate limiting
        clientIP := rl.getClientIP(r)
        
        // Get or create token bucket for client
        bucket := rl.getBucket(clientIP)
        
        // Try to consume one token for this request
        if !bucket.TryConsume(1) {
            // Rate limit exceeded - return 429 status
            w.Header().Set("X-RateLimit-Limit", string(rune(rl.capacity)))
            w.Header().Set("X-RateLimit-Remaining", "0")
            w.WriteHeader(http.StatusTooManyRequests)
            w.Write([]byte("Rate limit exceeded"))
            return
        }
        
        // Rate limit OK - process request
        w.Header().Set("X-RateLimit-Limit", string(rune(rl.capacity)))
        next.ServeHTTP(w, r)
    })
}

// getBucket retrieves or creates token bucket for client IP
// Uses lazy initialisation to avoid memory waste for inactive clients
// Double-checked locking pattern for thread safety and performance
// Time Complexity: O(1) - hash map lookup
// Space Complexity: O(1) per new client IP
func (rl *RateLimiter) getBucket(clientIP string) *TokenBucket {
    // Try read lock first for performance
    rl.mutex.RLock()
    bucket, exists := rl.buckets[clientIP]
    rl.mutex.RUnlock()
    
    if exists {
        return bucket
    }
    
    // Need to create bucket - acquire write lock
    rl.mutex.Lock()
    defer rl.mutex.Unlock()
    
    // Double-check in case another goroutine created it
    if bucket, exists := rl.buckets[clientIP]; exists {
        return bucket
    }
    
    // Create new bucket for client
    bucket = NewTokenBucket(rl.capacity, rl.refillRate)
    rl.buckets[clientIP] = bucket
    return bucket
}

// getClientIP extracts client IP address from request
// Checks proxy headers before falling back to remote address
// Handles X-Forwarded-For and X-Real-IP headers for proxy scenarios
// Time Complexity: O(1) - header lookups
// Space Complexity: O(1) - returns string reference
func (rl *RateLimiter) getClientIP(r *http.Request) string {
    // Check X-Forwarded-For header (comma-separated list, first is client)
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        // Take first IP from comma-separated list
        if commaIdx := len(xff); commaIdx > 0 {
            for i, char := range xff {
                if char == ',' {
                    commaIdx = i
                    break
                }
            }
            return xff[:commaIdx]
        }
        return xff
    }
    
    // Check X-Real-IP header
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return xri
    }
    
    // Fall back to remote address
    return r.RemoteAddr
}