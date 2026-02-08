package tracker

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tracker := New()
	if tracker == nil {
		t.Fatal("New() returned nil")
	}
	if tracker.maxSize != DefaultMaxSize {
		t.Errorf("expected maxSize %d, got %d", DefaultMaxSize, tracker.maxSize)
	}
	if tracker.maxAge != DefaultMaxAge {
		t.Errorf("expected maxAge %v, got %v", DefaultMaxAge, tracker.maxAge)
	}
}

func TestNewWithConfig(t *testing.T) {
	tracker := NewWithConfig(100, 1*time.Minute)
	if tracker.maxSize != 100 {
		t.Errorf("expected maxSize 100, got %d", tracker.maxSize)
	}
	if tracker.maxAge != 1*time.Minute {
		t.Errorf("expected maxAge 1m, got %v", tracker.maxAge)
	}
}

func TestNewWithConfigDefaults(t *testing.T) {
	// Test that invalid values get defaults
	tracker := NewWithConfig(-1, -1)
	if tracker.maxSize != DefaultMaxSize {
		t.Errorf("expected default maxSize, got %d", tracker.maxSize)
	}
	if tracker.maxAge != DefaultMaxAge {
		t.Errorf("expected default maxAge, got %v", tracker.maxAge)
	}
}

func TestRecord(t *testing.T) {
	tracker := New()

	change := StateChange{
		Type:    ChangeAgentOutput,
		Session: "test-session",
		Pane:    "test-pane",
		Details: map[string]interface{}{"key": "value"},
	}

	tracker.Record(change)

	if tracker.Count() != 1 {
		t.Errorf("expected count 1, got %d", tracker.Count())
	}

	changes := tracker.All()
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].Type != ChangeAgentOutput {
		t.Errorf("expected type %s, got %s", ChangeAgentOutput, changes[0].Type)
	}
	if changes[0].Session != "test-session" {
		t.Errorf("expected session 'test-session', got %s", changes[0].Session)
	}
}

func TestRecordSetsTimestamp(t *testing.T) {
	tracker := New()

	before := time.Now()
	tracker.Record(StateChange{Type: ChangeAgentState})
	after := time.Now()

	changes := tracker.All()
	if len(changes) != 1 {
		t.Fatal("expected 1 change")
	}

	if changes[0].Timestamp.Before(before) || changes[0].Timestamp.After(after) {
		t.Error("timestamp should be set automatically")
	}
}

func TestSince(t *testing.T) {
	tracker := New()

	// Record some changes with known times
	t1 := time.Now()
	tracker.Record(StateChange{
		Timestamp: t1.Add(-3 * time.Second),
		Type:      ChangeAgentOutput,
		Session:   "s1",
	})
	tracker.Record(StateChange{
		Timestamp: t1.Add(-1 * time.Second),
		Type:      ChangeAgentState,
		Session:   "s1",
	})
	tracker.Record(StateChange{
		Timestamp: t1,
		Type:      ChangeAlert,
		Session:   "s1",
	})

	// Get changes since 2 seconds ago
	changes := tracker.Since(t1.Add(-2 * time.Second))
	if len(changes) != 2 {
		t.Errorf("expected 2 changes since -2s, got %d", len(changes))
	}
}

func TestMaxSize(t *testing.T) {
	tracker := NewWithConfig(3, 1*time.Hour)

	for i := 0; i < 5; i++ {
		tracker.Record(StateChange{
			Type:    ChangeAgentOutput,
			Session: "s1",
			Details: map[string]interface{}{"i": i},
		})
	}

	if tracker.Count() != 3 {
		t.Errorf("expected count 3 (maxSize), got %d", tracker.Count())
	}

	// The oldest two should be gone (i=0, i=1)
	changes := tracker.All()
	for _, c := range changes {
		idx := c.Details["i"].(int)
		if idx < 2 {
			t.Errorf("expected oldest entries to be pruned, found i=%d", idx)
		}
	}
}

func TestMaxAge(t *testing.T) {
	// Use very short maxAge for testing
	tracker := NewWithConfig(100, 50*time.Millisecond)

	// Record an old change
	tracker.Record(StateChange{
		Timestamp: time.Now().Add(-100 * time.Millisecond),
		Type:      ChangeAgentOutput,
		Session:   "old",
	})

	// Wait a bit and record a new change (triggers pruning)
	time.Sleep(10 * time.Millisecond)
	tracker.Record(StateChange{
		Type:    ChangeAgentState,
		Session: "new",
	})

	changes := tracker.All()
	if len(changes) != 1 {
		t.Errorf("expected 1 change (old should be pruned), got %d", len(changes))
	}
	if len(changes) > 0 && changes[0].Session != "new" {
		t.Error("expected 'new' session to remain")
	}
}

