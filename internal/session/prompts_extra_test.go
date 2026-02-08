package session

import (
	"os"
	"testing"
	"time"
)

// =============================================================================
// savePromptHistory — 58.8% → higher
// =============================================================================

func TestSavePromptHistory_NilHistory(t *testing.T) {
	err := savePromptHistory(nil)
	if err == nil {
		t.Error("expected error for nil history")
	}
}

// =============================================================================
// GetLatestPrompts — 75% → 100% (test limit=0 returns all)
// =============================================================================

func TestGetLatestPrompts_NoLimit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ntm-prompts-nolimit-test")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	sessionName := "test-nolimit"

	// Save 3 prompts
	for i := 0; i < 3; i++ {
		entry := PromptEntry{
			Session:   sessionName,
			Content:   "prompt",
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Targets:   []string{"1"},
			Source:    "cli",
		}
		if err := SavePrompt(entry); err != nil {
			t.Fatalf("SavePrompt: %v", err)
		}
	}

	// Get with limit=0 should return all
	all, err := GetLatestPrompts(sessionName, 0)
	if err != nil {
		t.Fatalf("GetLatestPrompts: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("len = %d, want 3", len(all))
	}
}

func TestGetLatestPrompts_LimitExceedsCount(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ntm-prompts-exceed-test")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	sessionName := "test-exceed"

	entry := PromptEntry{
		Session: sessionName,
		Content: "single",
		Targets: []string{"1"},
		Source:  "cli",
	}
	if err := SavePrompt(entry); err != nil {
		t.Fatalf("SavePrompt: %v", err)
	}

	// Limit > count should return all
	result, err := GetLatestPrompts(sessionName, 100)
	if err != nil {
		t.Fatalf("GetLatestPrompts: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("len = %d, want 1", len(result))
	}
}

// =============================================================================
// ClearPromptHistory — 71.4% → higher (test clearing non-existent session)
// =============================================================================

func TestClearPromptHistory_NonExistentSession(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ntm-prompts-clear-noexist")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Clearing a session that never had prompts should not error
	err = ClearPromptHistory("nonexistent-session")
	if err != nil {
		t.Errorf("ClearPromptHistory(nonexistent) = %v, want nil", err)
	}
}

// =============================================================================
// ListSessionDirs — edge case: no sessions dir
// =============================================================================

func TestListSessionDirs_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ntm-list-empty")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	dirs, err := ListSessionDirs()
	if err != nil {
		t.Fatalf("ListSessionDirs: %v", err)
	}
	if len(dirs) != 0 {
		t.Errorf("len = %d, want 0", len(dirs))
	}
}

// =============================================================================
// promptsFilePath — exercise the happy path
// =============================================================================

func TestPromptsFilePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ntm-prompts-path")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	path, err := promptsFilePath("my-session")
	if err != nil {
		t.Fatalf("promptsFilePath: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
}
