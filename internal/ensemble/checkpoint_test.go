package ensemble

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCheckpointStore(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}
	if store == nil {
		t.Fatal("store is nil")
	}

	// Verify directory was created
	checkpointDir := filepath.Join(tmpDir, checkpointDirName)
	if _, err := os.Stat(checkpointDir); os.IsNotExist(err) {
		t.Error("checkpoint directory was not created")
	}

	t.Logf("TEST: %s - assertion: checkpoint store created successfully", t.Name())
}

func TestCheckpointStore_SaveAndLoadCheckpoint(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	runID := "test-run-1"
	checkpoint := ModeCheckpoint{
		ModeID: "deductive",
		Output: &ModeOutput{
			ModeID: "deductive",
			Thesis: "Test thesis",
		},
		Status:      string(AssignmentDone),
		CapturedAt:  time.Now().UTC(),
		ContextHash: "abc123",
		TokensUsed:  1000,
	}

	// Save checkpoint
	if err := store.SaveCheckpoint(runID, checkpoint); err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	// Load checkpoint
	loaded, err := store.LoadCheckpoint(runID, "deductive")
	if err != nil {
		t.Fatalf("LoadCheckpoint failed: %v", err)
	}

	if loaded.ModeID != checkpoint.ModeID {
		t.Errorf("ModeID = %q, want %q", loaded.ModeID, checkpoint.ModeID)
	}
	if loaded.Status != checkpoint.Status {
		t.Errorf("Status = %q, want %q", loaded.Status, checkpoint.Status)
	}
	if loaded.TokensUsed != checkpoint.TokensUsed {
		t.Errorf("TokensUsed = %d, want %d", loaded.TokensUsed, checkpoint.TokensUsed)
	}
	if loaded.Output == nil || loaded.Output.Thesis != checkpoint.Output.Thesis {
		t.Error("Output not loaded correctly")
	}

	t.Logf("TEST: %s - assertion: checkpoint save/load works", t.Name())
}

func TestCheckpointStore_SaveAndLoadMetadata(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	meta := CheckpointMetadata{
		SessionName: "test-session",
		Question:    "What is the meaning of life?",
		RunID:       "test-run-2",
		Status:      EnsembleActive,
		CreatedAt:   time.Now().UTC(),
		ContextHash: "def456",
		PendingIDs:  []string{"deductive", "inductive"},
		TotalModes:  2,
	}

	// Save metadata
	if err := store.SaveMetadata(meta); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	// Load metadata
	loaded, err := store.LoadMetadata("test-run-2")
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}

	if loaded.SessionName != meta.SessionName {
		t.Errorf("SessionName = %q, want %q", loaded.SessionName, meta.SessionName)
	}
	if loaded.Question != meta.Question {
		t.Errorf("Question = %q, want %q", loaded.Question, meta.Question)
	}
	if len(loaded.PendingIDs) != len(meta.PendingIDs) {
		t.Errorf("PendingIDs count = %d, want %d", len(loaded.PendingIDs), len(meta.PendingIDs))
	}

	t.Logf("TEST: %s - assertion: metadata save/load works", t.Name())
}

func TestCheckpointStore_SaveAndLoadSynthesisCheckpoint(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	runID := "test-synth-run"
	checkpoint := SynthesisCheckpoint{
		RunID:       runID,
		SessionName: "test-session",
		LastIndex:   7,
		Error:       "context canceled",
		CreatedAt:   time.Now().UTC(),
	}

	if err := store.SaveSynthesisCheckpoint(runID, checkpoint); err != nil {
		t.Fatalf("SaveSynthesisCheckpoint failed: %v", err)
	}

	loaded, err := store.LoadSynthesisCheckpoint(runID)
	if err != nil {
		t.Fatalf("LoadSynthesisCheckpoint failed: %v", err)
	}

	if loaded.LastIndex != checkpoint.LastIndex {
		t.Errorf("LastIndex = %d, want %d", loaded.LastIndex, checkpoint.LastIndex)
	}
	if loaded.Error != checkpoint.Error {
		t.Errorf("Error = %q, want %q", loaded.Error, checkpoint.Error)
	}
	if loaded.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}

	t.Logf("TEST: %s - assertion: synthesis checkpoint save/load works", t.Name())
}

