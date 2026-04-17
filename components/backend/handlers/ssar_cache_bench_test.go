package handlers

import (
	"fmt"
	"sync"
	"testing"
)

// BenchmarkSSARCache_Hit measures the cost of a cache hit (the hot path after the fix).
// This is what every API request does instead of calling K8s SSAR.
func BenchmarkSSARCache_Hit(b *testing.B) {
	cache := &ssarCache{entries: make(map[string]ssarCacheEntry)}

	// Pre-populate with a cached entry
	key := ssarCacheKey("test-token-123", "my-project", "list", "vteam.ambient-code", "agenticsessions")
	cache.store(key, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.check(key)
	}
}

// BenchmarkSSARCache_Miss measures the cost of a cache miss + store.
func BenchmarkSSARCache_Miss(b *testing.B) {
	cache := &ssarCache{entries: make(map[string]ssarCacheEntry)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := ssarCacheKey(fmt.Sprintf("token-%d", i), "project", "list", "vteam.ambient-code", "agenticsessions")
		cache.check(key)
		cache.store(key, true)
	}
}

// BenchmarkSSARCache_ConcurrentHits measures cache performance under contention
// from many goroutines (simulates concurrent API requests).
func BenchmarkSSARCache_ConcurrentHits(b *testing.B) {
	cache := &ssarCache{entries: make(map[string]ssarCacheEntry)}

	// Pre-populate 100 entries (100 user/project combos)
	keys := make([]string, 100)
	for i := 0; i < 100; i++ {
		keys[i] = ssarCacheKey(fmt.Sprintf("token-%d", i), fmt.Sprintf("project-%d", i%10), "list", "vteam.ambient-code", "agenticsessions")
		cache.store(keys[i], true)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.check(keys[i%100])
			i++
		}
	})
}

// BenchmarkSSARCacheKey measures the cost of computing a cache key (includes SHA256).
func BenchmarkSSARCacheKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ssarCacheKey("eyJhbGciOiJSUzI1NiIsImtpZCI6InRlc3QifQ.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OmRlZmF1bHQ6dGVzdCJ9",
			"my-project-namespace", "list", "vteam.ambient-code", "agenticsessions")
	}
}

// TestSSARCache_Correctness verifies cache behavior.
func TestSSARCache_Correctness(t *testing.T) {
	cache := &ssarCache{entries: make(map[string]ssarCacheEntry)}

	key := ssarCacheKey("token-1", "project-a", "list", "vteam.ambient-code", "agenticsessions")

	// Miss
	if _, found := cache.check(key); found {
		t.Fatal("expected miss on empty cache")
	}

	// Store + hit
	cache.store(key, true)
	allowed, found := cache.check(key)
	if !found || !allowed {
		t.Fatal("expected hit after store")
	}

	// Different user, same project = miss
	key2 := ssarCacheKey("token-2", "project-a", "list", "vteam.ambient-code", "agenticsessions")
	if _, found := cache.check(key2); found {
		t.Fatal("expected miss for different user")
	}

	// Same user, different project = miss
	key3 := ssarCacheKey("token-1", "project-b", "list", "vteam.ambient-code", "agenticsessions")
	if _, found := cache.check(key3); found {
		t.Fatal("expected miss for different project")
	}

	// Concurrent safety
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			k := ssarCacheKey(fmt.Sprintf("token-%d", n), "project", "list", "vteam.ambient-code", "agenticsessions")
			cache.store(k, n%2 == 0)
			cache.check(k)
		}(i)
	}
	wg.Wait()
}
