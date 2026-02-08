package tracker

import (
	"testing"
	"time"
)

func TestDetectConflicts(t *testing.T) {
	now := time.Now()

	changes := []RecordedFileChange{
		{
			Timestamp: now.Add(-10 * time.Minute),
			Session:   "s1",
			Agents:    []string{"cc_1"},
			Change: FileChange{
				Path: "/src/api.go",
				Type: FileModified,
			},
		},
		{
			Timestamp: now.Add(-5 * time.Minute),
			Session:   "s1",
			Agents:    []string{"cod_1"},
			Change: FileChange{
				Path: "/src/api.go",
				Type: FileModified,
			},
		},
		{
			Timestamp: now.Add(-2 * time.Minute),
			Session:   "s1",
			Agents:    []string{"cc_1"}, // Same agent as first, but different from second
			Change: FileChange{
				Path: "/src/api.go",
				Type: FileModified,
			},
		},
		{
			Timestamp: now.Add(-1 * time.Minute),
			Session:   "s1",
			Agents:    []string{"cc_1"},
			Change: FileChange{
				Path: "/src/other.go",
				Type: FileModified,
			},
		},
	}

	conflicts := DetectConflicts(changes)

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}

	c := conflicts[0]
	if c.Path != "/src/api.go" {
		t.Errorf("expected conflict in /src/api.go, got %s", c.Path)
	}
	if len(c.Changes) != 3 {
		t.Errorf("expected 3 conflicting changes, got %d", len(c.Changes))
	}

	if c.Severity != "critical" {
		t.Errorf("expected critical severity due to tight timing, got %s", c.Severity)
	}
	if c.LastAt.IsZero() {
		t.Errorf("expected LastAt to be set")
	}
}

func TestNoConflictSameAgent(t *testing.T) {
	now := time.Now()

	changes := []RecordedFileChange{
		{
			Timestamp: now.Add(-10 * time.Minute),
			Session:   "s1",
			Agents:    []string{"cc_1"},
			Change: FileChange{
				Path: "/src/api.go",
				Type: FileModified,
			},
		},
		{
			Timestamp: now.Add(-5 * time.Minute),
			Session:   "s1",
			Agents:    []string{"cc_1"},
			Change: FileChange{
				Path: "/src/api.go",
				Type: FileModified,
			},
		},
	}

	conflicts := DetectConflicts(changes)

	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(conflicts))
	}
}

func TestConflictSeverityAgentCount(t *testing.T) {
	now := time.Now()
	changes := []RecordedFileChange{
		{Timestamp: now.Add(-3 * time.Minute), Session: "s1", Agents: []string{"a1"}, Change: FileChange{Path: "/p", Type: FileModified}},
		{Timestamp: now.Add(-2 * time.Minute), Session: "s1", Agents: []string{"a2"}, Change: FileChange{Path: "/p", Type: FileModified}},
		{Timestamp: now.Add(-1 * time.Minute), Session: "s1", Agents: []string{"a3"}, Change: FileChange{Path: "/p", Type: FileModified}},
	}
	conflicts := DetectConflicts(changes)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Severity != "critical" {
		t.Errorf("expected critical severity with 3 agents, got %s", conflicts[0].Severity)
	}
}

// =============================================================================
// Additional Tests for Coverage Improvement
// =============================================================================

func TestConflictSeverity_Warning(t *testing.T) {
	t.Parallel()
	now := time.Now()

	// Two agents, edits spread over more than CriticalConflictWindow (10 min)
	changes := []RecordedFileChange{
		{Timestamp: now.Add(-20 * time.Minute), Session: "s1", Agents: []string{"a1"}, Change: FileChange{Path: "/p", Type: FileModified}},
		{Timestamp: now.Add(-5 * time.Minute), Session: "s1", Agents: []string{"a2"}, Change: FileChange{Path: "/p", Type: FileModified}},
	}
	conflicts := DetectConflicts(changes)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Severity != "warning" {
		t.Errorf("expected warning severity (2 agents, >10min window), got %s", conflicts[0].Severity)
	}
}

func TestConflictSeverity_CriticalTightWindow(t *testing.T) {
	t.Parallel()
	now := time.Now()

	// Two agents, edits within CriticalConflictWindow (10 min)
	changes := []RecordedFileChange{
		{Timestamp: now.Add(-5 * time.Minute), Session: "s1", Agents: []string{"a1"}, Change: FileChange{Path: "/p", Type: FileModified}},
		{Timestamp: now.Add(-3 * time.Minute), Session: "s1", Agents: []string{"a2"}, Change: FileChange{Path: "/p", Type: FileModified}},
	}
	conflicts := DetectConflicts(changes)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Severity != "critical" {
		t.Errorf("expected critical severity (tight window), got %s", conflicts[0].Severity)
	}
}