func TestCheckpointStore_LoadCheckpoint_NotFound(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	_, err = store.LoadCheckpoint("nonexistent-run", "nonexistent-mode")
	if err == nil {
		t.Error("expected error for nonexistent checkpoint")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}

	t.Logf("TEST: %s - assertion: not found error returned", t.Name())
}

func TestCheckpointStore_LoadAllCheckpoints(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	runID := "test-run-3"
	modes := []string{"deductive", "inductive", "causal"}

	for _, mode := range modes {
		checkpoint := ModeCheckpoint{
			ModeID: mode,
			Status: string(AssignmentDone),
		}
		if err := store.SaveCheckpoint(runID, checkpoint); err != nil {
			t.Fatalf("SaveCheckpoint failed for %s: %v", mode, err)
		}
	}

	// Load all
	checkpoints, err := store.LoadAllCheckpoints(runID)
	if err != nil {
		t.Fatalf("LoadAllCheckpoints failed: %v", err)
	}

	if len(checkpoints) != len(modes) {
		t.Errorf("got %d checkpoints, want %d", len(checkpoints), len(modes))
	}

	t.Logf("TEST: %s - assertion: all checkpoints loaded", t.Name())
}

func TestCheckpointStore_LoadAllCheckpoints_SkipsSynthesis(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	runID := "test-run-skip-synthesis"

	// Save a mode checkpoint
	modeCP := ModeCheckpoint{
		ModeID: "deductive",
		Status: string(AssignmentDone),
	}
	if err := store.SaveCheckpoint(runID, modeCP); err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	// Save a synthesis checkpoint (should be skipped by LoadAllCheckpoints)
	synthCP := SynthesisCheckpoint{
		RunID:     runID,
		LastIndex: 5,
	}
	if err := store.SaveSynthesisCheckpoint(runID, synthCP); err != nil {
		t.Fatalf("SaveSynthesisCheckpoint failed: %v", err)
	}

	// LoadAllCheckpoints should only return the mode checkpoint, not the synthesis
	checkpoints, err := store.LoadAllCheckpoints(runID)
	if err != nil {
		t.Fatalf("LoadAllCheckpoints failed: %v", err)
	}

	if len(checkpoints) != 1 {
		t.Errorf("got %d checkpoints, want 1 (synthesis.json should be skipped)", len(checkpoints))
	}
	if len(checkpoints) > 0 && checkpoints[0].ModeID != "deductive" {
		t.Errorf("expected deductive mode, got %q", checkpoints[0].ModeID)
	}

	t.Logf("TEST: %s - assertion: synthesis checkpoint correctly skipped", t.Name())
}

func TestCheckpointStore_ListRuns(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	// Create multiple runs
	runs := []string{"run-a", "run-b", "run-c"}
	for _, runID := range runs {
		meta := CheckpointMetadata{
			RunID:     runID,
			CreatedAt: time.Now().UTC(),
		}
		if err := store.SaveMetadata(meta); err != nil {
			t.Fatalf("SaveMetadata failed for %s: %v", runID, err)
		}
	}

	// List runs
	listed, err := store.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}

	if len(listed) != len(runs) {
		t.Errorf("got %d runs, want %d", len(listed), len(runs))
	}

	t.Logf("TEST: %s - assertion: all runs listed", t.Name())
}

func TestCheckpointStore_DeleteRun(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	runID := "test-run-delete"
	meta := CheckpointMetadata{
		RunID:     runID,
		CreatedAt: time.Now().UTC(),
	}
	if err := store.SaveMetadata(meta); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	// Verify exists
	if !store.RunExists(runID) {
		t.Error("run should exist before delete")
	}

	// Delete
	if err := store.DeleteRun(runID); err != nil {
		t.Fatalf("DeleteRun failed: %v", err)
	}

	// Verify gone
	if store.RunExists(runID) {
		t.Error("run should not exist after delete")
	}

	t.Logf("TEST: %s - assertion: run deleted successfully", t.Name())
}

