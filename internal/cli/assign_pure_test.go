package cli

import (
	"encoding/json"
	"testing"

	"github.com/Dicklesworthstone/ntm/internal/tui/theme"
)

// =============================================================================
// IsBeadInCycle — 0% → 100%
// =============================================================================

func TestIsBeadInCycle_Found(t *testing.T) {
	t.Parallel()

	cycles := [][]string{
		{"bd-001", "bd-002", "bd-003"},
		{"bd-010", "bd-020"},
	}

	if !IsBeadInCycle("bd-002", cycles) {
		t.Error("expected bd-002 to be in cycle")
	}
	if !IsBeadInCycle("bd-020", cycles) {
		t.Error("expected bd-020 to be in cycle")
	}
}

func TestIsBeadInCycle_NotFound(t *testing.T) {
	t.Parallel()

	cycles := [][]string{
		{"bd-001", "bd-002"},
	}

	if IsBeadInCycle("bd-999", cycles) {
		t.Error("expected bd-999 NOT to be in cycle")
	}
}

func TestIsBeadInCycle_EmptyCycles(t *testing.T) {
	t.Parallel()

	if IsBeadInCycle("bd-001", nil) {
		t.Error("expected false for nil cycles")
	}
	if IsBeadInCycle("bd-001", [][]string{}) {
		t.Error("expected false for empty cycles")
	}
}

// =============================================================================
// getAgentStyle — 0% → 100%
// =============================================================================

func TestGetAgentStyle(t *testing.T) {
	t.Parallel()

	th := theme.Current()

	tests := []struct {
		name      string
		agentType string
	}{
		{"claude", "claude"},
		{"codex", "codex"},
		{"gemini", "gemini"},
		{"unknown", "aider"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			style := getAgentStyle(tt.agentType, th)
			// Style should be usable — render something to verify no panic
			rendered := style.Render("test")
			if rendered == "" {
				t.Error("expected non-empty styled string")
			}
		})
	}
}

// =============================================================================
// getPriorityStyle — 0% → 100%
// =============================================================================

func TestGetPriorityStyle(t *testing.T) {
	t.Parallel()

	th := theme.Current()

	tests := []struct {
		name     string
		priority string
	}{
		{"P0", "P0"},
		{"P1", "P1"},
		{"P2", "P2"},
		{"P3_default", "P3"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			style := getPriorityStyle(tt.priority, th)
			rendered := style.Render("test")
			if rendered == "" {
				t.Error("expected non-empty styled string")
			}
		})
	}
}

// =============================================================================
// makeRetryEnvelope — 0% → 100%
// =============================================================================

func TestMakeRetryEnvelope_Success(t *testing.T) {
	t.Parallel()

	data := &RetryData{
		Summary: RetrySummary{TotalFailed: 3, RetriedCount: 2, SkippedCount: 1},
	}
	env := makeRetryEnvelope("my-session", true, data, "", "", nil)

	if env.Command != "assign" {
		t.Errorf("Command = %q, want assign", env.Command)
	}
	if env.Subcommand != "retry" {
		t.Errorf("Subcommand = %q, want retry", env.Subcommand)
	}
	if env.Session != "my-session" {
		t.Errorf("Session = %q, want my-session", env.Session)
	}
	if !env.Success {
		t.Error("expected Success=true")
	}
	if env.Error != nil {
		t.Error("expected nil error")
	}
	if env.Data == nil {
		t.Fatal("expected non-nil Data")
	}
	if env.Data.Summary.TotalFailed != 3 {
		t.Errorf("Data.Summary.TotalFailed = %d, want 3", env.Data.Summary.TotalFailed)
	}
	// Nil warnings should be converted to empty slice
	if len(env.Warnings) != 0 {
		t.Errorf("Warnings = %v, want empty", env.Warnings)
	}
}

func TestMakeRetryEnvelope_WithError(t *testing.T) {
	t.Parallel()

	env := makeRetryEnvelope("s", false, nil, "STORE_ERROR", "broken", []string{"w1"})

	if env.Success {
		t.Error("expected Success=false")
	}
	if env.Error == nil {
		t.Fatal("expected non-nil error")
	}
	if env.Error.Code != "STORE_ERROR" {
		t.Errorf("Error.Code = %q, want STORE_ERROR", env.Error.Code)
	}
	if env.Error.Message != "broken" {
		t.Errorf("Error.Message = %q, want broken", env.Error.Message)
	}
	if len(env.Warnings) != 1 || env.Warnings[0] != "w1" {
		t.Errorf("Warnings = %v, want [w1]", env.Warnings)
	}
}

func TestMakeRetryEnvelope_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	env := makeRetryEnvelope("sess", true, nil, "", "", nil)
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if decoded["command"] != "assign" {
		t.Errorf("decoded command = %v", decoded["command"])
	}
}

// =============================================================================
// makeDirectAssignEnvelope — 0% → 100%
// =============================================================================

func TestMakeDirectAssignEnvelope_Success(t *testing.T) {
	t.Parallel()

	data := &DirectAssignData{
		Assignment: &DirectAssignItem{BeadID: "bd-1"},
	}
	env := makeDirectAssignEnvelope("sess", true, data, "", "", nil)

	if env.Command != "assign" {
		t.Errorf("Command = %q", env.Command)
	}
	if env.Subcommand != "pane" {
		t.Errorf("Subcommand = %q", env.Subcommand)
	}
	if !env.Success {
		t.Error("expected success")
	}
	if env.Error != nil {
		t.Error("expected nil error")
	}
	// nil warnings → empty
	if len(env.Warnings) != 0 {
		t.Errorf("Warnings = %v", env.Warnings)
	}
}

func TestMakeDirectAssignEnvelope_WithError(t *testing.T) {
	t.Parallel()

	env := makeDirectAssignEnvelope("s", false, nil, "INVALID_ARGS", "bad", []string{"w"})

	if env.Success {
		t.Error("expected failure")
	}
	if env.Error == nil || env.Error.Code != "INVALID_ARGS" {
		t.Errorf("Error = %+v", env.Error)
	}
}

// =============================================================================
// marshalAssignOutput — 0% → 100%
// =============================================================================

func TestMarshalAssignOutput_Nil(t *testing.T) {
	t.Parallel()

	data, err := marshalAssignOutput(nil)
	if err != nil {
		t.Fatalf("marshalAssignOutput(nil): %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}
}