func TestClear(t *testing.T) {
	tracker := New()
	tracker.Record(StateChange{Type: ChangeAgentOutput})
	tracker.Record(StateChange{Type: ChangeAgentState})

	if tracker.Count() != 2 {
		t.Errorf("expected 2 before clear, got %d", tracker.Count())
	}

	tracker.Clear()

	if tracker.Count() != 0 {
		t.Errorf("expected 0 after clear, got %d", tracker.Count())
	}
}

func TestCoalesce(t *testing.T) {
	tracker := New()

	// Add consecutive changes of same type for same pane
	now := time.Now()
	tracker.Record(StateChange{Timestamp: now, Type: ChangeAgentOutput, Session: "s1", Pane: "p1"})
	tracker.Record(StateChange{Timestamp: now.Add(1 * time.Second), Type: ChangeAgentOutput, Session: "s1", Pane: "p1"})
	tracker.Record(StateChange{Timestamp: now.Add(2 * time.Second), Type: ChangeAgentOutput, Session: "s1", Pane: "p1"})
	// Different type
	tracker.Record(StateChange{Timestamp: now.Add(3 * time.Second), Type: ChangeAgentState, Session: "s1", Pane: "p1"})
	// Back to first type
	tracker.Record(StateChange{Timestamp: now.Add(4 * time.Second), Type: ChangeAgentOutput, Session: "s1", Pane: "p1"})

	coalesced := tracker.Coalesce()
	if len(coalesced) != 3 {
		t.Errorf("expected 3 coalesced groups, got %d", len(coalesced))
	}

	// First group: 3 agent_output changes
	if coalesced[0].Type != ChangeAgentOutput || coalesced[0].Count != 3 {
		t.Errorf("expected first group: 3 agent_output, got %s count %d", coalesced[0].Type, coalesced[0].Count)
	}

	// Second group: 1 agent_state change
	if coalesced[1].Type != ChangeAgentState || coalesced[1].Count != 1 {
		t.Errorf("expected second group: 1 agent_state, got %s count %d", coalesced[1].Type, coalesced[1].Count)
	}

	// Third group: 1 agent_output change
	if coalesced[2].Type != ChangeAgentOutput || coalesced[2].Count != 1 {
		t.Errorf("expected third group: 1 agent_output, got %s count %d", coalesced[2].Type, coalesced[2].Count)
	}
}

func TestSinceByType(t *testing.T) {
	tracker := New()

	now := time.Now()
	tracker.Record(StateChange{Timestamp: now.Add(-2 * time.Second), Type: ChangeAgentOutput, Session: "s1"})
	tracker.Record(StateChange{Timestamp: now.Add(-1 * time.Second), Type: ChangeAgentState, Session: "s1"})
	tracker.Record(StateChange{Timestamp: now, Type: ChangeAgentOutput, Session: "s1"})

	changes := tracker.SinceByType(now.Add(-3*time.Second), ChangeAgentOutput)
	if len(changes) != 2 {
		t.Errorf("expected 2 agent_output changes, got %d", len(changes))
	}
}

func TestSinceBySession(t *testing.T) {
	tracker := New()

	now := time.Now()
	tracker.Record(StateChange{Timestamp: now.Add(-2 * time.Second), Type: ChangeAgentOutput, Session: "s1"})
	tracker.Record(StateChange{Timestamp: now.Add(-1 * time.Second), Type: ChangeAgentState, Session: "s2"})
	tracker.Record(StateChange{Timestamp: now, Type: ChangeAgentOutput, Session: "s1"})

	changes := tracker.SinceBySession(now.Add(-3*time.Second), "s1")
	if len(changes) != 2 {
		t.Errorf("expected 2 s1 changes, got %d", len(changes))
	}
}