func TestCheckpointStore_RunExists(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	if store.RunExists("nonexistent") {
		t.Error("nonexistent run should return false")
	}

	runID := "existing-run"
	meta := CheckpointMetadata{RunID: runID}
	if err := store.SaveMetadata(meta); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	if !store.RunExists(runID) {
		t.Error("existing run should return true")
	}

	t.Logf("TEST: %s - assertion: RunExists works correctly", t.Name())
}

func TestCheckpointStore_UpdateModeStatus(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	runID := "test-run-status"
	meta := CheckpointMetadata{
		RunID:      runID,
		PendingIDs: []string{"mode-a", "mode-b"},
	}
	if err := store.SaveMetadata(meta); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	// Update mode-a to done
	if err := store.UpdateModeStatus(runID, "mode-a", string(AssignmentDone)); err != nil {
		t.Fatalf("UpdateModeStatus failed: %v", err)
	}

	// Verify
	loaded, err := store.LoadMetadata(runID)
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}

	if len(loaded.CompletedIDs) != 1 || loaded.CompletedIDs[0] != "mode-a" {
		t.Errorf("CompletedIDs = %v, want [mode-a]", loaded.CompletedIDs)
	}
	if len(loaded.PendingIDs) != 1 || loaded.PendingIDs[0] != "mode-b" {
		t.Errorf("PendingIDs = %v, want [mode-b]", loaded.PendingIDs)
	}

	t.Logf("TEST: %s - assertion: mode status updated correctly", t.Name())
}

func TestCheckpointStore_GetCompletedOutputs(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	runID := "test-run-outputs"

	// Save completed checkpoint
	completedCP := ModeCheckpoint{
		ModeID: "completed-mode",
		Output: &ModeOutput{ModeID: "completed-mode", Thesis: "Done"},
		Status: string(AssignmentDone),
	}
	if err := store.SaveCheckpoint(runID, completedCP); err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	// Save error checkpoint
	errorCP := ModeCheckpoint{
		ModeID: "error-mode",
		Status: string(AssignmentError),
		Error:  "something failed",
	}
	if err := store.SaveCheckpoint(runID, errorCP); err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	// Get completed outputs
	outputs, err := store.GetCompletedOutputs(runID)
	if err != nil {
		t.Fatalf("GetCompletedOutputs failed: %v", err)
	}

	if len(outputs) != 1 {
		t.Errorf("got %d completed outputs, want 1", len(outputs))
	}
	if outputs[0].ModeID != "completed-mode" {
		t.Errorf("output ModeID = %q, want completed-mode", outputs[0].ModeID)
	}

	t.Logf("TEST: %s - assertion: only completed outputs returned", t.Name())
}

func TestCheckpointManager_Initialize(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	manager := NewCheckpointManager(store, "test-manager-run")

	session := &EnsembleSession{
		SessionName: "test-session",
		Question:    "Test question?",
		Assignments: []ModeAssignment{
			{ModeID: "mode-1"},
			{ModeID: "mode-2"},
		},
		Status:    EnsembleActive,
		CreatedAt: time.Now().UTC(),
	}

	if err := manager.Initialize(session, "context-hash-123"); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify metadata was created
	meta, err := store.LoadMetadata("test-manager-run")
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}

	if meta.SessionName != session.SessionName {
		t.Errorf("SessionName = %q, want %q", meta.SessionName, session.SessionName)
	}
	if len(meta.PendingIDs) != 2 {
		t.Errorf("PendingIDs count = %d, want 2", len(meta.PendingIDs))
	}

	t.Logf("TEST: %s - assertion: checkpoint manager initialized", t.Name())
}

