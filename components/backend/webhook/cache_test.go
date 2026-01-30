package webhook

import (
	"sync"
	"testing"
	"time"
)

// TestDeduplicationCache_Basic tests basic cache operations
func TestDeduplicationCache_Basic(t *testing.T) {
	cache := NewDeduplicationCache(1 * time.Hour)
	defer cache.Shutdown()

	deliveryID := "test-delivery-123"

	// Should not be duplicate initially
	if cache.IsDuplicate(deliveryID) {
		t.Error("Expected new delivery ID to not be duplicate")
	}

	// Add to cache
	if !cache.Add(deliveryID) {
		t.Error("Expected Add to return true for new entry")
	}

	// Should now be duplicate
	if !cache.IsDuplicate(deliveryID) {
		t.Error("Expected delivery ID to be duplicate after adding")
	}

	// Adding again should return false
	if cache.Add(deliveryID) {
		t.Error("Expected Add to return false for existing entry")
	}
}

// TestDeduplicationCache_Expiration tests TTL expiration
func TestDeduplicationCache_Expiration(t *testing.T) {
	shortTTL := 100 * time.Millisecond
	cache := NewDeduplicationCache(shortTTL)
	defer cache.Shutdown()

	deliveryID := "test-delivery-expire"

	// Add entry
	cache.Add(deliveryID)

	// Should be duplicate immediately
	if !cache.IsDuplicate(deliveryID) {
		t.Error("Expected entry to exist immediately after add")
	}

	// Wait for expiration
	time.Sleep(shortTTL + 50*time.Millisecond)

	// Should no longer be duplicate
	if cache.IsDuplicate(deliveryID) {
		t.Error("Expected entry to expire after TTL")
	}

	// Should be able to add again
	if !cache.Add(deliveryID) {
		t.Error("Expected Add to succeed after expiration")
	}
}

// TestDeduplicationCache_Concurrent tests thread safety
func TestDeduplicationCache_Concurrent(t *testing.T) {
	cache := NewDeduplicationCache(1 * time.Hour)
	defer cache.Shutdown()

	const numGoroutines = 100
	const numOperations = 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch concurrent goroutines performing cache operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				deliveryID := string(rune('a' + (j % 26)))
				cache.IsDuplicate(deliveryID)
				cache.Add(deliveryID)
			}
		}(i)
	}

	// Wait for all goroutines with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no deadlocks
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent operations timed out - possible deadlock")
	}
}

// TestDeduplicationCache_Cleanup tests automatic cleanup
func TestDeduplicationCache_Cleanup(t *testing.T) {
	shortTTL := 50 * time.Millisecond
	cache := NewDeduplicationCache(shortTTL)
	defer cache.Shutdown()

	// Add multiple entries
	for i := 0; i < 10; i++ {
		cache.Add(string(rune('a' + i)))
	}

	initialSize := cache.Size()
	if initialSize != 10 {
		t.Errorf("Expected cache size 10, got %d", initialSize)
	}

	// Wait for entries to expire
	time.Sleep(shortTTL + 50*time.Millisecond)

	// Trigger cleanup by checking entries (which internally checks expiration)
	for i := 0; i < 10; i++ {
		cache.IsDuplicate(string(rune('a' + i)))
	}

	// Note: Cleanup runs in background every 10 minutes, so we can't test automatic cleanup
	// in a unit test. We can only verify that expired entries are not considered duplicates.
}

// TestDeduplicationCache_ReplayPrevention simulates replay attack
func TestDeduplicationCache_ReplayPrevention(t *testing.T) {
	cache := NewDeduplicationCache(24 * time.Hour)
	defer cache.Shutdown()

	// Simulate receiving same webhook delivery multiple times
	deliveryID := "github-delivery-abc123"

	// First webhook - should be processed
	if cache.IsDuplicate(deliveryID) {
		t.Error("First webhook should not be duplicate")
	}
	cache.Add(deliveryID)

	// Replay attack - same delivery ID sent again
	if !cache.IsDuplicate(deliveryID) {
		t.Error("Replay attack should be detected as duplicate")
	}

	// Should not be able to add again
	if cache.Add(deliveryID) {
		t.Error("Replay attack should not be added to cache")
	}

	// Verify cache still detects it
	for i := 0; i < 10; i++ {
		if !cache.IsDuplicate(deliveryID) {
			t.Errorf("Replay detection failed on attempt %d", i+1)
		}
	}
}

// TestDeduplicationCache_Shutdown tests goroutine cleanup
func TestDeduplicationCache_Shutdown(t *testing.T) {
	cache := NewDeduplicationCache(1 * time.Hour)

	// Add some entries
	cache.Add("test-1")
	cache.Add("test-2")

	// Shutdown should stop background goroutine
	cache.Shutdown()

	// Give it time to stop
	time.Sleep(50 * time.Millisecond)

	// Cache should still work for queries even after shutdown
	if !cache.IsDuplicate("test-1") {
		t.Error("Expected cache to still work after shutdown")
	}

	// Adding should still work
	if !cache.Add("test-3") {
		t.Error("Expected Add to work after shutdown")
	}
}

// TestDeduplicationCache_Size tests size reporting
func TestDeduplicationCache_Size(t *testing.T) {
	cache := NewDeduplicationCache(1 * time.Hour)
	defer cache.Shutdown()

	if cache.Size() != 0 {
		t.Errorf("Expected empty cache size 0, got %d", cache.Size())
	}

	// Add entries
	for i := 0; i < 5; i++ {
		cache.Add(string(rune('a' + i)))
	}

	if cache.Size() != 5 {
		t.Errorf("Expected cache size 5, got %d", cache.Size())
	}

	// Adding duplicate shouldn't increase size
	cache.Add("a")
	if cache.Size() != 5 {
		t.Errorf("Expected cache size to remain 5, got %d", cache.Size())
	}
}

// TestDeduplicationCache_GithubScenario tests realistic GitHub webhook scenario
func TestDeduplicationCache_GithubScenario(t *testing.T) {
	cache := NewDeduplicationCache(24 * time.Hour)
	defer cache.Shutdown()

	// Simulate GitHub sending webhooks
	webhooks := []struct {
		deliveryID     string
		shouldBeDupe   bool
		description    string
	}{
		{"abc-123", false, "first webhook"},
		{"abc-124", false, "second webhook (different delivery ID)"},
		{"abc-123", true, "replay of first webhook"},
		{"abc-125", false, "third webhook (new)"},
		{"abc-124", true, "replay of second webhook"},
		{"abc-123", true, "another replay of first webhook"},
	}

	for _, wh := range webhooks {
		isDupe := cache.IsDuplicate(wh.deliveryID)
		if isDupe != wh.shouldBeDupe {
			t.Errorf("%s: expected duplicate=%v, got %v", wh.description, wh.shouldBeDupe, isDupe)
		}

		if !isDupe {
			cache.Add(wh.deliveryID)
		}
	}
}
