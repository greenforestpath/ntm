package git

import (
	"context"
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/config"
)

func TestNewWorktreeService(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	svc := NewWorktreeService(cfg)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.managers == nil {
		t.Fatal("expected managers map to be initialized")
	}
	if svc.config != cfg {
		t.Fatal("expected config to be stored")
	}
}

func TestWorktreeService_getManager(t *testing.T) {
	t.Parallel()

	tmp := setupGitRepo(t)

	cfg := &config.Config{}
	svc := NewWorktreeService(cfg)

	// First call creates a new manager
	m1, err := svc.getManager(tmp)
	if err != nil {
		t.Fatalf("getManager: %v", err)
	}
	if m1 == nil {
		t.Fatal("expected non-nil manager")
	}
	if m1.projectDir != tmp {
		t.Errorf("projectDir = %q, want %q", m1.projectDir, tmp)
	}

	// Second call returns the cached manager
	m2, err := svc.getManager(tmp)
	if err != nil {
		t.Fatalf("getManager (cached): %v", err)
	}
	if m1 != m2 {
		t.Fatal("expected same manager instance from cache")
	}
}

func TestWorktreeService_getManager_NotGitRepo(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cfg := &config.Config{}
	svc := NewWorktreeService(cfg)

	_, err := svc.getManager(tmp)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

func TestWorktreeService_GetAllWorktrees_Empty(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	svc := NewWorktreeService(cfg)

	result, err := svc.GetAllWorktrees(context.Background())
	if err != nil {
		t.Fatalf("GetAllWorktrees: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d projects", len(result))
	}
}

func TestWorktreeService_GetAllWorktrees_WithManagers(t *testing.T) {
	t.Parallel()

	tmp := setupGitRepo(t)

	cfg := &config.Config{}
	svc := NewWorktreeService(cfg)

	// Populate via getManager
	_, err := svc.getManager(tmp)
	if err != nil {
		t.Fatalf("getManager: %v", err)
	}

	result, err := svc.GetAllWorktrees(context.Background())
	if err != nil {
		t.Fatalf("GetAllWorktrees: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 project, got %d", len(result))
	}
	if wts, ok := result[tmp]; !ok {
		t.Error("expected project dir key in result")
	} else if len(wts) != 0 {
		t.Errorf("expected 0 worktrees (no agents), got %d", len(wts))
	}
}

func TestWorktreeService_CleanupStaleWorktrees_NoManagers(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	svc := NewWorktreeService(cfg)

	err := svc.CleanupStaleWorktrees(context.Background(), 24*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleWorktrees: %v", err)
	}
}

func TestWorktreeService_CleanupStaleWorktrees_WithManagers(t *testing.T) {
	t.Parallel()

	tmp := setupGitRepo(t)

	cfg := &config.Config{}
	svc := NewWorktreeService(cfg)

	_, err := svc.getManager(tmp)
	if err != nil {
		t.Fatalf("getManager: %v", err)
	}

	// No worktrees exist, so this should be a no-op
	err = svc.CleanupStaleWorktrees(context.Background(), 24*time.Hour)
	if err != nil {
		t.Fatalf("CleanupStaleWorktrees: %v", err)
	}
}