func TestCheckpointManager_RecordOutput(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	runID := "test-record-run"
	manager := NewCheckpointManager(store, runID)

	// Initialize with metadata
	meta := CheckpointMetadata{
		RunID:      runID,
		PendingIDs: []string{"mode-1"},
	}
	if err := store.SaveMetadata(meta); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	// Record output
	output := &ModeOutput{
		ModeID: "mode-1",
		Thesis: "Test output",
	}
	if err := manager.RecordOutput("mode-1", output, 500, "ctx-hash"); err != nil {
		t.Fatalf("RecordOutput failed: %v", err)
	}

	// Verify checkpoint was saved
	cp, err := store.LoadCheckpoint(runID, "mode-1")
	if err != nil {
		t.Fatalf("LoadCheckpoint failed: %v", err)
	}

	if cp.Status != string(AssignmentDone) {
		t.Errorf("Status = %q, want %q", cp.Status, string(AssignmentDone))
	}
	if cp.TokensUsed != 500 {
		t.Errorf("TokensUsed = %d, want 500", cp.TokensUsed)
	}

	t.Logf("TEST: %s - assertion: output recorded successfully", t.Name())
}

func TestCheckpointManager_IsResumable(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	// Test with pending modes
	runID1 := "resumable-run"
	meta1 := CheckpointMetadata{
		RunID:      runID1,
		PendingIDs: []string{"mode-1"},
	}
	if err := store.SaveMetadata(meta1); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	manager1 := NewCheckpointManager(store, runID1)
	if !manager1.IsResumable() {
		t.Error("run with pending modes should be resumable")
	}

	// Test with all completed
	runID2 := "complete-run"
	meta2 := CheckpointMetadata{
		RunID:        runID2,
		CompletedIDs: []string{"mode-1"},
	}
	if err := store.SaveMetadata(meta2); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	manager2 := NewCheckpointManager(store, runID2)
	if manager2.IsResumable() {
		t.Error("fully completed run should not be resumable")
	}

	t.Logf("TEST: %s - assertion: IsResumable works correctly", t.Name())
}

func TestCheckpointStore_NilReceiver(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	var store *CheckpointStore

	if _, err := store.LoadCheckpoint("run", "mode"); err == nil {
		t.Error("LoadCheckpoint on nil should return error")
	}
	if _, err := store.LoadMetadata("run"); err == nil {
		t.Error("LoadMetadata on nil should return error")
	}
	if err := store.SaveCheckpoint("run", ModeCheckpoint{}); err == nil {
		t.Error("SaveCheckpoint on nil should return error")
	}
	if err := store.SaveMetadata(CheckpointMetadata{}); err == nil {
		t.Error("SaveMetadata on nil should return error")
	}
	if store.RunExists("run") {
		t.Error("RunExists on nil should return false")
	}

	t.Logf("TEST: %s - assertion: nil receiver handling works", t.Name())
}

func TestSliceContains(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	slice := []string{"a", "b", "c"}

	if !sliceContains(slice, "a") {
		t.Error("sliceContains should return true for existing item")
	}
	if !sliceContains(slice, "c") {
		t.Error("sliceContains should return true for existing item")
	}
	if sliceContains(slice, "d") {
		t.Error("sliceContains should return false for non-existing item")
	}
	if sliceContains(nil, "a") {
		t.Error("sliceContains should return false for nil slice")
	}
	if sliceContains([]string{}, "a") {
		t.Error("sliceContains should return false for empty slice")
	}

	t.Logf("TEST: %s - assertion: sliceContains works correctly", t.Name())
}

func TestRemoveFromSlice(t *testing.T) {
	t.Logf("TEST: %s - starting", t.Name())

	tests := []struct {
		slice  []string
		item   string
		expect []string
	}{
		{[]string{"a", "b", "c"}, "b", []string{"a", "c"}},
		{[]string{"a", "b", "c"}, "a", []string{"b", "c"}},
		{[]string{"a", "b", "c"}, "c", []string{"a", "b"}},
		{[]string{"a", "b", "c"}, "d", []string{"a", "b", "c"}},
		{[]string{}, "a", []string{}},
		{nil, "a", []string{}},
	}

	for _, tt := range tests {
		result := removeFromSlice(tt.slice, tt.item)
		if len(result) != len(tt.expect) {
			t.Errorf("removeFromSlice(%v, %q) = %v, want %v", tt.slice, tt.item, result, tt.expect)
		}
	}

	t.Logf("TEST: %s - assertion: removeFromSlice works correctly", t.Name())
}

