package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"ambient-code-backend/types"
)

// setupEventLog creates a temporary event log with N events for benchmarking.
func setupEventLog(b *testing.B, eventCount int) (stateDir string, sessionName string) {
	b.Helper()
	stateDir = b.TempDir()
	sessionName = "bench-session"
	sessionDir := filepath.Join(stateDir, "sessions", sessionName)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		b.Fatal(err)
	}

	logPath := filepath.Join(sessionDir, "agui-events.jsonl")
	f, err := os.Create(logPath)
	if err != nil {
		b.Fatal(err)
	}

	// Write a realistic event sequence ending with RUN_STARTED (so status = "working")
	threadID := "thread-1"
	runID := "run-1"
	for i := 0; i < eventCount-1; i++ {
		evt := map[string]interface{}{
			"type":      "TEXT_MESSAGE_CONTENT",
			"threadId":  threadID,
			"runId":     runID,
			"messageId": fmt.Sprintf("msg-%d", i),
			"delta":     "some text content for benchmarking purposes",
			"timestamp": time.Now().UnixMilli(),
		}
		data, _ := json.Marshal(evt)
		f.Write(append(data, '\n'))
	}
	// Last event: RUN_STARTED (makes DeriveAgentStatus return "working")
	lastEvt := map[string]interface{}{
		"type":      "RUN_STARTED",
		"threadId":  threadID,
		"runId":     runID,
		"timestamp": time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(lastEvt)
	f.Write(append(data, '\n'))
	f.Close()

	return stateDir, sessionName
}

// BenchmarkEnrichAgentStatus_Uncached measures the cost of deriving agent status
// from the event log without caching (the old behavior).
func BenchmarkEnrichAgentStatus_Uncached(b *testing.B) {
	stateDir, sessionName := setupEventLog(b, 10000)

	// Point the websocket package at our temp dir
	origDerive := DeriveAgentStatusFromEvents
	defer func() { DeriveAgentStatusFromEvents = origDerive }()

	// Import the real DeriveAgentStatus from websocket package via the function pointer.
	// Since we can't import websocket here (circular), simulate with a file-scanning function.
	DeriveAgentStatusFromEvents = func(name string) string {
		path := filepath.Join(stateDir, "sessions", name, "agui-events.jsonl")
		return deriveStatusFromFile(path)
	}

	session := &types.AgenticSession{
		Metadata: map[string]interface{}{"name": sessionName},
		Status:   &types.AgenticSessionStatus{Phase: "Running"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear cache to force file scan every time
		agentStatusCache.Lock()
		delete(agentStatusCache.entries, sessionName)
		agentStatusCache.Unlock()

		enrichAgentStatus(session)
	}
}

// BenchmarkEnrichAgentStatus_Cached measures the cost with caching (the fix).
func BenchmarkEnrichAgentStatus_Cached(b *testing.B) {
	stateDir, sessionName := setupEventLog(b, 10000)

	origDerive := DeriveAgentStatusFromEvents
	defer func() { DeriveAgentStatusFromEvents = origDerive }()

	DeriveAgentStatusFromEvents = func(name string) string {
		path := filepath.Join(stateDir, "sessions", name, "agui-events.jsonl")
		return deriveStatusFromFile(path)
	}

	session := &types.AgenticSession{
		Metadata: map[string]interface{}{"name": sessionName},
		Status:   &types.AgenticSessionStatus{Phase: "Running"},
	}

	// Prime the cache with one call
	enrichAgentStatus(session)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enrichAgentStatus(session)
	}
}

// BenchmarkEnrichAgentStatus_Concurrent measures cached path under contention.
func BenchmarkEnrichAgentStatus_Concurrent(b *testing.B) {
	stateDir, _ := setupEventLog(b, 10000)

	origDerive := DeriveAgentStatusFromEvents
	defer func() { DeriveAgentStatusFromEvents = origDerive }()

	DeriveAgentStatusFromEvents = func(name string) string {
		path := filepath.Join(stateDir, "sessions", name, "agui-events.jsonl")
		return deriveStatusFromFile(path)
	}

	// Create 20 "running" sessions (simulates a list page with 20 running)
	sessions := make([]*types.AgenticSession, 20)
	for i := 0; i < 20; i++ {
		sessions[i] = &types.AgenticSession{
			Metadata: map[string]interface{}{"name": "bench-session"},
			Status:   &types.AgenticSessionStatus{Phase: "Running"},
		}
	}

	// Prime cache
	enrichAgentStatus(sessions[0])

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// Each goroutine gets its own session to avoid racing on AgentStatus mutation
		session := &types.AgenticSession{
			Metadata: map[string]interface{}{"name": "bench-session"},
			Status:   &types.AgenticSessionStatus{Phase: "Running"},
		}
		for pb.Next() {
			enrichAgentStatus(session)
		}
	})
}

