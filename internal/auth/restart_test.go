package auth

import (
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/config"
)

// =============================================================================
// NewOrchestrator
// =============================================================================

func TestNewOrchestrator(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	orch := NewOrchestrator(cfg)

	if orch == nil {
		t.Fatal("NewOrchestrator returned nil")
	}
	if orch.cfg != cfg {
		t.Error("config not stored correctly")
	}
	if orch.authFlows == nil {
		t.Error("authFlows map should be initialized")
	}
	if len(orch.authFlows) != 0 {
		t.Errorf("authFlows should be empty, got %d entries", len(orch.authFlows))
	}
	if orch.captureOutput == nil {
		t.Error("captureOutput should be set")
	}
}

// =============================================================================
// RegisterAuthFlow
// =============================================================================

type mockAuthFlow struct{}

func (m *mockAuthFlow) InitiateAuth(paneID string) error { return nil }

func TestRegisterAuthFlow(t *testing.T) {
	t.Parallel()

	orch := NewOrchestrator(config.Default())

	t.Run("register single flow", func(t *testing.T) {
		flow := &mockAuthFlow{}
		orch.RegisterAuthFlow("claude", flow)
		if got, ok := orch.authFlows["claude"]; !ok {
			t.Error("flow not registered")
		} else if got != flow {
			t.Error("wrong flow stored")
		}
	})

	t.Run("register multiple flows", func(t *testing.T) {
		orch2 := NewOrchestrator(config.Default())
		orch2.RegisterAuthFlow("claude", &mockAuthFlow{})
		orch2.RegisterAuthFlow("codex", &mockAuthFlow{})
		orch2.RegisterAuthFlow("gemini", &mockAuthFlow{})
		if len(orch2.authFlows) != 3 {
			t.Errorf("got %d flows, want 3", len(orch2.authFlows))
		}
	})

	t.Run("overwrite existing flow", func(t *testing.T) {
		orch3 := NewOrchestrator(config.Default())
		flow1 := &mockAuthFlow{}
		flow2 := &mockAuthFlow{}
		orch3.RegisterAuthFlow("claude", flow1)
		orch3.RegisterAuthFlow("claude", flow2)
		if orch3.authFlows["claude"] != flow2 {
			t.Error("flow should be overwritten")
		}
		if len(orch3.authFlows) != 1 {
			t.Errorf("got %d flows, want 1", len(orch3.authFlows))
		}
	})
}

// =============================================================================
// Shell prompt regex matching
// =============================================================================

func TestShellPromptRegexps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		output  string
		matches bool
	}{
		{"bash dollar sign", "user@host:~$ ", true},
		{"bare dollar", "$ ", true},
		{"dollar end of line", "$", true},
		{"dollar with trailing space", "$  ", true},
		{"zsh percent", "user@host %", true},
		{"bare percent", "% ", true},
		{"percent end of line", "%", true},
		{"generic prompt", "> ", true},
		{"bare angle bracket", ">", true},
		{"no prompt - text only", "still running command", false},
		{"no prompt - empty", "", false},
		{"dollar mid-text", "cost is $5 for this", false},
		{"percent mid-text", "100% complete", false},
		{"angle mid-text", "a > b", false},
		{"multiline with prompt at end", "output line 1\noutput line 2\n$", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			matched := false
			for _, re := range shellPromptRegexps {
				if re.MatchString(tt.output) {
					matched = true
					break
				}
			}
			if matched != tt.matches {
				t.Errorf("shellPromptRegexps match(%q) = %v, want %v", tt.output, matched, tt.matches)
			}
		})
	}
}

// =============================================================================
// WaitForShellPrompt
// =============================================================================

func TestWaitForShellPrompt(t *testing.T) {
	orch := NewOrchestrator(config.Default())

	tests := []struct {
		name        string
		mockOutputs []string
		timeout     time.Duration
		wantErr     bool
	}{
		{
			name:        "detect bash prompt immediately",
			mockOutputs: []string{"user@host:~$"},
			timeout:     1 * time.Second,
			wantErr:     false,
		},
		{
			name:        "detect zsh prompt after delay",
			mockOutputs: []string{"output line 1", "output line 2", "user@host %"},
			timeout:     2 * time.Second,
			wantErr:     false,
		},
		{
			name:        "detect generic prompt",
			mockOutputs: []string{"> "},
			timeout:     1 * time.Second,
			wantErr:     false,
		},
		{
			name:        "timeout waiting for prompt",
			mockOutputs: []string{"still running...", "still running...", "still running..."},
			timeout:     100 * time.Millisecond,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := 0
			orch.captureOutput = func(paneID string, lines int) (string, error) {
				if idx >= len(tt.mockOutputs) {
					return tt.mockOutputs[len(tt.mockOutputs)-1], nil
				}
				out := tt.mockOutputs[idx]
				idx++
				return out, nil
			}

			start := time.Now()
			err := orch.WaitForShellPrompt("dummy", tt.timeout)
			duration := time.Since(start)

			if (err != nil) != tt.wantErr {
				t.Errorf("WaitForShellPrompt() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && duration > tt.timeout {
				t.Errorf("WaitForShellPrompt() took %v, want < %v", duration, tt.timeout)
			}
		})
	}
}

// =============================================================================
// RestartContext fields
// =============================================================================

func TestRestartContextFields(t *testing.T) {
	t.Parallel()

	ctx := RestartContext{
		PaneID:      "%42",
		Provider:    "claude",
		TargetEmail: "user@example.com",
		ModelAlias:  "opus",
		SessionName: "test-session",
		PaneIndex:   3,
		ProjectDir:  "/data/projects/myapp",
	}

	if ctx.PaneID != "%42" {
		t.Errorf("PaneID = %q", ctx.PaneID)
	}
	if ctx.Provider != "claude" {
		t.Errorf("Provider = %q", ctx.Provider)
	}
	if ctx.PaneIndex != 3 {
		t.Errorf("PaneIndex = %d", ctx.PaneIndex)
	}
}
