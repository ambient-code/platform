package webhook

import (
	"context"
	"sync"
	"time"
)

// DeduplicationCache provides in-memory storage of delivery IDs to prevent duplicate
// webhook processing (FR-011, FR-023). It automatically expires entries after the TTL.
//
// This cache is safe for concurrent use and handles pod restarts gracefully via
// deterministic session naming (FR-024).
type DeduplicationCache struct {
	mu      sync.RWMutex
	entries map[string]time.Time // deliveryID -> expirationTime
	ttl     time.Duration
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewDeduplicationCache creates a new deduplication cache with the specified TTL
// For GitHub webhooks, the recommended TTL is 24 hours to handle potential retry windows
func NewDeduplicationCache(ttl time.Duration) *DeduplicationCache {
	ctx, cancel := context.WithCancel(context.Background())
	cache := &DeduplicationCache{
		entries: make(map[string]time.Time),
		ttl:     ttl,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Start background cleanup goroutine (C3 fix: with context cancellation)
	go cache.cleanupExpired()

	return cache
}

// Shutdown stops the background cleanup goroutine (C3 fix)
func (c *DeduplicationCache) Shutdown() {
	c.cancel()
}

// IsDuplicate checks if a delivery ID has been seen before
// Returns true if the delivery ID exists in the cache (duplicate), false otherwise
func (c *DeduplicationCache) IsDuplicate(deliveryID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	expirationTime, exists := c.entries[deliveryID]
	if !exists {
		return false
	}

	// Check if entry has expired
	if time.Now().After(expirationTime) {
		// Entry expired but not yet cleaned up
		return false
	}

	return true
}

// Add adds a delivery ID to the cache with automatic expiration after TTL
// Returns true if the entry was added, false if it already exists
func (c *DeduplicationCache) Add(deliveryID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already exists and not expired
	if expirationTime, exists := c.entries[deliveryID]; exists {
		if time.Now().Before(expirationTime) {
			return false // Already exists and not expired
		}
	}

	// Add new entry or update expired entry
	c.entries[deliveryID] = time.Now().Add(c.ttl)
	return true
}

// cleanupExpired runs periodically to remove expired entries from the cache
// This prevents unbounded memory growth over time
func (c *DeduplicationCache) cleanupExpired() {
	ticker := time.NewTicker(10 * time.Minute) // Cleanup every 10 minutes
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			// Context cancelled - stop cleanup goroutine (C3 fix)
			return
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for deliveryID, expirationTime := range c.entries {
				if now.After(expirationTime) {
					delete(c.entries, deliveryID)
				}
			}
			c.mu.Unlock()
		}
	}
}

// Size returns the current number of entries in the cache (for monitoring)
func (c *DeduplicationCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Clear removes all entries from the cache (primarily for testing)
func (c *DeduplicationCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]time.Time)
}