func TestHelperFunctions(t *testing.T) {
	tracker := New()

	tracker.RecordAgentOutput("sess", "pane1", "hello world")
	tracker.RecordAgentState("sess", "pane1", "idle")
	tracker.RecordAlert("sess", "pane1", "error", "something went wrong")
	tracker.RecordPaneCreated("sess", "pane2", "claude")
	tracker.RecordSessionCreated("sess2")

	if tracker.Count() != 5 {
		t.Errorf("expected 5 changes from helpers, got %d", tracker.Count())
	}

	changes := tracker.All()

	// Check output change
	if changes[0].Type != ChangeAgentOutput {
		t.Error("first change should be agent_output")
	}
	if changes[0].Details["output_length"].(int) != 11 {
		t.Error("output length should be 11")
	}

	// Check state change
	if changes[1].Type != ChangeAgentState {
		t.Error("second change should be agent_state")
	}
	if changes[1].Details["state"].(string) != "idle" {
		t.Error("state should be 'idle'")
	}

	// Check alert
	if changes[2].Type != ChangeAlert {
		t.Error("third change should be alert")
	}
	if changes[2].Details["message"].(string) != "something went wrong" {
		t.Error("alert message mismatch")
	}

	// Check pane created
	if changes[3].Type != ChangePaneCreated {
		t.Error("fourth change should be pane_created")
	}
	if changes[3].Details["agent_type"].(string) != "claude" {
		t.Error("agent_type should be 'claude'")
	}

	// Check session created
	if changes[4].Type != ChangeSessionCreated {
		t.Error("fifth change should be session_created")
	}
}

func TestConcurrency(t *testing.T) {
	tracker := New()
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			tracker.Record(StateChange{Type: ChangeAgentOutput, Session: "test"})
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = tracker.All()
			_ = tracker.Count()
			_ = tracker.Since(time.Now().Add(-1 * time.Hour))
		}
		done <- true
	}()

	<-done
	<-done

	// If we get here without deadlock or panic, concurrency is working
}

func TestPrune(t *testing.T) {
	tracker := NewWithConfig(100, 50*time.Millisecond)

	// Add old entries
	tracker.Record(StateChange{
		Timestamp: time.Now().Add(-100 * time.Millisecond),
		Type:      ChangeAgentOutput,
	})
	tracker.Record(StateChange{
		Timestamp: time.Now().Add(-80 * time.Millisecond),
		Type:      ChangeAgentOutput,
	})

	// Manually prune
	tracker.Prune()

	if tracker.Count() != 0 {
		t.Errorf("expected 0 after prune (all old), got %d", tracker.Count())
	}
}

// =============================================================================
// Additional Pure Function Tests for Coverage Improvement
// =============================================================================

func TestScanNullTerminated(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		data    []byte
		atEOF   bool
		advance int
		token   []byte
		wantNil bool
	}{
		{
			name:    "empty at EOF",
			data:    []byte{},
			atEOF:   true,
			advance: 0,
			token:   nil,
			wantNil: true,
		},
		{
			name:    "single token with null",
			data:    []byte("hello\x00"),
			atEOF:   false,
			advance: 6,
			token:   []byte("hello"),
		},
		{
			name:    "multiple tokens",
			data:    []byte("first\x00second"),
			atEOF:   false,
			advance: 6,
			token:   []byte("first"),
		},
		{
			name:    "final token at EOF no null",
			data:    []byte("final"),
			atEOF:   true,
			advance: 5,
			token:   []byte("final"),
		},
		{
			name:    "need more data",
			data:    []byte("incomplete"),
			atEOF:   false,
			advance: 0,
			token:   nil,
			wantNil: true,
		},
		{
			name:    "empty token",
			data:    []byte("\x00rest"),
			atEOF:   false,
			advance: 1,
			token:   []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			advance, token, err := scanNullTerminated(tt.data, tt.atEOF)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if advance != tt.advance {
				t.Errorf("advance = %d, want %d", advance, tt.advance)
			}
			if tt.wantNil {
				if token != nil {
					t.Errorf("token = %v, want nil", token)
				}
			} else {
				if string(token) != string(tt.token) {
					t.Errorf("token = %q, want %q", token, tt.token)
				}
			}
		})
	}
}

func TestCoalesce_Empty(t *testing.T) {
	tracker := New()
	coalesced := tracker.Coalesce()
	if coalesced != nil {
		t.Errorf("expected nil for empty tracker, got %v", coalesced)
	}
}

func TestSince_WithDetails(t *testing.T) {
	tracker := New()

	now := time.Now()
	details := map[string]interface{}{"key": "value", "num": 42}
	tracker.Record(StateChange{
		Timestamp: now.Add(-1 * time.Second),
		Type:      ChangeAgentOutput,
		Details:   details,
	})

	changes := tracker.Since(now.Add(-2 * time.Second))
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	// Verify deep copy - modifying returned details shouldn't affect original
	changes[0].Details["key"] = "modified"

	original := tracker.All()
	if original[0].Details["key"] != "value" {
		t.Error("original details should not be modified")
	}
}

