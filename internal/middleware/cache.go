package middleware

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/WillKirkmanM/proxy/internal/config"
)

// CacheEntry represents a cached HTTP response with metadata
// Stores complete response data including headers and expiration time
// TTL-based expiration ensures stale data is not served indefinitely
type CacheEntry struct {
    Body       []byte      // Response body content
    Headers    http.Header // HTTP response headers
    StatusCode int         // HTTP status code
    ExpiresAt  time.Time   // Absolute expiration time for TTL
}

// IsExpired checks if cache entry has exceeded its TTL
// Used by cache lookup to determine if entry should be evicted
// Time Complexity: O(1) - simple time comparison
// Space Complexity: O(1) - no additional allocations
func (ce *CacheEntry) IsExpired() bool {
    return time.Now().After(ce.ExpiresAt)
}

// Cache implements LRU caching middleware for HTTP responses
// Reduces backend load by serving frequently requested content from memory
// Uses LRU eviction policy when cache reaches maximum size
// Time Complexity: O(1) for cache operations with hash map and doubly-linked list
// Space Complexity: O(n) where n is number of cached entries
type Cache struct {
    entries   map[string]*cacheNode // Hash map for O(1) key lookup
    head      *cacheNode            // Most recently used entry (dummy head)
    tail      *cacheNode            // Least recently used entry (dummy tail)
    mutex     sync.RWMutex          // Protects cache data structures
    maxSize   int                   // Maximum number of entries before eviction
    ttl       time.Duration         // Time-to-live for cache entries
    currentSize int                 // Current number of entries in cache
}

// cacheNode represents a node in the doubly-linked list for LRU tracking
// Doubly-linked structure allows O(1) insertion and removal operations
// Contains both key and value for efficient eviction
type cacheNode struct {
    key   string      // Cache key for reverse lookup during eviction
    entry *CacheEntry // Cached response data
    prev  *cacheNode  // Previous node in LRU order
    next  *cacheNode  // Next node in LRU order
}

// NewCache creates a new caching middleware with LRU eviction policy
// Initializes doubly-linked list with dummy head and tail nodes
// Dummy nodes simplify insertion and removal logic
// Time Complexity: O(1) - constant time initialisation
// Space Complexity: O(1) initial, grows to O(maxSize)
func NewCache(config config.CacheConfig) *Cache {
    // Create dummy head and tail nodes for simplified list operations
    head := &cacheNode{}
    tail := &cacheNode{}
    head.next = tail
    tail.prev = head

    return &Cache{
        entries:     make(map[string]*cacheNode),
        head:        head,
        tail:        tail,
        maxSize:     config.MaxSize,
        ttl:         config.TTL,
        currentSize: 0,
    }
}

// Wrap decorates handler with response caching functionality
// Checks cache before forwarding request, stores response after processing
// Only caches successful GET requests to avoid caching errors or side effects
// Time Complexity: O(1) for cache hit, O(n) for cache miss where n is response size
// Space Complexity: O(1) for cache operations, O(n) for response buffering
func (c *Cache) Wrap(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Only cache GET requests as they should be idempotent
        // POST, PUT, DELETE may have side effects and shouldn't be cached
        if r.Method != http.MethodGet {
            next.ServeHTTP(w, r)
            return
        }

        // Generate cache key from request URL and relevant headers
        // Key includes URL and headers that affect response content
        cacheKey := c.generateCacheKey(r)

        // Check cache for existing entry
        if entry := c.get(cacheKey); entry != nil {
            // Cache hit - serve response from cache
            c.serveFromCache(w, entry)
            return
        }

        // Cache miss - create response writer wrapper to capture response
        wrapper := &responseWriter{
            ResponseWriter: w,
            body:           &bytes.Buffer{},
            headers:        make(http.Header),
        }

        // Process request with wrapped response writer
        next.ServeHTTP(wrapper, r)

        // Cache successful responses (2xx status codes)
        // Error responses are not cached to avoid serving stale errors
        if wrapper.statusCode >= 200 && wrapper.statusCode < 300 {
            entry := &CacheEntry{
                Body:       wrapper.body.Bytes(),
                Headers:    wrapper.headers,
                StatusCode: wrapper.statusCode,
                ExpiresAt:  time.Now().Add(c.ttl),
            }
            c.set(cacheKey, entry)
        }
    })
}

// generateCacheKey creates unique key for request caching
// Includes URL and headers that affect response content (Accept, Accept-Encoding)
// MD5 hash ensures consistent key length regardless of URL complexity
// Time Complexity: O(n) where n is URL length plus relevant headers
// Space Complexity: O(1) - fixed size hash output
func (c *Cache) generateCacheKey(r *http.Request) string {
    // Include URL and relevant headers in cache key
    // Headers like Accept and Accept-Encoding affect response content
    keyData := fmt.Sprintf("%s|%s|%s", 
        r.URL.String(),
        r.Header.Get("Accept"),
        r.Header.Get("Accept-Encoding"),
    )

    // Use MD5 hash for consistent key length and character set
    // Cryptographic security not required for cache keys
    hash := md5.Sum([]byte(keyData))
    return fmt.Sprintf("%x", hash)
}

