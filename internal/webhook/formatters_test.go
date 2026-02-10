package webhook

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// classifySeverity tests (bd-1ced7)
// =============================================================================

func TestClassifySeverity_Empty(t *testing.T) {
	t.Parallel()
	if got := classifySeverity(""); got != severityInfo {
		t.Errorf("classifySeverity(\"\") = %q, want severityInfo (%q)", got, severityInfo)
	}
}

func TestClassifySeverity_Error(t *testing.T) {
	t.Parallel()
	for _, typ := range []string{"agent.error", "task_failed", "crash_detected", "kernel_panic"} {
		if got := classifySeverity(typ); got != severityError {
			t.Errorf("classifySeverity(%q) = %q, want severityError (%q)", typ, got, severityError)
		}
	}
}

func TestClassifySeverity_Warning(t *testing.T) {
	t.Parallel()
	for _, typ := range []string{"performance_warning", "degraded_response", "rate_limit_hit", "rate-limit"} {
		if got := classifySeverity(typ); got != severityWarning {
			t.Errorf("classifySeverity(%q) = %q, want severityWarning (%q)", typ, got, severityWarning)
		}
	}
}

func TestClassifySeverity_Success(t *testing.T) {
	t.Parallel()
	for _, typ := range []string{"task.success", "build.complete", "deployment_done", "agent.healthy"} {
		if got := classifySeverity(typ); got != severitySuccess {
			t.Errorf("classifySeverity(%q) = %q, want severitySuccess (%q)", typ, got, severitySuccess)
		}
	}
}

func TestClassifySeverity_InfoDefault(t *testing.T) {
	t.Parallel()
	for _, typ := range []string{"agent.started", "session.created", "status.update"} {
		if got := classifySeverity(typ); got != severityInfo {
			t.Errorf("classifySeverity(%q) = %q, want severityInfo (%q)", typ, got, severityInfo)
		}
	}
}

// =============================================================================
// discordColorForSeverity tests (bd-1ced7)
// =============================================================================

func TestDiscordColorForSeverity_Error(t *testing.T) {
	t.Parallel()
	if got := discordColorForSeverity(severityError); got != 0xE74C3C {
		t.Errorf("error color = 0x%X, want 0xE74C3C", got)
	}
}

func TestDiscordColorForSeverity_Warning(t *testing.T) {
	t.Parallel()
	if got := discordColorForSeverity(severityWarning); got != 0xF1C40F {
		t.Errorf("warning color = 0x%X, want 0xF1C40F", got)
	}
}

func TestDiscordColorForSeverity_Success(t *testing.T) {
	t.Parallel()
	if got := discordColorForSeverity(severitySuccess); got != 0x2ECC71 {
		t.Errorf("success color = 0x%X, want 0x2ECC71", got)
	}
}

func TestDiscordColorForSeverity_Info(t *testing.T) {
	t.Parallel()
	if got := discordColorForSeverity(severityInfo); got != 0x3498DB {
		t.Errorf("info color = 0x%X, want 0x3498DB", got)
	}
}

func TestBuildBuiltInPayload_InvalidFormat(t *testing.T) {
	t.Parallel()

	_, err := buildBuiltInPayload(Event{Type: "test", Message: "hello"}, "nope")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "unknown webhook format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildBuiltInPayload_JSON(t *testing.T) {
	t.Parallel()

	ev := Event{
		ID:        "evt-1",
		Type:      "agent.error",
		Timestamp: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
		Session:   "myproj",
		Pane:      "myproj__cc_1",
		Agent:     "claude",
		Message:   "hello",
		Details:   map[string]string{"k": "v"},
	}

	b, err := buildBuiltInPayload(ev, "json")
	if err != nil {
		t.Fatalf("buildBuiltInPayload: %v", err)
	}

	var got Event
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Type != ev.Type {
		t.Fatalf("type=%q want %q", got.Type, ev.Type)
	}
	if got.Message != ev.Message {
		t.Fatalf("message=%q want %q", got.Message, ev.Message)
	}
}

func TestBuildBuiltInPayload_Slack(t *testing.T) {
	t.Parallel()

	ev := Event{
		Type:      "agent.completed",
		Timestamp: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
		Session:   "myproj",
		Pane:      "myproj__cc_1",
		Agent:     "claude",
		Message:   "all good",
		Details:   map[string]string{"task": "tests", "result": "pass"},
	}

	b, err := buildBuiltInPayload(ev, "slack")
	if err != nil {
		t.Fatalf("buildBuiltInPayload: %v", err)
	}

	var got slackPayload
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Text == "" {
		t.Fatalf("expected text fallback")
	}
	if len(got.Blocks) < 2 {
		t.Fatalf("expected blocks, got %d", len(got.Blocks))
	}
	if got.Blocks[0].Type != "header" {
		t.Fatalf("expected header block first, got %q", got.Blocks[0].Type)
	}
}

func TestBuildBuiltInPayload_Discord(t *testing.T) {
	t.Parallel()

	ev := Event{
		ID:        "evt-2",
		Type:      "agent.error",
		Timestamp: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
		Session:   "myproj",
		Pane:      "myproj__cod_1",
		Agent:     "codex",
		Message:   "something broke",
		Details:   map[string]string{"code": "E123"},
	}

	b, err := buildBuiltInPayload(ev, "discord")
	if err != nil {
		t.Fatalf("buildBuiltInPayload: %v", err)
	}

	var got discordPayload
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(got.Embeds))
	}
	if got.Embeds[0].Title == "" {
		t.Fatalf("expected embed title")
	}
	if got.Embeds[0].Description != ev.Message {
		t.Fatalf("embed description=%q want %q", got.Embeds[0].Description, ev.Message)
	}
	if got.Embeds[0].Timestamp == "" {
		t.Fatalf("expected embed timestamp")
	}
}

func TestBuildBuiltInPayload_Teams(t *testing.T) {
	t.Parallel()

	ev := Event{
		Type:      "agent.error",
		Timestamp: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
		Session:   "myproj",
		Message:   "something broke",
		Details:   map[string]string{"code": "E123"},
	}

	b, err := buildBuiltInPayload(ev, "teams")
	if err != nil {
		t.Fatalf("buildBuiltInPayload: %v", err)
	}

	var got teamsPayload
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Type != "message" {
		t.Fatalf("type=%q want %q", got.Type, "message")
	}
	if len(got.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(got.Attachments))
	}
	if got.Attachments[0].ContentType != "application/vnd.microsoft.card.adaptive" {
		t.Fatalf("contentType=%q", got.Attachments[0].ContentType)
	}
	if len(got.Attachments[0].Content.Body) < 2 {
		t.Fatalf("expected at least 2 card body elements, got %d", len(got.Attachments[0].Content.Body))
	}
}