func TestSince_NoDetails(t *testing.T) {
	tracker := New()

	now := time.Now()
	tracker.Record(StateChange{
		Timestamp: now.Add(-1 * time.Second),
		Type:      ChangeAgentOutput,
		// No Details
	})

	changes := tracker.Since(now.Add(-2 * time.Second))
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].Details != nil {
		t.Errorf("expected nil details, got %v", changes[0].Details)
	}
}

func TestSinceByType_WithDetails(t *testing.T) {
	tracker := New()

	now := time.Now()
	tracker.Record(StateChange{
		Timestamp: now.Add(-1 * time.Second),
		Type:      ChangeAgentOutput,
		Details:   map[string]interface{}{"key": "value"},
	})

	changes := tracker.SinceByType(now.Add(-2*time.Second), ChangeAgentOutput)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	// Verify deep copy
	changes[0].Details["key"] = "modified"
	original := tracker.All()
	if original[0].Details["key"] != "value" {
		t.Error("original details should not be modified")
	}
}

func TestSinceBySession_WithDetails(t *testing.T) {
	tracker := New()

	now := time.Now()
	tracker.Record(StateChange{
		Timestamp: now.Add(-1 * time.Second),
		Type:      ChangeAgentOutput,
		Session:   "test-session",
		Details:   map[string]interface{}{"key": "value"},
	})

	changes := tracker.SinceBySession(now.Add(-2*time.Second), "test-session")
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	// Verify deep copy
	changes[0].Details["key"] = "modified"
	original := tracker.All()
	if original[0].Details["key"] != "value" {
		t.Error("original details should not be modified")
	}
}

func TestFileChangeStore_Since(t *testing.T) {
	store := NewFileChangeStore(10)

	now := time.Now()
	store.Add(RecordedFileChange{Timestamp: now.Add(-3 * time.Second), Session: "s1"})
	store.Add(RecordedFileChange{Timestamp: now.Add(-2 * time.Second), Session: "s2"})
	store.Add(RecordedFileChange{Timestamp: now.Add(-1 * time.Second), Session: "s3"})

	changes := store.Since(now.Add(-2500 * time.Millisecond))
	if len(changes) != 2 {
		t.Errorf("expected 2 changes since -2.5s, got %d", len(changes))
	}
	if len(changes) >= 2 {
		if changes[0].Session != "s2" || changes[1].Session != "s3" {
			t.Errorf("got sessions %s, %s; want s2, s3", changes[0].Session, changes[1].Session)
		}
	}
}

func TestFileChangeStore_Since_Empty(t *testing.T) {
	store := NewFileChangeStore(10)
	changes := store.Since(time.Now().Add(-1 * time.Hour))
	if len(changes) != 0 {
		t.Errorf("expected 0 changes for empty store, got %d", len(changes))
	}
}

func TestFileChangeStore_Since_Wrapped(t *testing.T) {
	store := NewFileChangeStore(3)

	now := time.Now()
	// Fill store and wrap
	store.Add(RecordedFileChange{Timestamp: now.Add(-4 * time.Second), Session: "s1"})
	store.Add(RecordedFileChange{Timestamp: now.Add(-3 * time.Second), Session: "s2"})
	store.Add(RecordedFileChange{Timestamp: now.Add(-2 * time.Second), Session: "s3"})
	store.Add(RecordedFileChange{Timestamp: now.Add(-1 * time.Second), Session: "s4"}) // Wraps, s1 gone

	changes := store.Since(now.Add(-5 * time.Second))
	if len(changes) != 3 {
		t.Errorf("expected 3 changes after wrap, got %d", len(changes))
	}
}

func TestFileChangeStore_Add_ZeroLimit(t *testing.T) {
	store := NewFileChangeStore(0) // Gets default 500
	store.Add(RecordedFileChange{Session: "test"})

	all := store.All()
	if len(all) != 1 {
		t.Errorf("expected 1 entry, got %d", len(all))
	}
}

func TestFileChangeStore_Add_SetsTimestamp(t *testing.T) {
	store := NewFileChangeStore(10)

	before := time.Now()
	store.Add(RecordedFileChange{Session: "test"}) // No timestamp
	after := time.Now()

	all := store.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(all))
	}

	if all[0].Timestamp.Before(before) || all[0].Timestamp.After(after) {
		t.Error("timestamp should be auto-set when zero")
	}
}

func TestFileChangeStore_All_NotWrapped(t *testing.T) {
	store := NewFileChangeStore(10)
	store.Add(RecordedFileChange{Session: "s1"})
	store.Add(RecordedFileChange{Session: "s2"})

	all := store.All()
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}
	if all[0].Session != "s1" || all[1].Session != "s2" {
		t.Error("order should be preserved")
	}
}