func TestDetectConflicts_FileAdded(t *testing.T) {
	t.Parallel()
	now := time.Now()

	changes := []RecordedFileChange{
		{Timestamp: now.Add(-5 * time.Minute), Session: "s1", Agents: []string{"a1"}, Change: FileChange{Path: "/new", Type: FileAdded}},
		{Timestamp: now.Add(-3 * time.Minute), Session: "s1", Agents: []string{"a2"}, Change: FileChange{Path: "/new", Type: FileModified}},
	}
	conflicts := DetectConflicts(changes)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict for added+modified, got %d", len(conflicts))
	}
}

func TestDetectConflicts_FileDeleted(t *testing.T) {
	t.Parallel()
	now := time.Now()

	changes := []RecordedFileChange{
		{Timestamp: now.Add(-5 * time.Minute), Session: "s1", Agents: []string{"a1"}, Change: FileChange{Path: "/del", Type: FileModified}},
		{Timestamp: now.Add(-3 * time.Minute), Session: "s1", Agents: []string{"a2"}, Change: FileChange{Path: "/del", Type: FileDeleted}},
	}
	conflicts := DetectConflicts(changes)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict for modify+delete, got %d", len(conflicts))
	}
}

func TestDetectConflicts_NoConflictSingleChange(t *testing.T) {
	t.Parallel()
	now := time.Now()

	changes := []RecordedFileChange{
		{Timestamp: now, Session: "s1", Agents: []string{"a1"}, Change: FileChange{Path: "/single", Type: FileModified}},
	}
	conflicts := DetectConflicts(changes)
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts for single change, got %d", len(conflicts))
	}
}

func TestDetectConflicts_Empty(t *testing.T) {
	t.Parallel()
	conflicts := DetectConflicts([]RecordedFileChange{})
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts for empty input, got %d", len(conflicts))
	}
}

// =============================================================================
// Global wrapper tests: DetectConflictsRecent, ConflictsSince
// =============================================================================

func TestDetectConflictsRecent(t *testing.T) {
	// Not parallel: modifies package-level GlobalFileChanges
	origStore := GlobalFileChanges
	store := NewFileChangeStore(100)
	GlobalFileChanges = store
	t.Cleanup(func() { GlobalFileChanges = origStore })

	now := time.Now()

	// Record changes from two agents within the last 5 minutes
	store.Add(RecordedFileChange{
		Timestamp: now.Add(-3 * time.Minute),
		Session:   "s1",
		Agents:    []string{"a1"},
		Change:    FileChange{Path: "/src/main.go", Type: FileModified},
	})
	store.Add(RecordedFileChange{
		Timestamp: now.Add(-2 * time.Minute),
		Session:   "s1",
		Agents:    []string{"a2"},
		Change:    FileChange{Path: "/src/main.go", Type: FileModified},
	})

	// Detect conflicts in the last 10 minutes
	conflicts := DetectConflictsRecent(10 * time.Minute)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Path != "/src/main.go" {
		t.Errorf("expected conflict on /src/main.go, got %s", conflicts[0].Path)
	}
}

func TestDetectConflictsRecent_NoConflicts(t *testing.T) {
	origStore := GlobalFileChanges
	store := NewFileChangeStore(100)
	GlobalFileChanges = store
	t.Cleanup(func() { GlobalFileChanges = origStore })

	now := time.Now()

	// Single agent - no conflict
	store.Add(RecordedFileChange{
		Timestamp: now.Add(-1 * time.Minute),
		Session:   "s1",
		Agents:    []string{"a1"},
		Change:    FileChange{Path: "/src/main.go", Type: FileModified},
	})

	conflicts := DetectConflictsRecent(10 * time.Minute)
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(conflicts))
	}
}

func TestConflictsSince(t *testing.T) {
	origStore := GlobalFileChanges
	store := NewFileChangeStore(100)
	GlobalFileChanges = store
	t.Cleanup(func() { GlobalFileChanges = origStore })

	now := time.Now()

	store.Add(RecordedFileChange{
		Timestamp: now.Add(-5 * time.Minute),
		Session:   "s1",
		Agents:    []string{"a1"},
		Change:    FileChange{Path: "/src/api.go", Type: FileModified},
	})
	store.Add(RecordedFileChange{
		Timestamp: now.Add(-3 * time.Minute),
		Session:   "s1",
		Agents:    []string{"a2"},
		Change:    FileChange{Path: "/src/api.go", Type: FileModified},
	})
	store.Add(RecordedFileChange{
		Timestamp: now.Add(-2 * time.Minute),
		Session:   "s2",
		Agents:    []string{"a3"},
		Change:    FileChange{Path: "/src/api.go", Type: FileModified},
	})

	// Filter by session s1 — should get a conflict (a1 and a2)
	conflicts := ConflictsSince(now.Add(-10*time.Minute), "s1")
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict for session s1, got %d", len(conflicts))
	}
	if len(conflicts[0].Changes) != 2 {
		t.Errorf("expected 2 changes for session s1, got %d", len(conflicts[0].Changes))
	}
}

