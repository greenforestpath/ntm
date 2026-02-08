package panels

import (
	"testing"

	"github.com/Dicklesworthstone/ntm/internal/scoring"
)

func TestNewEffectivenessPanel(t *testing.T) {
	panel := NewEffectivenessPanel()
	if panel == nil {
		t.Fatal("NewEffectivenessPanel returned nil")
	}

	cfg := panel.Config()
	if cfg.ID != "effectiveness" {
		t.Errorf("Expected ID 'effectiveness', got %q", cfg.ID)
	}
	if cfg.Title != "Agent Effectiveness" {
		t.Errorf("Expected Title 'Agent Effectiveness', got %q", cfg.Title)
	}
}

func TestEffectivenessPanel_SetSize(t *testing.T) {
	panel := NewEffectivenessPanel()
	panel.SetSize(80, 24)

	if panel.Width() != 80 {
		t.Errorf("Expected Width 80, got %d", panel.Width())
	}
	if panel.Height() != 24 {
		t.Errorf("Expected Height 24, got %d", panel.Height())
	}
}

func TestEffectivenessPanel_FocusBlur(t *testing.T) {
	panel := NewEffectivenessPanel()
	if panel.IsFocused() {
		t.Error("Panel should not be focused initially")
	}

	panel.Focus()
	if !panel.IsFocused() {
		t.Error("Panel should be focused after Focus()")
	}

	panel.Blur()
	if panel.IsFocused() {
		t.Error("Panel should not be focused after Blur()")
	}
}

func TestEffectivenessPanel_SetData_Sorts(t *testing.T) {
	panel := NewEffectivenessPanel()
	panel.SetSize(60, 20)

	panel.SetData(EffectivenessData{
		Summaries: []*scoring.AgentSummary{
			{AgentType: "codex", AvgOverall: 0.7, TotalScores: 5},
			{AgentType: "claude", AvgOverall: 0.9, TotalScores: 10},
			{AgentType: "gemini", AvgOverall: 0.5, TotalScores: 3},
		},
		SessionName: "test_session",
		WindowDays:  14,
	}, nil)

	// Should be sorted by score descending
	if panel.data.Summaries[0].AgentType != "claude" {
		t.Errorf("Expected claude first (highest score), got %q", panel.data.Summaries[0].AgentType)
	}
	if panel.data.Summaries[1].AgentType != "codex" {
		t.Errorf("Expected codex second, got %q", panel.data.Summaries[1].AgentType)
	}
	if panel.data.Summaries[2].AgentType != "gemini" {
		t.Errorf("Expected gemini third (lowest score), got %q", panel.data.Summaries[2].AgentType)
	}
}

func TestEffectivenessPanel_SetData_SortsStableOnTie(t *testing.T) {
	panel := NewEffectivenessPanel()
	panel.SetSize(60, 20)

	// Same score, should sort alphabetically
	panel.SetData(EffectivenessData{
		Summaries: []*scoring.AgentSummary{
			{AgentType: "codex", AvgOverall: 0.8, TotalScores: 5},
			{AgentType: "claude", AvgOverall: 0.8, TotalScores: 5},
		},
	}, nil)

	// Alphabetical order on tie
	if panel.data.Summaries[0].AgentType != "claude" {
		t.Errorf("Expected claude first (alphabetical on tie), got %q", panel.data.Summaries[0].AgentType)
	}
	if panel.data.Summaries[1].AgentType != "codex" {
		t.Errorf("Expected codex second, got %q", panel.data.Summaries[1].AgentType)
	}
}

func TestEffectivenessPanel_HasData(t *testing.T) {
	panel := NewEffectivenessPanel()
	if panel.HasData() {
		t.Fatal("Expected HasData=false initially")
	}

	panel.SetData(EffectivenessData{
		Summaries: []*scoring.AgentSummary{
			{AgentType: "claude", AvgOverall: 0.9, TotalScores: 10},
		},
	}, nil)
	if !panel.HasData() {
		t.Fatal("Expected HasData=true when summaries present")
	}
}

func TestEffectivenessPanel_HasError(t *testing.T) {
	panel := NewEffectivenessPanel()
	if panel.HasError() {
		t.Fatal("Expected HasError=false initially")
	}

	panel.SetData(EffectivenessData{}, errMock)
	if !panel.HasError() {
		t.Fatal("Expected HasError=true after error set")
	}
}

func TestEffectivenessPanel_View_Empty(t *testing.T) {
	panel := NewEffectivenessPanel()
	panel.SetSize(60, 20)

	view := panel.View()
	if view == "" {
		t.Fatal("Expected non-empty View output")
	}
	if !containsString(view, "No effectiveness data") {
		t.Error("Expected empty state message in view")
	}
}