// get retrieves entry from cache with LRU update
// Returns nil if entry doesn't exist or has expired
// Moves accessed entry to front of LRU list
// Time Complexity: O(1) - hash map lookup and list manipulation
// Space Complexity: O(1) - no additional allocations
func (c *Cache) get(key string) *CacheEntry {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    node, exists := c.entries[key]
    if !exists {
        return nil
    }

    // Check if entry has expired
    if node.entry.IsExpired() {
        c.removeNode(node)
        delete(c.entries, key)
        c.currentSize--
        return nil
    }

    // Move accessed node to front (most recently used)
    c.moveToFront(node)
    return node.entry
}

// set stores entry in cache with LRU eviction if necessary
// Creates new node and adds to front of LRU list
// Evicts least recently used entry if cache is full
// Time Complexity: O(1) - hash map insertion and list manipulation
// Space Complexity: O(1) per entry - stores response data
func (c *Cache) set(key string, entry *CacheEntry) {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    // Check if key already exists (update scenario)
    if node, exists := c.entries[key]; exists {
        node.entry = entry
        c.moveToFront(node)
        return
    }

    // Create new node and add to cache
    node := &cacheNode{
        key:   key,
        entry: entry,
    }

    c.entries[key] = node
    c.addToFront(node)
    c.currentSize++

    // Evict least recently used entry if cache is full
    if c.currentSize > c.maxSize {
        c.evictLRU()
    }
}

// moveToFront moves existing node to front of LRU list
// Indicates recent access for LRU tracking
// Time Complexity: O(1) - constant time list manipulation
// Space Complexity: O(1) - no additional allocations
func (c *Cache) moveToFront(node *cacheNode) {
    c.removeNode(node)
    c.addToFront(node)
}

// addToFront adds node immediately after dummy head
// New nodes are most recently used by definition
// Time Complexity: O(1) - constant time list insertion
// Space Complexity: O(1) - no additional allocations
func (c *Cache) addToFront(node *cacheNode) {
    node.prev = c.head
    node.next = c.head.next
    c.head.next.prev = node
    c.head.next = node
}

// removeNode removes node from doubly-linked list
// Maintains list integrity by updating neighbor pointers
// Time Complexity: O(1) - constant time list removal
// Space Complexity: O(1) - no additional allocations
func (c *Cache) removeNode(node *cacheNode) {
    node.prev.next = node.next
    node.next.prev = node.prev
}

// evictLRU removes least recently used entry from cache
// Called when cache reaches maximum size to make room for new entries
// Time Complexity: O(1) - removes from tail of LRU list
// Space Complexity: O(1) - frees memory by removing entry
func (c *Cache) evictLRU() {
    lru := c.tail.prev
    c.removeNode(lru)
    delete(c.entries, lru.key)
    c.currentSize--
}

// serveFromCache writes cached response to HTTP response writer
// Copies headers, status code, and body from cache entry
// Adds cache status header to indicate cache hit
// Time Complexity: O(n) where n is response body size
// Space Complexity: O(1) - streams data without additional buffering
func (c *Cache) serveFromCache(w http.ResponseWriter, entry *CacheEntry) {
    // Copy cached headers to response
    for key, values := range entry.Headers {
        for _, value := range values {
            w.Header().Add(key, value)
        }
    }

    // Add cache status header for debugging and monitoring
    w.Header().Set("X-Cache-Status", "HIT")
    
    // Set status code and write response body
    w.WriteHeader(entry.StatusCode)
    w.Write(entry.Body)
}

// responseWriter wraps http.ResponseWriter to capture response data
// Implements decorator pattern to intercept response writes
// Buffers response body and headers for caching while preserving original behavior
type responseWriter struct {
    http.ResponseWriter
    body       *bytes.Buffer
    headers    http.Header
    statusCode int
}

// Write captures response body data while passing through to original writer
// Implements io.Writer interface for HTTP response writing
// Time Complexity: O(n) where n is data length
// Space Complexity: O(n) for buffering response data
func (rw *responseWriter) Write(data []byte) (int, error) {
    // Buffer data for caching
    rw.body.Write(data)
    
    // Pass through to original writer
    return rw.ResponseWriter.Write(data)
}

// WriteHeader captures status code and headers while passing through
// Must be called before Write() as per HTTP specification
// Time Complexity: O(h) where h is number of headers
// Space Complexity: O(h) for header storage
func (rw *responseWriter) WriteHeader(statusCode int) {
    rw.statusCode = statusCode
    
    // Copy headers for caching
    for key, values := range rw.ResponseWriter.Header() {
        rw.headers[key] = make([]string, len(values))
        copy(rw.headers[key], values)
    }
    
    // Pass through to original writer
    rw.ResponseWriter.WriteHeader(statusCode)
}

// Header returns the header map for the response
// Preserves original ResponseWriter interface
// Time Complexity: O(1) - returns reference
// Space Complexity: O(1) - no additional allocations
func (rw *responseWriter) Header() http.Header {
    return rw.ResponseWriter.Header()
}