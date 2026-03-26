package handlers

import (
	"crypto/sha256"
	"fmt"
	"os"
	"sync"
	"time"
)

func init() {
	if os.Getenv("DISABLE_SSAR_CACHE") == "true" {
		CacheDisabled = true
	}
}

// ssarCacheTTL is how long a cached SSAR result is valid.
// RBAC changes take at most this long to take effect.
const ssarCacheTTL = 30 * time.Second

// ssarCacheMaxSize is the maximum number of entries before random eviction.
const ssarCacheMaxSize = 10000

// ssarCacheEntry holds a cached SSAR result with expiry.
type ssarCacheEntry struct {
	allowed   bool
	expiresAt time.Time
}

// ssarCache is a thread-safe TTL cache for SelfSubjectAccessReview results.
// Cache key: sha256(token):namespace:verb:group:resource
type ssarCache struct {
	mu      sync.RWMutex
	entries map[string]ssarCacheEntry
}

// globalSSARCache is the package-level SSAR cache instance.
var globalSSARCache = &ssarCache{
	entries: make(map[string]ssarCacheEntry),
}

// ssarCacheKey builds a cache key from the request parameters.
// The token is hashed so raw credentials are never stored.
func ssarCacheKey(token, namespace, verb, group, resource string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x:%s:%s:%s:%s", h[:8], namespace, verb, group, resource)
}

// CacheDisabled can be set to true to bypass the cache for A/B benchmarking.
var CacheDisabled bool

// check returns the cached SSAR result if present and not expired.
func (c *ssarCache) check(key string) (allowed bool, found bool) {
	if CacheDisabled {
		return false, false
	}
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return false, false
	}
	if time.Now().After(entry.expiresAt) {
		// Expired — remove lazily, but re-check under write lock
		// in case another goroutine refreshed the entry concurrently.
		c.mu.Lock()
		if current, stillExists := c.entries[key]; stillExists {
			if time.Now().After(current.expiresAt) {
				delete(c.entries, key)
				c.mu.Unlock()
				return false, false
			}
			// Entry was refreshed by a concurrent store — use it
			c.mu.Unlock()
			return current.allowed, true
		}
		c.mu.Unlock()
		return false, false
	}
	return entry.allowed, true
}

// store saves an SSAR result in the cache with TTL.
func (c *ssarCache) store(key string, allowed bool) {
	if CacheDisabled {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity (simple: clear half the entries)
	if len(c.entries) >= ssarCacheMaxSize {
		count := 0
		for k := range c.entries {
			delete(c.entries, k)
			count++
			if count >= ssarCacheMaxSize/2 {
				break
			}
		}
	}

	c.entries[key] = ssarCacheEntry{
		allowed:   allowed,
		expiresAt: time.Now().Add(ssarCacheTTL),
	}
}
