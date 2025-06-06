package loadbalancer

import (
    "net/http/httptest"
    "testing"
)

// BenchmarkRoundRobinSelection benchmarks backend selection performance
// Measures time complexity of round-robin algorithm under load
func BenchmarkRoundRobinSelection(b *testing.B) {
    // Create backends for benchmarking
    backends := make([]Backend, 10)
    for i := 0; i < 10; i++ {
        backend, _ := NewHTTPBackend("http://example.com:808"+string(rune(i)), 1)
        backends[i] = backend
    }

    lb := NewRoundRobinBalancer(backends)
    req := httptest.NewRequest("GET", "/", nil)

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        _, err := lb.SelectBackend(req)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// BenchmarkRoundRobinConcurrent benchmarks concurrent backend selection
// Tests performance under concurrent load with multiple goroutines
func BenchmarkRoundRobinConcurrent(b *testing.B) {
    backends := make([]Backend, 10)
    for i := 0; i < 10; i++ {
        backend, _ := NewHTTPBackend("http://example.com:808"+string(rune(i)), 1)
        backends[i] = backend
    }

    lb := NewRoundRobinBalancer(backends)
    req := httptest.NewRequest("GET", "/", nil)

    b.ResetTimer()
    b.ReportAllocs()

    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := lb.SelectBackend(req)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}