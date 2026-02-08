package bv

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// renderFullTriage — missing branch coverage tests
// ---------------------------------------------------------------------------

func TestRenderFullTriage_NoRecommendations(t *testing.T) {
	t.Parallel()

	triage := &TriageResponse{
		Triage: TriageData{
			QuickRef: TriageQuickRef{
				OpenCount:       5,
				ActionableCount: 0,
			},
			Recommendations: nil,
		},
	}

	opts := DefaultMarkdownOptions()
	result := RenderTriageMarkdown(triage, opts)

	if !strings.Contains(result, "_No recommendations._") {
		t.Errorf("expected 'No recommendations' message, got:\n%s", result)
	}
}

func TestRenderFullTriage_QuickWins(t *testing.T) {
	t.Parallel()

	triage := &TriageResponse{
		Triage: TriageData{
			QuickRef: TriageQuickRef{OpenCount: 10},
			Recommendations: []TriageRecommendation{
				{ID: "bd-1", Title: "Main task", Type: "task", Score: 0.9, Action: "Do it"},
			},
			QuickWins: []TriageRecommendation{
				{ID: "bd-qw1", Title: "Quick fix one", Type: "fix", Action: "Apply fix"},
				{ID: "bd-qw2", Title: "Quick fix two", Type: "fix"},
			},
		},
	}

	opts := DefaultMarkdownOptions()
	result := RenderTriageMarkdown(triage, opts)

	if !strings.Contains(result, "### Quick Wins") {
		t.Errorf("expected Quick Wins section, got:\n%s", result)
	}
	if !strings.Contains(result, "bd-qw1") {
		t.Errorf("expected first quick win ID, got:\n%s", result)
	}
	if !strings.Contains(result, "bd-qw2") {
		t.Errorf("expected second quick win ID, got:\n%s", result)
	}
	// First has action, second doesn't
	if !strings.Contains(result, "Action: Apply fix") {
		t.Errorf("expected action for first quick win, got:\n%s", result)
	}
}

func TestRenderFullTriage_BlockersToClear(t *testing.T) {
	t.Parallel()

	triage := &TriageResponse{
		Triage: TriageData{
			QuickRef: TriageQuickRef{OpenCount: 10, BlockedCount: 3},
			Recommendations: []TriageRecommendation{
				{ID: "bd-1", Title: "Task", Type: "task", Score: 0.8, Action: "Go"},
			},
			BlockersToClear: []BlockerToClear{
				{ID: "bd-bl1", Title: "Blocker one", UnblocksIDs: []string{"bd-2", "bd-3"}},
				{ID: "bd-bl2", Title: "Blocker two"},
			},
		},
	}

	opts := DefaultMarkdownOptions()
	result := RenderTriageMarkdown(triage, opts)

	if !strings.Contains(result, "### Blockers to Clear") {
		t.Errorf("expected Blockers to Clear section, got:\n%s", result)
	}
	if !strings.Contains(result, "bd-bl1") {
		t.Errorf("expected first blocker ID, got:\n%s", result)
	}
	// First blocker should show unblocks count
	if !strings.Contains(result, "unblocks 2") {
		t.Errorf("expected 'unblocks 2' for first blocker, got:\n%s", result)
	}
}

func TestRenderFullTriage_RecommendationsTruncated(t *testing.T) {
	t.Parallel()

	recs := make([]TriageRecommendation, 10)
	for i := range recs {
		recs[i] = TriageRecommendation{
			ID:     "bd-" + strings.Repeat("x", 3),
			Title:  "Recommendation",
			Type:   "task",
			Score:  0.5,
			Action: "Do it",
		}
	}

	triage := &TriageResponse{
		Triage: TriageData{
			QuickRef:        TriageQuickRef{OpenCount: 20},
			Recommendations: recs,
		},
	}

	opts := DefaultMarkdownOptions()
	opts.MaxRecommendations = 3
	result := RenderTriageMarkdown(triage, opts)

	if !strings.Contains(result, "...and 7 more") {
		t.Errorf("expected truncation message '...and 7 more', got:\n%s", result)
	}
}

func TestRenderFullTriage_NoProjectHealth(t *testing.T) {
	t.Parallel()

	triage := &TriageResponse{
		Triage: TriageData{
			QuickRef: TriageQuickRef{OpenCount: 5},
			Recommendations: []TriageRecommendation{
				{ID: "bd-1", Title: "Task", Type: "task", Score: 0.8, Action: "Go"},
			},
			ProjectHealth: nil, // No health data
		},
	}

	opts := DefaultMarkdownOptions()
	result := RenderTriageMarkdown(triage, opts)

	// Should not contain health section
	if strings.Contains(result, "### Project Health") {
		t.Errorf("unexpected Project Health section when nil, got:\n%s", result)
	}
}

func TestRenderFullTriage_AllSectionsCombined(t *testing.T) {
	t.Parallel()

	triage := &TriageResponse{
		Triage: TriageData{
			QuickRef: TriageQuickRef{
				OpenCount:       25,
				ActionableCount: 12,
				BlockedCount:    5,
				InProgressCount: 4,
			},
			Recommendations: []TriageRecommendation{
				{ID: "bd-r1", Title: "Top task", Type: "task", Score: 0.95, Action: "Start now"},
			},
			QuickWins: []TriageRecommendation{
				{ID: "bd-qw1", Title: "Easy fix", Type: "fix", Action: "Quick action"},
			},
			BlockersToClear: []BlockerToClear{
				{ID: "bd-bl1", Title: "Blocker", UnblocksIDs: []string{"bd-x", "bd-y", "bd-z"}},
			},
			ProjectHealth: &ProjectHealth{
				GraphMetrics: &GraphMetrics{
					TotalNodes: 50,
					TotalEdges: 80,
					Density:    0.06,
					CycleCount: 0,
				},
			},
		},
	}

	opts := DefaultMarkdownOptions()
	result := RenderTriageMarkdown(triage, opts)

	// All sections should be present
	sections := []string{
		"## Beads Triage",
		"### Recommendations",
		"### Quick Wins",
		"### Blockers to Clear",
		"### Project Health",
	}
	for _, section := range sections {
		if !strings.Contains(result, section) {
			t.Errorf("expected section %q, got:\n%s", section, result)
		}
	}

	// Verify counts in table
	if !strings.Contains(result, "| Open | 25 |") {
		t.Error("expected open count 25 in table")
	}
	if !strings.Contains(result, "| Blocked | 5 |") {
		t.Error("expected blocked count 5 in table")
	}
}

// ---------------------------------------------------------------------------
// setNoDBState / getNoDBState
// ---------------------------------------------------------------------------

func TestSetAndGetNoDBState(t *testing.T) {
	dir := t.TempDir()

	// Default should be false (not set)
	if getNoDBState(dir) {
		t.Error("expected getNoDBState to return false for unset dir")
	}

	// Set to true
	setNoDBState(dir, true)
	if !getNoDBState(dir) {
		t.Error("expected getNoDBState to return true after setNoDBState(true)")
	}

	// Set back to false
	setNoDBState(dir, false)
	if getNoDBState(dir) {
		t.Error("expected getNoDBState to return false after setNoDBState(false)")
	}
}