func TestCheckpointStore_WithLogger(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	// Should return self for chaining
	result := store.WithLogger(nil)
	if result != store {
		t.Error("WithLogger should return store for chaining")
	}

	// Should work with non-nil logger
	result = store.WithLogger(store.logger)
	if result != store {
		t.Error("WithLogger should return store for chaining")
	}
}

func TestCheckpointStore_CleanOld(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	// Create old run - manually write metadata with old timestamp
	oldRunID := "old-run"
	oldRunDir := filepath.Join(tmpDir, checkpointDirName, oldRunID)
	if err := os.MkdirAll(oldRunDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	oldMeta := CheckpointMetadata{
		RunID:     oldRunID,
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-48 * time.Hour),
	}
	oldData, _ := json.Marshal(oldMeta)
	if err := os.WriteFile(filepath.Join(oldRunDir, checkpointMetaFile), oldData, 0o644); err != nil {
		t.Fatalf("write old metadata failed: %v", err)
	}

	// Create new run
	newMeta := CheckpointMetadata{
		RunID:     "new-run",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.SaveMetadata(newMeta); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	// Clean runs older than 24 hours
	removed, err := store.CleanOld(24 * time.Hour)
	if err != nil {
		t.Fatalf("CleanOld failed: %v", err)
	}

	if removed != 1 {
		t.Errorf("CleanOld removed %d, want 1", removed)
	}

	// Old run should be gone
	if store.RunExists(oldRunID) {
		t.Error("old run should be removed")
	}

	// New run should still exist
	if !store.RunExists("new-run") {
		t.Error("new run should still exist")
	}
}

func TestCheckpointStore_CleanOld_NilStore(t *testing.T) {
	var store *CheckpointStore
	_, err := store.CleanOld(24 * time.Hour)
	if err == nil {
		t.Error("CleanOld on nil should return error")
	}
}

func TestCheckpointStore_GetPendingModeIDs(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	runID := "pending-test"
	meta := CheckpointMetadata{
		RunID:      runID,
		PendingIDs: []string{"mode-1", "mode-2", "mode-3"},
	}
	if err := store.SaveMetadata(meta); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	pending, err := store.GetPendingModeIDs(runID)
	if err != nil {
		t.Fatalf("GetPendingModeIDs failed: %v", err)
	}

	if len(pending) != 3 {
		t.Errorf("GetPendingModeIDs returned %d, want 3", len(pending))
	}
}

func TestCheckpointManager_WithLogger(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	manager := NewCheckpointManager(store, "test-run")

	// Should return self for chaining
	result := manager.WithLogger(nil)
	if result != manager {
		t.Error("WithLogger should return manager for chaining")
	}

	// Should work with non-nil logger
	result = manager.WithLogger(manager.logger)
	if result != manager {
		t.Error("WithLogger should return manager for chaining")
	}
}

func TestCheckpointManager_RecordError(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	runID := "error-test"
	manager := NewCheckpointManager(store, runID)

	// Initialize metadata
	meta := CheckpointMetadata{
		RunID:      runID,
		PendingIDs: []string{"failing-mode"},
	}
	if err := store.SaveMetadata(meta); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	// Record error
	testErr := os.ErrNotExist
	if err := manager.RecordError("failing-mode", testErr); err != nil {
		t.Fatalf("RecordError failed: %v", err)
	}

	// Verify checkpoint was saved
	cp, err := store.LoadCheckpoint(runID, "failing-mode")
	if err != nil {
		t.Fatalf("LoadCheckpoint failed: %v", err)
	}

	if cp.Status != string(AssignmentError) {
		t.Errorf("Status = %q, want %q", cp.Status, string(AssignmentError))
	}
	if cp.Error == "" {
		t.Error("Error message should be set")
	}

	// Verify metadata was updated
	loaded, err := store.LoadMetadata(runID)
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}
	if len(loaded.ErrorIDs) != 1 || loaded.ErrorIDs[0] != "failing-mode" {
		t.Errorf("ErrorIDs = %v, want [failing-mode]", loaded.ErrorIDs)
	}
}

