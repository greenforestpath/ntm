package summary

import (
	"strings"
	"testing"

	"github.com/Dicklesworthstone/ntm/internal/agentmail"
	"github.com/Dicklesworthstone/ntm/internal/handoff"
)

// ---------------------------------------------------------------------------
// formatDetailed — missing ThreadSummaries branch (57.1% → higher)
// ---------------------------------------------------------------------------

func TestFormatDetailed_WithThreads(t *testing.T) {
	t.Parallel()

	summary := &SessionSummary{
		Session: "thread-test",
		ThreadSummaries: []agentmail.ThreadSummary{
			{
				ThreadID:    "TKT-123",
				KeyPoints:   []string{"Discussed migration plan", "Agreed on approach"},
				ActionItems: []string{"Update schema", "Run tests"},
			},
			{
				ThreadID:    "TKT-456",
				KeyPoints:   []string{"Reviewed PR"},
				ActionItems: nil,
			},
		},
	}

	result := formatDetailed(summary)
	if !strings.Contains(result, "## Thread Summaries") {
		t.Error("expected Thread Summaries section")
	}
	if !strings.Contains(result, "TKT-123") {
		t.Error("expected first thread ID")
	}
	if !strings.Contains(result, "Discussed migration plan") {
		t.Error("expected key point")
	}
	if !strings.Contains(result, "TODO: Update schema") {
		t.Error("expected action item with TODO prefix")
	}
	if !strings.Contains(result, "TKT-456") {
		t.Error("expected second thread ID")
	}
}

func TestFormatDetailed_EmptySession(t *testing.T) {
	t.Parallel()

	summary := &SessionSummary{}
	result := formatDetailed(summary)
	if !strings.Contains(result, "(unknown)") {
		t.Error("expected (unknown) for empty session name")
	}
}

// ---------------------------------------------------------------------------
// formatBrief — missing ThreadSummaries branch (84.6% → higher)
// ---------------------------------------------------------------------------

func TestFormatBrief_WithThreads(t *testing.T) {
	t.Parallel()

	summary := &SessionSummary{
		Session: "brief-threads",
		ThreadSummaries: []agentmail.ThreadSummary{
			{ThreadID: "TKT-1"},
			{ThreadID: "TKT-2"},
		},
	}

	result := formatBrief(summary)
	if !strings.Contains(result, "Threads summarized: 2") {
		t.Errorf("expected thread count in brief output, got:\n%s", result)
	}
}

func TestFormatBrief_EmptySession(t *testing.T) {
	t.Parallel()

	summary := &SessionSummary{}
	result := formatBrief(summary)
	if !strings.Contains(result, "(unknown)") {
		t.Error("expected (unknown) for empty session name")
	}
}

// ---------------------------------------------------------------------------
// formatHandoff — 66.7% → 100%
// ---------------------------------------------------------------------------

func TestFormatHandoff_Nil(t *testing.T) {
	t.Parallel()

	result := formatHandoff(nil)
	if result != "" {
		t.Errorf("expected empty string for nil handoff, got %q", result)
	}
}

func TestFormatHandoff_Valid(t *testing.T) {
	t.Parallel()

	h := &handoff.Handoff{
		Session: "test-session",
		Goal:    "Complete auth module",
	}

	result := formatHandoff(h)
	if result == "" {
		t.Error("expected non-empty YAML output")
	}
	if !strings.Contains(result, "test-session") {
		t.Errorf("expected session in output, got:\n%s", result)
	}
}

// ---------------------------------------------------------------------------
// appendPromptList — 33.3% → 100%
// ---------------------------------------------------------------------------

func TestAppendPromptList_Empty(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	appendPromptList(&sb, "Items", nil)
	if sb.Len() > 0 {
		t.Errorf("expected no output for empty list, got %q", sb.String())
	}
}

func TestAppendPromptList_NonEmpty(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	appendPromptList(&sb, "Tasks", []string{"Task A", "Task B"})
	result := sb.String()
	if !strings.Contains(result, "Tasks:") {
		t.Error("expected label")
	}
	if !strings.Contains(result, "- Task A") {
		t.Error("expected first item")
	}
	if !strings.Contains(result, "- Task B") {
		t.Error("expected second item")
	}
}

// ---------------------------------------------------------------------------
// appendPromptFiles — 33.3% → 100%
// ---------------------------------------------------------------------------

func TestAppendPromptFiles_Empty(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	appendPromptFiles(&sb, "Files", nil)
	if sb.Len() > 0 {
		t.Errorf("expected no output for empty files, got %q", sb.String())
	}
}

func TestAppendPromptFiles_NonEmpty(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	appendPromptFiles(&sb, "Modified Files", []FileChange{
		{Action: FileActionCreated, Path: "new.go"},
		{Action: FileActionModified, Path: "old.go"},
	})
	result := sb.String()
	if !strings.Contains(result, "Modified Files:") {
		t.Error("expected label")
	}
	if !strings.Contains(result, "created new.go") {
		t.Error("expected created file entry")
	}
	if !strings.Contains(result, "modified old.go") {
		t.Error("expected modified file entry")
	}
}

// ---------------------------------------------------------------------------
// cleanContextLine — missing truncation branch (88.9% → 100%)
// ---------------------------------------------------------------------------

func TestCleanContextLine_Long(t *testing.T) {
	t.Parallel()

	// Create a line longer than 120 chars.
	longLine := strings.Repeat("x", 200)
	result := cleanContextLine(longLine)
	if len(result) > 130 { // 120 + "..." = 123
		t.Errorf("expected truncation, got len=%d", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Error("expected ... suffix for truncated line")
	}
}

func TestCleanContextLine_BulletStrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"dash bullet", "- Task item", "Task item"},
		{"star bullet", "* Task item", "Task item"},
		{"bullet char", "• Task item", "Task item"},
		{"numbered", "1. First item", "First item"},
		{"checkbox done", "[x] Done item", "Done item"},
		{"checkbox todo", "[ ] Todo item", "Todo item"},
		{"no bullet", "Plain text line", "Plain text line"},
		{"too short", "- ab", "ab"},
		{"empty", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := cleanContextLine(tc.input)
			if got != tc.want {
				t.Errorf("cleanContextLine(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseBulletItem — missing short-line branch (81.8% → 100%)
// ---------------------------------------------------------------------------

func TestParseBulletItem_Short(t *testing.T) {
	t.Parallel()

	// Items after stripping prefix that are < 3 chars should return ""
	if got := parseBulletItem("- ab"); got != "" {
		t.Errorf("expected empty for short item, got %q", got)
	}
}

func TestParseBulletItem_Various(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"- Task done", "Task done"},
		{"* Another task", "Another task"},
		{"1. First", "First"},
		{"  - Indented", "Indented"},
		{"", ""},
		{"  ", ""},
		{"No prefix here text", "No prefix here text"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := parseBulletItem(tc.input)
			if got != tc.want {
				t.Errorf("parseBulletItem(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
