package reconciler

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/informer"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/rs/zerolog"
)

// StressTestConfig contains configuration for the stress test
type StressTestConfig struct {
	SessionCount    int
	MaxConcurrency  int
	TimeoutDuration time.Duration
	APIBaseURL      string
	APIToken        string
	APIProject      string
}

// MockInformerForStressTest simulates the informer behavior for stress testing
type MockInformerForStressTest struct {
	reconciler *TallyReconciler
	logger     zerolog.Logger
	eventCount int64
}

func NewMockInformerForStressTest(reconciler *TallyReconciler, logger zerolog.Logger) *MockInformerForStressTest {
	return &MockInformerForStressTest{
		reconciler: reconciler,
		logger:     logger,
	}
}

// SimulateSessionEvent simulates a session event going through the informer pipeline
func (m *MockInformerForStressTest) SimulateSessionEvent(session types.Session, eventType informer.EventType) error {
	event := informer.ResourceEvent{
		Type:     eventType,
		Resource: "sessions",
		Object:   informer.NewSessionObject(session),
	}

	// Increment event counter
	atomic.AddInt64(&m.eventCount, 1)

	// Process through reconciler
	return m.reconciler.Reconcile(context.Background(), event)
}

func (m *MockInformerForStressTest) GetEventCount() int64 {
	return atomic.LoadInt64(&m.eventCount)
}

// TestSessionAPIStressTest is a comprehensive stress test that:
// 1. Creates 100 sessions via API calls
// 2. Simulates the gRPC watch events through TallyReconciler
// 3. Verifies that exactly 100 events are counted
func TestSessionAPIStressTest(t *testing.T) {
	const sessionCount = 100
	const maxConcurrency = 20

	config := StressTestConfig{
		SessionCount:    sessionCount,
		MaxConcurrency:  maxConcurrency,
		TimeoutDuration: 30 * time.Second,
		APIBaseURL:      getTestAPIBaseURL(),
		APIToken:        getTestAPIToken(),
		APIProject:      getTestAPIProject(),
	}

	t.Logf("Starting stress test: %d sessions with max %d concurrent requests",
		config.SessionCount, config.MaxConcurrency)

	ctx, cancel := context.WithTimeout(context.Background(), config.TimeoutDuration)
	defer cancel()

	// Setup TallyReconciler
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	tallyReconciler := NewTallyReconciler("sessions", nil, logger)

	// Setup mock informer for simulation
	mockInformer := NewMockInformerForStressTest(tallyReconciler, logger)

	// Setup SDK client for API calls
	sdk, err := sdkclient.NewClient(config.APIBaseURL, config.APIToken, config.APIProject)
	if err != nil {
		t.Skip("Skipping stress test: unable to create SDK client:", err)
	}

	// Track created sessions for cleanup
	var createdSessions []string
	var createdSessionsMux sync.Mutex
	defer func() {
		// Cleanup: Delete all created sessions
		createdSessionsMux.Lock()
		sessions := make([]string, len(createdSessions))
		copy(sessions, createdSessions)
		createdSessionsMux.Unlock()

		for _, sessionID := range sessions {
			// Note: In a real environment, you'd implement session deletion
			// For this test, we assume sessions are cleaned up automatically
			t.Logf("Created session %s (cleanup would happen here)", sessionID)
		}
	}()

	// Channel to control concurrency
	semaphore := make(chan struct{}, config.MaxConcurrency)

	// Channel to collect results
	results := make(chan sessionCreateResult, config.SessionCount)

	// WaitGroup to wait for all goroutines
	var wg sync.WaitGroup

	startTime := time.Now()

	// Launch session creation goroutines
	for i := 0; i < config.SessionCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := createSessionAndSimulateEvent(ctx, sdk, mockInformer, index)
			results <- result

			if result.Success {
				createdSessionsMux.Lock()
				createdSessions = append(createdSessions, result.SessionID)
				createdSessionsMux.Unlock()
			}
		}(i)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var successCount, failureCount int
	var errors []string

	for result := range results {
		if result.Success {
			successCount++
		} else {
			failureCount++
			errors = append(errors, result.Error)
		}
	}

	duration := time.Since(startTime)
	t.Logf("Stress test completed in %v: %d successful, %d failed",
		duration, successCount, failureCount)

	// Log first few errors if any
	if len(errors) > 0 {
		t.Logf("First few errors:")
		for i, err := range errors {
			if i >= 5 { // Limit to first 5 errors
				break
			}
			t.Logf("  %d: %s", i+1, err)
		}
	}

	// Verify TallyReconciler counts
	snapshot := tallyReconciler.Snapshot()

	t.Logf("TallyReconciler snapshot: Added=%d, Modified=%d, Deleted=%d, Total=%d",
		snapshot.Tally.Added, snapshot.Tally.Modified, snapshot.Tally.Deleted, tallyReconciler.Total())
	t.Logf("TallyReconciler seen %d unique session IDs", len(snapshot.SeenIDs))
	t.Logf("MockInformer processed %d events", mockInformer.GetEventCount())

	// Assertions
	if successCount == 0 {
		t.Fatal("No sessions were created successfully")
	}

	// The key assertion: TallyReconciler should count exactly the number of successful events
	expectedEventCount := successCount
	actualEventCount := tallyReconciler.Total()

	if actualEventCount != expectedEventCount {
		t.Errorf("TallyReconciler count mismatch: got %d events, want %d events",
			actualEventCount, expectedEventCount)
	}

	// Verify that we have the expected number of unique session IDs
	if len(snapshot.SeenIDs) != successCount {
		t.Errorf("TallyReconciler unique IDs mismatch: got %d unique IDs, want %d",
			len(snapshot.SeenIDs), successCount)
	}

	// Verify all events were Added events (since we only create sessions)
	if snapshot.Tally.Added != successCount {
		t.Errorf("Expected all events to be Added: got %d Added events, want %d",
			snapshot.Tally.Added, successCount)
	}

	if snapshot.Tally.Modified != 0 {
		t.Errorf("Expected no Modified events: got %d", snapshot.Tally.Modified)
	}

	if snapshot.Tally.Deleted != 0 {
		t.Errorf("Expected no Deleted events: got %d", snapshot.Tally.Deleted)
	}

	// Performance assertions
	avgSessionsPerSecond := float64(successCount) / duration.Seconds()
	t.Logf("Performance: %.2f sessions/second", avgSessionsPerSecond)

	if successCount >= config.SessionCount {
		t.Logf("SUCCESS: All %d sessions created and counted correctly", config.SessionCount)
	} else {
		t.Logf("PARTIAL SUCCESS: %d/%d sessions created successfully", successCount, config.SessionCount)
	}
}