func TestEffectivenessPanel_View_WithData(t *testing.T) {
	panel := NewEffectivenessPanel()
	panel.SetSize(80, 30)

	panel.SetData(EffectivenessData{
		Summaries: []*scoring.AgentSummary{
			{
				AgentType:     "claude",
				AvgOverall:    0.85,
				AvgCompletion: 0.9,
				AvgQuality:    0.8,
				AvgEfficiency: 0.85,
				TotalScores:   15,
				Trend: &scoring.TrendAnalysis{
					Trend:         scoring.TrendImproving,
					SampleCount:   10,
					AvgScore:      0.85,
					RecentAvg:     0.88,
					EarlierAvg:    0.82,
					ChangePercent: 7.3,
				},
			},
			{
				AgentType:     "codex",
				AvgOverall:    0.35,
				AvgCompletion: 0.4,
				AvgQuality:    0.3,
				AvgEfficiency: 0.35,
				TotalScores:   8,
				Trend: &scoring.TrendAnalysis{
					Trend:         scoring.TrendDeclining,
					SampleCount:   5,
					AvgScore:      0.35,
					RecentAvg:     0.32,
					EarlierAvg:    0.38,
					ChangePercent: -15.8,
				},
			},
		},
		SessionName: "test_session",
		WindowDays:  14,
	}, nil)

	view := panel.View()
	if view == "" {
		t.Fatal("Expected non-empty View output")
	}

	// Should contain agent names
	if !containsString(view, "claude") {
		t.Error("Expected 'claude' in view")
	}
	if !containsString(view, "codex") {
		t.Error("Expected 'codex' in view")
	}

	// Should contain sample counts
	if !containsString(view, "15 samples") {
		t.Error("Expected sample count in view")
	}
}

func TestEffectivenessPanel_View_Recommendations(t *testing.T) {
	panel := NewEffectivenessPanel()
	panel.SetSize(80, 30)

	// Low performer should trigger recommendation
	panel.SetData(EffectivenessData{
		Summaries: []*scoring.AgentSummary{
			{
				AgentType:   "struggling",
				AvgOverall:  0.25,
				TotalScores: 5,
				Trend: &scoring.TrendAnalysis{
					Trend:       scoring.TrendDeclining,
					SampleCount: 5,
				},
			},
		},
	}, nil)

	view := panel.View()
	if !containsString(view, "Recommendations") {
		t.Error("Expected recommendations section for low performer")
	}
	if !containsString(view, "below threshold") {
		t.Error("Expected 'below threshold' recommendation")
	}
}

func TestEffectivenessPanel_Keybindings(t *testing.T) {
	panel := NewEffectivenessPanel()
	bindings := panel.Keybindings()

	if len(bindings) == 0 {
		t.Fatal("Expected at least one keybinding")
	}

	found := false
	for _, binding := range bindings {
		if binding.Action == "toggle_details" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected toggle_details keybinding")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactfit10", 10, "exactfit10"},
		{"this is too long", 10, "this is..."},
		{"ab", 2, "ab"},
		{"abcdef", 3, "abc"},
		{"", 5, ""},
	}

	for _, tc := range tests {
		result := truncate(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("truncate(%q, %d) = %q, expected %q", tc.input, tc.maxLen, result, tc.expected)
		}
	}
}

func TestEffectivenessPanel_ScoreColor(t *testing.T) {
	panel := NewEffectivenessPanel()

	// Test that different score ranges produce different colors
	// We can't easily test the exact colors without importing theme,
	// but we can verify the function doesn't panic
	panel.scoreColor(0.9)  // High
	panel.scoreColor(0.7)  // Medium
	panel.scoreColor(0.5)  // Low-medium
	panel.scoreColor(0.2)  // Low
	panel.scoreColor(0.0)  // Zero
	panel.scoreColor(1.0)  // Perfect
}

func TestEffectivenessPanel_TrendArrow(t *testing.T) {
	panel := NewEffectivenessPanel()
	panel.SetSize(60, 20)

	// Nil trend
	arrow := panel.trendArrow(nil)
	if arrow == "" {
		t.Error("Expected non-empty arrow for nil trend")
	}

	// Insufficient samples
	arrow = panel.trendArrow(&scoring.TrendAnalysis{SampleCount: 1})
	if arrow == "" {
		t.Error("Expected non-empty arrow for insufficient samples")
	}

	// Improving trend
	arrow = panel.trendArrow(&scoring.TrendAnalysis{
		Trend:       scoring.TrendImproving,
		SampleCount: 5,
	})
	if !containsString(arrow, "↑") {
		t.Error("Expected up arrow for improving trend")
	}

	// Declining trend
	arrow = panel.trendArrow(&scoring.TrendAnalysis{
		Trend:       scoring.TrendDeclining,
		SampleCount: 5,
	})
	if !containsString(arrow, "↓") {
		t.Error("Expected down arrow for declining trend")
	}

	// Stable trend
	arrow = panel.trendArrow(&scoring.TrendAnalysis{
		Trend:       scoring.TrendStable,
		SampleCount: 5,
	})
	if !containsString(arrow, "→") {
		t.Error("Expected right arrow for stable trend")
	}
}

func TestEffectivenessPanel_RenderScoreBar(t *testing.T) {
	panel := NewEffectivenessPanel()
	panel.SetSize(60, 20)

	bar := panel.renderScoreBar(0.5)
	if bar == "" {
		t.Error("Expected non-empty score bar")
	}
	if !containsString(bar, "[") || !containsString(bar, "]") {
		t.Error("Expected bar to have brackets")
	}

	// Edge cases
	bar = panel.renderScoreBar(0.0)
	if bar == "" {
		t.Error("Expected non-empty score bar for 0.0")
	}

	bar = panel.renderScoreBar(1.0)
	if bar == "" {
		t.Error("Expected non-empty score bar for 1.0")
	}
}

// containsString checks if haystack contains needle.
func containsString(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 &&
		(haystack == needle || len(haystack) >= len(needle) &&
		(haystack[:len(needle)] == needle ||
		 containsString(haystack[1:], needle)))
}

// errMock is a simple error for testing.
var errMock = &mockError{}

type mockError struct{}

func (e *mockError) Error() string { return "mock error" }
