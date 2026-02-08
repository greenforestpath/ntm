package ensemble

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// tierRank — missing default branch (60% → 100%)
// ---------------------------------------------------------------------------

func TestTierRank_AllCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tier ModeTier
		want int
	}{
		{TierCore, 0},
		{TierAdvanced, 1},
		{TierExperimental, 2},
		{"unknown-tier", 3},
		{"", 3},
	}

	for _, tc := range tests {
		t.Run(string(tc.tier), func(t *testing.T) {
			t.Parallel()
			got := tierRank(tc.tier)
			if got != tc.want {
				t.Errorf("tierRank(%q) = %d, want %d", tc.tier, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// coverageBar — missing bounds branches (70% → higher)
// ---------------------------------------------------------------------------

func TestCoverageBar_Branches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		used  int
		total int
		width int
		want  string
	}{
		{"zero width", 5, 10, 0, ""},
		{"negative width", 5, 10, -1, ""},
		{"all filled", 10, 10, 10, "##########"},
		{"none filled", 0, 10, 10, "----------"},
		{"zero total", 0, 0, 10, "----------"},
		{"half filled", 5, 10, 10, "#####-----"},
		{"overcount", 15, 10, 10, "##########"}, // filled > width → clamped
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := coverageBar(tc.used, tc.total, tc.width)
			if got != tc.want {
				t.Errorf("coverageBar(%d, %d, %d) = %q, want %q", tc.used, tc.total, tc.width, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// truncateForDiff — missing short-text branch (66.7% → 100%)
// ---------------------------------------------------------------------------

func TestTruncateForDiff_Branches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		text   string
		maxLen int
		want   string
	}{
		{"short text", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncated", "hello world", 8, "hello..."},
		{"empty", "", 5, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := truncateForDiff(tc.text, tc.maxLen)
			if got != tc.want {
				t.Errorf("truncateForDiff(%q, %d) = %q, want %q", tc.text, tc.maxLen, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// windowStartIndex — missing nil/edge branches (77.8% → higher)
// ---------------------------------------------------------------------------

func TestWindowStartIndex_NilReceiver(t *testing.T) {
	t.Parallel()
	var d *EarlyStopDetector
	got := d.windowStartIndex()
	if got != 0 {
		t.Errorf("nil.windowStartIndex() = %d, want 0", got)
	}
}

func TestWindowStartIndex_WindowLargerThanOutputs(t *testing.T) {
	t.Parallel()
	d := &EarlyStopDetector{
		Config:  EarlyStopConfig{WindowSize: 100},
		Outputs: []ModeOutput{{ModeID: "a"}, {ModeID: "b"}},
	}
	got := d.windowStartIndex()
	if got != 0 {
		t.Errorf("windowStartIndex() = %d, want 0 (window > outputs)", got)
	}
}

func TestWindowStartIndex_NormalWindow(t *testing.T) {
	t.Parallel()
	d := &EarlyStopDetector{
		Config:  EarlyStopConfig{WindowSize: 2},
		Outputs: make([]ModeOutput, 5),
	}
	got := d.windowStartIndex()
	if got != 3 { // 5 - 2 = 3
		t.Errorf("windowStartIndex() = %d, want 3", got)
	}
}

func TestWindowStartIndex_ZeroWindow(t *testing.T) {
	t.Parallel()
	d := &EarlyStopDetector{
		Config:  EarlyStopConfig{WindowSize: 0},
		Outputs: make([]ModeOutput, 3),
	}
	got := d.windowStartIndex()
	if got != 0 {
		t.Errorf("windowStartIndex(window=0) = %d, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// mergeText — missing empty-addition branch (80% → 100%)
// ---------------------------------------------------------------------------

func TestMergeText_Branches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing string
		addition string
		want     string
	}{
		{"empty existing", "", "new text", "new text"},
		{"contains addition", "hello world", "hello", "hello world"},
		{"append", "line one", "line two", "line one\nline two"},
		{"both empty", "", "", ""},
		{"empty addition", "existing", "", "existing"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mergeText(tc.existing, tc.addition)
			if got != tc.want {
				t.Errorf("mergeText(%q, %q) = %q, want %q", tc.existing, tc.addition, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// mergeList — missing empty-key/dedup branches (80% → 100%)
// ---------------------------------------------------------------------------

func TestMergeList_Branches(t *testing.T) {
	t.Parallel()

	t.Run("empty additions", func(t *testing.T) {
		t.Parallel()
		got := mergeList([]string{"a", "b"}, nil)
		if len(got) != 2 {
			t.Errorf("expected 2 items, got %d", len(got))
		}
	})

	t.Run("duplicate skipped", func(t *testing.T) {
		t.Parallel()
		got := mergeList([]string{"Alpha"}, []string{"alpha"}) // case-insensitive dedup
		if len(got) != 1 {
			t.Errorf("expected 1 item (dedup), got %d", len(got))
		}
	})

	t.Run("empty strings skipped in additions", func(t *testing.T) {
		t.Parallel()
		// existing kept as-is; empty additions skipped
		got := mergeList([]string{"a", "", " "}, []string{"", "b"})
		if len(got) != 4 { // existing ["a", "", " "] + new "b"
			t.Errorf("expected 4 items, got %d: %v", len(got), got)
		}
	})

	t.Run("new items appended", func(t *testing.T) {
		t.Parallel()
		got := mergeList([]string{"a"}, []string{"b", "c"})
		if len(got) != 3 {
			t.Errorf("expected 3 items, got %d", len(got))
		}
	})
}

// ---------------------------------------------------------------------------
// firstEvidencePointer — missing empty-findings branches (75% → 100%)
// ---------------------------------------------------------------------------

func TestFirstEvidencePointer_Branches(t *testing.T) {
	t.Parallel()

	t.Run("empty findings", func(t *testing.T) {
		t.Parallel()
		got := firstEvidencePointer(ModeOutput{})
		if got != "" {
			t.Errorf("expected empty for no findings, got %q", got)
		}
	})

	t.Run("all empty evidence", func(t *testing.T) {
		t.Parallel()
		output := ModeOutput{
			TopFindings: []Finding{
				{Finding: "bug", EvidencePointer: ""},
				{Finding: "issue", EvidencePointer: "  "},
			},
		}
		got := firstEvidencePointer(output)
		if got != "" {
			t.Errorf("expected empty for all-empty evidence, got %q", got)
		}
	})

	t.Run("finds first non-empty", func(t *testing.T) {
		t.Parallel()
		output := ModeOutput{
			TopFindings: []Finding{
				{Finding: "bug", EvidencePointer: ""},
				{Finding: "issue", EvidencePointer: "file.go:42"},
				{Finding: "other", EvidencePointer: "file.go:99"},
			},
		}
		got := firstEvidencePointer(output)
		if got != "file.go:42" {
			t.Errorf("expected 'file.go:42', got %q", got)
		}
	})
}

// ---------------------------------------------------------------------------
// summarizeFindings — missing empty/limit branches (81.8% → 100%)
// ---------------------------------------------------------------------------

func TestSummarizeFindings_Branches(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		got := summarizeFindings(nil, 5)
		if got != "" {
			t.Errorf("expected empty for nil findings, got %q", got)
		}
	})

	t.Run("unlimited", func(t *testing.T) {
		t.Parallel()
		findings := []Finding{{Finding: "a"}, {Finding: "b"}, {Finding: "c"}}
		got := summarizeFindings(findings, 0)
		if !strings.Contains(got, "a") || !strings.Contains(got, "c") {
			t.Errorf("expected all findings with limit=0, got %q", got)
		}
	})

	t.Run("with limit", func(t *testing.T) {
		t.Parallel()
		findings := []Finding{{Finding: "a"}, {Finding: "b"}, {Finding: "c"}}
		got := summarizeFindings(findings, 2)
		if !strings.Contains(got, "a") || !strings.Contains(got, "b") {
			t.Errorf("expected first 2 findings, got %q", got)
		}
		if strings.Contains(got, "c") {
			t.Errorf("expected only 2 findings, got %q", got)
		}
	})

	t.Run("skips empty finding text", func(t *testing.T) {
		t.Parallel()
		findings := []Finding{{Finding: ""}, {Finding: "real"}}
		got := summarizeFindings(findings, 0)
		if got != "real" {
			t.Errorf("expected 'real', got %q", got)
		}
	})
}

// ---------------------------------------------------------------------------
// summarizeRisks — missing empty/limit branches (81.8% → 100%)
// ---------------------------------------------------------------------------

func TestSummarizeRisks_Branches(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		if got := summarizeRisks(nil, 5); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("unlimited", func(t *testing.T) {
		t.Parallel()
		risks := []Risk{{Risk: "r1"}, {Risk: "r2"}}
		got := summarizeRisks(risks, -1)
		if !strings.Contains(got, "r1") || !strings.Contains(got, "r2") {
			t.Errorf("expected all risks, got %q", got)
		}
	})

	t.Run("skips empty risk text", func(t *testing.T) {
		t.Parallel()
		risks := []Risk{{Risk: ""}, {Risk: "real risk"}}
		got := summarizeRisks(risks, 0)
		if got != "real risk" {
			t.Errorf("expected 'real risk', got %q", got)
		}
	})
}

// ---------------------------------------------------------------------------
// summarizeRecommendations — missing empty/limit branches (81.8% → 100%)
// ---------------------------------------------------------------------------

func TestSummarizeRecommendations_Branches(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		if got := summarizeRecommendations(nil, 5); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("unlimited", func(t *testing.T) {
		t.Parallel()
		recs := []Recommendation{{Recommendation: "fix it"}, {Recommendation: "test it"}}
		got := summarizeRecommendations(recs, 0)
		if !strings.Contains(got, "fix it") || !strings.Contains(got, "test it") {
			t.Errorf("expected all recommendations, got %q", got)
		}
	})

	t.Run("skips empty text", func(t *testing.T) {
		t.Parallel()
		recs := []Recommendation{{Recommendation: ""}, {Recommendation: "action"}}
		got := summarizeRecommendations(recs, 0)
		if got != "action" {
			t.Errorf("expected 'action', got %q", got)
		}
	})
}
