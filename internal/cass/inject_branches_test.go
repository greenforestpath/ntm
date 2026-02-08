package cass

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// formatMarkdown — missing age=="" branch (93.8% → higher)
// ---------------------------------------------------------------------------

func TestFormatMarkdown_NoDateInPath(t *testing.T) {
	t.Parallel()

	hits := []ScoredHit{
		{
			CASSHit:       CASSHit{SourcePath: "sessions/my-session.jsonl", Content: "Some relevant content here"},
			ComputedScore: 0.85,
		},
	}

	result := formatMarkdown(hits)
	if !strings.Contains(result, "## Relevant Context") {
		t.Error("expected header")
	}
	if !strings.Contains(result, "85% match") {
		t.Error("expected relevance percentage")
	}
	// With no date in path, formatAge returns "" → no age text
	if strings.Contains(result, "today") || strings.Contains(result, "ago") {
		t.Error("expected no age text for path without date")
	}
}

func TestFormatMarkdown_EmptyContent(t *testing.T) {
	t.Parallel()

	hits := []ScoredHit{
		{
			CASSHit:       CASSHit{SourcePath: "sessions/my-session.jsonl"},
			ComputedScore: 0.5,
		},
	}

	result := formatMarkdown(hits)
	if !strings.Contains(result, "50% match") {
		t.Error("expected relevance percentage")
	}
}

func TestFormatMarkdown_MultipleHits(t *testing.T) {
	t.Parallel()

	hits := []ScoredHit{
		{CASSHit: CASSHit{SourcePath: "sessions/a.jsonl", Content: "First"}, ComputedScore: 0.9},
		{CASSHit: CASSHit{SourcePath: "sessions/b.jsonl", Content: "Second"}, ComputedScore: 0.7},
	}

	result := formatMarkdown(hits)
	if !strings.Contains(result, "90% match") {
		t.Error("expected first hit")
	}
	if !strings.Contains(result, "70% match") {
		t.Error("expected second hit")
	}
}

// ---------------------------------------------------------------------------
// isSameProject — missing edge-case branches (85.7% → higher)
// ---------------------------------------------------------------------------

func TestIsSameProject_SingleComponent(t *testing.T) {
	t.Parallel()

	// Workspace path with single component (after slash split)
	got := isSameProject("sessions/myproject/log.jsonl", "myproject")
	if !got {
		t.Error("expected match for single-component workspace")
	}
}

func TestIsSameProject_TrailingSlash(t *testing.T) {
	t.Parallel()

	got := isSameProject("sessions/myproject/log.jsonl", "/home/user/myproject/")
	if !got {
		t.Error("expected match with trailing slash workspace")
	}
}