func TestCheckpointManager_RecordError_NilManager(t *testing.T) {
	var manager *CheckpointManager
	err := manager.RecordError("mode", os.ErrNotExist)
	if err == nil {
		t.Error("RecordError on nil should return error")
	}
}

func TestCheckpointManager_MarkComplete(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	runID := "complete-test"
	manager := NewCheckpointManager(store, runID)

	// Initialize metadata
	meta := CheckpointMetadata{
		RunID:        runID,
		Status:       EnsembleActive,
		CompletedIDs: []string{"mode-1"},
	}
	if err := store.SaveMetadata(meta); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	// Mark complete without cleanup
	if err := manager.MarkComplete(false); err != nil {
		t.Fatalf("MarkComplete failed: %v", err)
	}

	// Verify status was updated
	loaded, err := store.LoadMetadata(runID)
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}
	if loaded.Status != EnsembleComplete {
		t.Errorf("Status = %q, want %q", loaded.Status, EnsembleComplete)
	}

	// Run should still exist
	if !store.RunExists(runID) {
		t.Error("run should still exist without cleanup")
	}
}

func TestCheckpointManager_MarkComplete_WithCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	runID := "cleanup-test"
	manager := NewCheckpointManager(store, runID)

	// Initialize metadata
	meta := CheckpointMetadata{
		RunID:        runID,
		Status:       EnsembleActive,
		CompletedIDs: []string{"mode-1"},
	}
	if err := store.SaveMetadata(meta); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	// Mark complete with cleanup
	if err := manager.MarkComplete(true); err != nil {
		t.Fatalf("MarkComplete with cleanup failed: %v", err)
	}

	// Run should be deleted
	if store.RunExists(runID) {
		t.Error("run should be deleted with cleanup")
	}
}

func TestCheckpointManager_MarkComplete_NilManager(t *testing.T) {
	var manager *CheckpointManager
	err := manager.MarkComplete(false)
	if err == nil {
		t.Error("MarkComplete on nil should return error")
	}
}

func TestCheckpointManager_GetResumeState(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewCheckpointStore(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointStore failed: %v", err)
	}

	runID := "resume-test"
	manager := NewCheckpointManager(store, runID)

	// Initialize metadata
	meta := CheckpointMetadata{
		RunID:        runID,
		SessionName:  "test-session",
		Question:     "Test question?",
		PendingIDs:   []string{"mode-2"},
		CompletedIDs: []string{"mode-1"},
	}
	if err := store.SaveMetadata(meta); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	// Save a completed checkpoint
	cp := ModeCheckpoint{
		ModeID: "mode-1",
		Output: &ModeOutput{ModeID: "mode-1", Thesis: "Result"},
		Status: string(AssignmentDone),
	}
	if err := store.SaveCheckpoint(runID, cp); err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	// Get resume state
	resumeMeta, outputs, err := manager.GetResumeState()
	if err != nil {
		t.Fatalf("GetResumeState failed: %v", err)
	}

	if resumeMeta == nil {
		t.Fatal("resumeMeta should not be nil")
	}
	if resumeMeta.SessionName != "test-session" {
		t.Errorf("SessionName = %q, want test-session", resumeMeta.SessionName)
	}
	if len(resumeMeta.PendingIDs) != 1 {
		t.Errorf("PendingIDs count = %d, want 1", len(resumeMeta.PendingIDs))
	}
	if len(outputs) != 1 {
		t.Errorf("outputs count = %d, want 1", len(outputs))
	}
	if outputs[0].Thesis != "Result" {
		t.Errorf("output Thesis = %q, want Result", outputs[0].Thesis)
	}
}

func TestCheckpointManager_GetResumeState_NilManager(t *testing.T) {
	var manager *CheckpointManager
	_, _, err := manager.GetResumeState()
	if err == nil {
		t.Error("GetResumeState on nil should return error")
	}
}