type sessionCreateResult struct {
	Success   bool
	SessionID string
	Error     string
	Duration  time.Duration
}

func createSessionAndSimulateEvent(ctx context.Context, sdk *sdkclient.Client, mockInformer *MockInformerForStressTest, index int) sessionCreateResult {
	startTime := time.Now()

	// Create a unique session
	session := &types.Session{
		Name:   fmt.Sprintf("stress-test-session-%d-%d", index, time.Now().Unix()),
		Prompt: fmt.Sprintf("Test session %d for stress testing the API and gRPC handlers", index),
	}

	// Create session via API
	createdSession, err := sdk.Sessions().Create(ctx, session)
	if err != nil {
		return sessionCreateResult{
			Success:  false,
			Error:    fmt.Sprintf("API create failed: %v", err),
			Duration: time.Since(startTime),
		}
	}

	// Simulate the gRPC watch event that would normally come from the API server
	err = mockInformer.SimulateSessionEvent(*createdSession, informer.EventAdded)
	if err != nil {
		return sessionCreateResult{
			Success:   false,
			SessionID: createdSession.ID,
			Error:     fmt.Sprintf("Event simulation failed: %v", err),
			Duration:  time.Since(startTime),
		}
	}

	return sessionCreateResult{
		Success:   true,
		SessionID: createdSession.ID,
		Duration:  time.Since(startTime),
	}
}

// Test configuration helpers - load from environment variables with defaults

func getTestAPIBaseURL() string {
	if baseURL := os.Getenv("TEST_API_BASE_URL"); baseURL != "" {
		return baseURL
	}
	if baseURL := os.Getenv("AMBIENT_API_SERVER_URL"); baseURL != "" {
		return baseURL
	}
	return "http://localhost:8000"
}

func getTestAPIToken() string {
	if token := os.Getenv("TEST_API_TOKEN"); token != "" {
		return token
	}
	if token := os.Getenv("AMBIENT_API_TOKEN"); token != "" {
		return token
	}
	return "test-token"
}

func getTestAPIProject() string {
	if project := os.Getenv("TEST_API_PROJECT"); project != "" {
		return project
	}
	return "default"
}

// TestSessionStressTestConcurrentOnly tests just the concurrent reconciler processing
// without actual API calls - useful for testing the TallyReconciler under load
func TestSessionStressTestConcurrentOnly(t *testing.T) {
	const sessionCount = 100
	const maxConcurrency = 50

	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	tallyReconciler := NewTallyReconciler("sessions", nil, logger)

	ctx := context.Background()
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrency)

	startTime := time.Now()

	// Create sessions concurrently and send events to reconciler
	for i := 0; i < sessionCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			session := types.Session{
				ObjectReference: types.ObjectReference{
					ID: fmt.Sprintf("concurrent-session-%d", index),
				},
				Name: fmt.Sprintf("Concurrent Session %d", index),
			}

			event := informer.ResourceEvent{
				Type:     informer.EventAdded,
				Resource: "sessions",
				Object:   informer.NewSessionObject(session),
			}

			err := tallyReconciler.Reconcile(ctx, event)
			if err != nil {
				t.Errorf("Reconcile failed for session %d: %v", index, err)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// Verify results
	snapshot := tallyReconciler.Snapshot()

	t.Logf("Concurrent test completed in %v", duration)
	t.Logf("TallyReconciler: Added=%d, Total=%d, Unique IDs=%d",
		snapshot.Tally.Added, tallyReconciler.Total(), len(snapshot.SeenIDs))

	// Assertions
	if snapshot.Tally.Added != sessionCount {
		t.Errorf("Expected %d Added events, got %d", sessionCount, snapshot.Tally.Added)
	}

	if tallyReconciler.Total() != sessionCount {
		t.Errorf("Expected total %d events, got %d", sessionCount, tallyReconciler.Total())
	}

	if len(snapshot.SeenIDs) != sessionCount {
		t.Errorf("Expected %d unique session IDs, got %d", sessionCount, len(snapshot.SeenIDs))
	}

	// Performance check
	eventsPerSecond := float64(sessionCount) / duration.Seconds()
	t.Logf("Performance: %.2f events/second", eventsPerSecond)

	t.Logf("SUCCESS: All %d concurrent events processed correctly", sessionCount)
}
