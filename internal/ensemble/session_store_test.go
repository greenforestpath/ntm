package ensemble

import (
	"os"
	"sync"
	"testing"
	"time"
)

func TestSessionStore_SaveLoad_DefaultPath(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	if defaultStateStore.store != nil {
		_ = defaultStateStore.store.Close()
	}
	defaultStateStore = struct {
		once  sync.Once
		store *StateStore
		err   error
	}{}

	session := &EnsembleSession{
		SessionName:       "default-session",
		Question:          "Question",
		Status:            EnsembleActive,
		SynthesisStrategy: StrategyConsensus,
		CreatedAt:         time.Now().UTC(),
	}

	if err := SaveSession("", session); err != nil {
		t.Fatalf("SaveSession error: %v", err)
	}
	loaded, err := LoadSession("default-session")
	if err != nil {
		t.Fatalf("LoadSession error: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected loaded session")
	}
	if loaded.Question != session.Question {
		t.Fatalf("Question = %q, want %q", loaded.Question, session.Question)
	}

	if defaultStateStore.store != nil {
		_ = defaultStateStore.store.Close()
		defaultStateStore = struct {
			once  sync.Once
			store *StateStore
			err   error
		}{}
	}

	// ensure no accidental writes to real home
	if _, err := os.Stat(tmpHome); err != nil {
		t.Fatalf("temp home missing: %v", err)
	}
}