// BenchmarkEnrichAgentStatus_ListPage simulates enriching all sessions in a
// paginated list response (20 running sessions, as the frontend would see).
func BenchmarkEnrichAgentStatus_ListPage(b *testing.B) {
	stateDir, _ := setupEventLog(b, 10000)

	origDerive := DeriveAgentStatusFromEvents
	defer func() { DeriveAgentStatusFromEvents = origDerive }()

	DeriveAgentStatusFromEvents = func(name string) string {
		path := filepath.Join(stateDir, "sessions", name, "agui-events.jsonl")
		return deriveStatusFromFile(path)
	}

	sessions := make([]types.AgenticSession, 20)
	for i := 0; i < 20; i++ {
		sessions[i] = types.AgenticSession{
			Metadata: map[string]interface{}{"name": "bench-session"},
			Status:   &types.AgenticSessionStatus{Phase: "Running"},
		}
	}

	b.Run("uncached", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for j := range sessions {
				// Clear cache before each session to force a file scan every time
				agentStatusCache.Lock()
				delete(agentStatusCache.entries, "bench-session")
				agentStatusCache.Unlock()

				enrichAgentStatus(&sessions[j])
			}
		}
	})

	b.Run("cached", func(b *testing.B) {
		// Prime
		for j := range sessions {
			enrichAgentStatus(&sessions[j])
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j := range sessions {
				enrichAgentStatus(&sessions[j])
			}
		}
	})
}

// deriveStatusFromFile simulates DeriveAgentStatus by tail-scanning the event log.
// This mirrors the real implementation in websocket/agui_store.go:DeriveAgentStatus.
func deriveStatusFromFile(path string) string {
	const maxTailBytes = 20 * 1024 * 1024

	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return ""
	}

	fileSize := stat.Size()
	var data []byte

	if fileSize <= maxTailBytes {
		data, err = os.ReadFile(path)
		if err != nil {
			return ""
		}
	} else {
		offset := fileSize - maxTailBytes
		file.Seek(offset, 0)
		data = make([]byte, maxTailBytes)
		n, _ := file.Read(data)
		data = data[:n]
	}

	// Scan backwards for lifecycle events
	lines := splitTailLines(data)
	for i := len(lines) - 1; i >= 0; i-- {
		if len(lines[i]) == 0 {
			continue
		}
		var evt map[string]interface{}
		if err := json.Unmarshal(lines[i], &evt); err != nil {
			continue
		}
		evtType, _ := evt["type"].(string)
		switch evtType {
		case "RUN_STARTED":
			return "working"
		case "RUN_FINISHED", "RUN_ERROR":
			return "idle"
		}
	}
	return ""
}

func splitTailLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

// TestAgentStatusCache_Correctness verifies the cache behaves correctly.
func TestAgentStatusCache_Correctness(t *testing.T) {
	callCount := 0
	origDerive := DeriveAgentStatusFromEvents
	defer func() { DeriveAgentStatusFromEvents = origDerive }()

	DeriveAgentStatusFromEvents = func(name string) string {
		callCount++
		return "working"
	}

	session := &types.AgenticSession{
		Metadata: map[string]interface{}{"name": "test-session"},
		Status:   &types.AgenticSessionStatus{Phase: "Running"},
	}

	// Clear cache
	agentStatusCache.Lock()
	agentStatusCache.entries = make(map[string]agentStatusCacheEntry)
	agentStatusCache.Unlock()

	// First call — cache miss, should call DeriveAgentStatusFromEvents
	enrichAgentStatus(session)
	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}

	// Second call within TTL — cache hit, should NOT call again
	enrichAgentStatus(session)
	if callCount != 1 {
		t.Fatalf("expected 1 call (cached), got %d", callCount)
	}

	// Verify status was set
	if *session.Status.AgentStatus != "working" {
		t.Fatalf("expected 'working', got %q", *session.Status.AgentStatus)
	}

	// Non-running session should skip cache entirely
	stopped := &types.AgenticSession{
		Metadata: map[string]interface{}{"name": "stopped-session"},
		Status:   &types.AgenticSessionStatus{Phase: "Stopped"},
	}
	enrichAgentStatus(stopped)
	if callCount != 1 {
		t.Fatalf("expected no call for stopped session, got %d", callCount)
	}

	// Concurrent safety
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := &types.AgenticSession{
				Metadata: map[string]interface{}{"name": "test-session"},
				Status:   &types.AgenticSessionStatus{Phase: "Running"},
			}
			enrichAgentStatus(s)
		}()
	}
	wg.Wait()
}