func TestConflictsSince_EmptySession(t *testing.T) {
	origStore := GlobalFileChanges
	store := NewFileChangeStore(100)
	GlobalFileChanges = store
	t.Cleanup(func() { GlobalFileChanges = origStore })

	now := time.Now()

	store.Add(RecordedFileChange{
		Timestamp: now.Add(-3 * time.Minute),
		Session:   "s1",
		Agents:    []string{"a1"},
		Change:    FileChange{Path: "/src/api.go", Type: FileModified},
	})
	store.Add(RecordedFileChange{
		Timestamp: now.Add(-2 * time.Minute),
		Session:   "s2",
		Agents:    []string{"a2"},
		Change:    FileChange{Path: "/src/api.go", Type: FileModified},
	})

	// Empty session means no filter — include all sessions
	conflicts := ConflictsSince(now.Add(-10*time.Minute), "")
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict with empty session filter, got %d", len(conflicts))
	}
	if len(conflicts[0].Changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(conflicts[0].Changes))
	}
}

// =============================================================================
// Global wrapper tests: RecordedChangesSince, RecordedChanges
// =============================================================================

func TestRecordedChangesSince(t *testing.T) {
	origStore := GlobalFileChanges
	store := NewFileChangeStore(100)
	GlobalFileChanges = store
	t.Cleanup(func() { GlobalFileChanges = origStore })

	now := time.Now()

	store.Add(RecordedFileChange{
		Timestamp: now.Add(-10 * time.Minute),
		Session:   "s1",
		Agents:    []string{"a1"},
		Change:    FileChange{Path: "/old.go", Type: FileModified},
	})
	store.Add(RecordedFileChange{
		Timestamp: now.Add(-1 * time.Minute),
		Session:   "s1",
		Agents:    []string{"a2"},
		Change:    FileChange{Path: "/new.go", Type: FileAdded},
	})

	changes := RecordedChangesSince(now.Add(-5 * time.Minute))
	if len(changes) != 1 {
		t.Fatalf("expected 1 change since 5min ago, got %d", len(changes))
	}
	if changes[0].Change.Path != "/new.go" {
		t.Errorf("expected /new.go, got %s", changes[0].Change.Path)
	}
}

func TestRecordedChanges(t *testing.T) {
	origStore := GlobalFileChanges
	store := NewFileChangeStore(100)
	GlobalFileChanges = store
	t.Cleanup(func() { GlobalFileChanges = origStore })

	now := time.Now()

	store.Add(RecordedFileChange{
		Timestamp: now.Add(-5 * time.Minute),
		Session:   "s1",
		Agents:    []string{"a1"},
		Change:    FileChange{Path: "/file1.go", Type: FileModified},
	})
	store.Add(RecordedFileChange{
		Timestamp: now.Add(-1 * time.Minute),
		Session:   "s1",
		Agents:    []string{"a2"},
		Change:    FileChange{Path: "/file2.go", Type: FileAdded},
	})

	all := RecordedChanges()
	if len(all) != 2 {
		t.Fatalf("expected 2 changes total, got %d", len(all))
	}
}

func TestRecordedChanges_Empty(t *testing.T) {
	origStore := GlobalFileChanges
	store := NewFileChangeStore(100)
	GlobalFileChanges = store
	t.Cleanup(func() { GlobalFileChanges = origStore })

	all := RecordedChanges()
	if len(all) != 0 {
		t.Errorf("expected 0 changes from empty store, got %d", len(all))
	}
}

func TestConflictSeverityFunction(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name       string
		changes    []RecordedFileChange
		agentCount int
		want       string
	}{
		{
			name:       "3+ agents is critical",
			changes:    []RecordedFileChange{{Timestamp: now}, {Timestamp: now}},
			agentCount: 3,
			want:       "critical",
		},
		{
			name: "2 agents tight window is critical",
			changes: []RecordedFileChange{
				{Timestamp: now},
				{Timestamp: now.Add(5 * time.Minute)},
			},
			agentCount: 2,
			want:       "critical",
		},
		{
			name: "2 agents wide window is warning",
			changes: []RecordedFileChange{
				{Timestamp: now},
				{Timestamp: now.Add(15 * time.Minute)},
			},
			agentCount: 2,
			want:       "warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := conflictSeverity(tt.changes, tt.agentCount)
			if got != tt.want {
				t.Errorf("conflictSeverity() = %q, want %q", got, tt.want)
			}
		})
	}
}
