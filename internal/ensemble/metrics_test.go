package ensemble

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewObservabilityMetrics(t *testing.T) {
	metrics := NewObservabilityMetrics(nil)
	if metrics == nil {
		t.Fatal("NewObservabilityMetrics returned nil")
	}
	if metrics.Coverage == nil {
		t.Error("Coverage tracker should be initialized")
	}
	if metrics.Velocity == nil {
		t.Error("Velocity tracker should be initialized")
	}
	if metrics.Conflicts == nil {
		t.Error("Conflicts tracker should be initialized")
	}
}

func TestObservabilityMetrics_ComputeFromOutputs(t *testing.T) {
	metrics := NewObservabilityMetrics(nil)

	outputs := []ModeOutput{
		{
			ModeID: "mode-a",
			Thesis: "Test thesis A",
			TopFindings: []Finding{
				{Finding: "Finding A1", Confidence: 0.8},
				{Finding: "Finding A2", Confidence: 0.7},
			},
			Recommendations: []Recommendation{
				{Recommendation: "Rec A1"},
			},
		},
		{
			ModeID: "mode-b",
			Thesis: "Test thesis B",
			TopFindings: []Finding{
				{Finding: "Finding B1", Confidence: 0.9},
				{Finding: "Finding A1", Confidence: 0.8}, // Shared with mode-a
			},
		},
	}

	err := metrics.ComputeFromOutputs(outputs, nil, nil)
	if err != nil {
		t.Fatalf("ComputeFromOutputs failed: %v", err)
	}

	// Verify mode IDs are sorted
	if len(metrics.ModeIDs) != 2 {
		t.Errorf("expected 2 mode IDs, got %d", len(metrics.ModeIDs))
	}
	if metrics.ModeIDs[0] != "mode-a" || metrics.ModeIDs[1] != "mode-b" {
		t.Errorf("expected sorted mode IDs [mode-a, mode-b], got %v", metrics.ModeIDs)
	}

	// Verify redundancy was computed
	if metrics.Redundancy == nil {
		t.Error("Redundancy should be computed")
	}
	if len(metrics.Redundancy.PairwiseScores) != 1 {
		t.Errorf("expected 1 pairwise score, got %d", len(metrics.Redundancy.PairwiseScores))
	}

	// Verify computed at is set
	if metrics.ComputedAt.IsZero() {
		t.Error("ComputedAt should be set")
	}
}

func TestObservabilityMetrics_GetReport(t *testing.T) {
	metrics := NewObservabilityMetrics(nil)

	outputs := []ModeOutput{
		{
			ModeID:      "mode-a",
			Thesis:      "Test thesis",
			TopFindings: []Finding{{Finding: "Finding 1", Confidence: 0.8}},
		},
		{
			ModeID:      "mode-b",
			Thesis:      "Test thesis",
			TopFindings: []Finding{{Finding: "Finding 2", Confidence: 0.7}},
		},
	}

	_ = metrics.ComputeFromOutputs(outputs, nil, nil)
	report := metrics.GetReport()

	if report == nil {
		t.Fatal("GetReport returned nil")
	}
	if report.ModeCount != 2 {
		t.Errorf("expected ModeCount 2, got %d", report.ModeCount)
	}
	if report.ComputedAt == "" {
		t.Error("ComputedAt should be set")
	}
	if report.Coverage == nil {
		t.Error("Coverage should be present in report")
	}
	if report.Velocity == nil {
		t.Error("Velocity should be present in report")
	}
	if report.Redundancy == nil {
		t.Error("Redundancy should be present in report")
	}
}

func TestObservabilityMetrics_Determinism(t *testing.T) {
	// Verify that metrics are deterministic across runs
	outputs := []ModeOutput{
		{ModeID: "z-mode", Thesis: "Z", TopFindings: []Finding{{Finding: "Z1"}}},
		{ModeID: "a-mode", Thesis: "A", TopFindings: []Finding{{Finding: "A1"}}},
		{ModeID: "m-mode", Thesis: "M", TopFindings: []Finding{{Finding: "M1"}}},
	}

	metrics1 := NewObservabilityMetrics(nil)
	_ = metrics1.ComputeFromOutputs(outputs, nil, nil)
	report1 := metrics1.GetReport()

	metrics2 := NewObservabilityMetrics(nil)
	_ = metrics2.ComputeFromOutputs(outputs, nil, nil)
	report2 := metrics2.GetReport()

	// Mode IDs should be sorted consistently
	for i := range metrics1.ModeIDs {
		if metrics1.ModeIDs[i] != metrics2.ModeIDs[i] {
			t.Errorf("ModeID ordering not deterministic at index %d", i)
		}
	}

	// Redundancy overall score should be identical
	if report1.Redundancy.OverallScore != report2.Redundancy.OverallScore {
		t.Errorf("Redundancy score not deterministic: %f vs %f",
			report1.Redundancy.OverallScore, report2.Redundancy.OverallScore)
	}
}

func TestObservabilityMetrics_WithBudget(t *testing.T) {
	metrics := NewObservabilityMetrics(nil)

	outputs := []ModeOutput{
		{ModeID: "mode-a", Thesis: "A", TopFindings: []Finding{{Finding: "F1"}}},
	}

	budgetState := &BudgetState{
		TotalSpent:    50000,
		TotalLimit:    100000,
		PerAgentSpent: map[string]int{"mode-a": 50000},
	}

	_ = metrics.ComputeFromOutputs(outputs, budgetState, nil)
	report := metrics.GetReport()

	if report.BudgetEfficiency == nil {
		t.Error("BudgetEfficiency should be present when budget is provided")
	}
	if report.BudgetEfficiency.TotalTokens != 50000 {
		t.Errorf("expected TotalTokens 50000, got %d", report.BudgetEfficiency.TotalTokens)
	}
}

func TestObservabilityMetrics_WithAuditReport(t *testing.T) {
	metrics := NewObservabilityMetrics(nil)

	outputs := []ModeOutput{
		{ModeID: "mode-a", Thesis: "A", TopFindings: []Finding{{Finding: "F1"}}},
		{ModeID: "mode-b", Thesis: "B", TopFindings: []Finding{{Finding: "F2"}}},
	}

	auditReport := &AuditReport{
		Conflicts: []DetailedConflict{
			{
				Topic:    "approach",
				Severity: ConflictHigh,
				Positions: []ConflictPosition{
					{ModeID: "mode-a", Position: "use X"},
					{ModeID: "mode-b", Position: "use Y"},
				},
			},
		},
	}

	_ = metrics.ComputeFromOutputs(outputs, nil, auditReport)

	if metrics.Conflicts.Source != "auditor" {
		t.Errorf("expected conflict source 'auditor', got '%s'", metrics.Conflicts.Source)
	}
}

func TestObservabilityMetrics_Render(t *testing.T) {
	metrics := NewObservabilityMetrics(nil)

	outputs := []ModeOutput{
		{ModeID: "mode-a", Thesis: "A", TopFindings: []Finding{{Finding: "F1", Confidence: 0.8}}},
		{ModeID: "mode-b", Thesis: "B", TopFindings: []Finding{{Finding: "F2", Confidence: 0.7}}},
	}

	_ = metrics.ComputeFromOutputs(outputs, nil, nil)
	rendered := metrics.Render()

	// Should contain key sections
	if !strings.Contains(rendered, "OBSERVABILITY METRICS DASHBOARD") {
		t.Error("rendered output should contain dashboard title")
	}
	if !strings.Contains(rendered, "Category Coverage") {
		t.Error("rendered output should contain coverage section")
	}
	if !strings.Contains(rendered, "Findings Velocity") {
		t.Error("rendered output should contain velocity section")
	}
	if !strings.Contains(rendered, "Redundancy Analysis") {
		t.Error("rendered output should contain redundancy section")
	}
	if !strings.Contains(rendered, "Conflict Analysis") {
		t.Error("rendered output should contain conflict section")
	}
}

func TestObservabilityMetrics_PostRunReport(t *testing.T) {
	metrics := NewObservabilityMetrics(nil)

	outputs := []ModeOutput{
		{ModeID: "mode-a", Thesis: "A", TopFindings: []Finding{{Finding: "F1"}}},
	}

	_ = metrics.ComputeFromOutputs(outputs, nil, nil)
	report := metrics.PostRunReport()

	// Should be markdown format
	if !strings.HasPrefix(report, "# Ensemble Metrics Report") {
		t.Error("post-run report should be in markdown format")
	}
	if !strings.Contains(report, "## Category Coverage") {
		t.Error("post-run report should contain coverage section")
	}
	if !strings.Contains(report, "## Findings Velocity") {
		t.Error("post-run report should contain velocity section")
	}
}

func TestObservabilityMetrics_NilSafety(t *testing.T) {
	var metrics *ObservabilityMetrics

	// Should not panic
	err := metrics.ComputeFromOutputs(nil, nil, nil)
	if err == nil {
		t.Error("expected error for nil metrics")
	}

	report := metrics.GetReport()
	if report == nil {
		t.Error("GetReport should return non-nil even for nil metrics")
	}

	rendered := metrics.Render()
	if rendered == "" {
		t.Error("Render should return non-empty for nil metrics")
	}
}

func TestMetricsReport_JSONSerialization(t *testing.T) {
	metrics := NewObservabilityMetrics(nil)

	outputs := []ModeOutput{
		{ModeID: "mode-a", Thesis: "A", TopFindings: []Finding{{Finding: "F1"}}},
		{ModeID: "mode-b", Thesis: "B", TopFindings: []Finding{{Finding: "F2"}}},
	}

	_ = metrics.ComputeFromOutputs(outputs, nil, nil)
	report := metrics.GetReport()

	// Should serialize to JSON without error
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// Should deserialize back
	var parsed MetricsReport
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if parsed.ModeCount != 2 {
		t.Errorf("expected ModeCount 2 after round-trip, got %d", parsed.ModeCount)
	}
}

func TestBudgetEfficiency_Ratings(t *testing.T) {
	tests := []struct {
		name           string
		findings       int
		tokens         int
		expectedRating string
	}{
		{"excellent", 30, 5000, "excellent"},
		{"good", 15, 5000, "good"},
		{"acceptable", 7, 5000, "acceptable"},
		{"low", 2, 5000, "low"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			metrics := NewObservabilityMetrics(nil)

			// Create outputs with specified number of findings
			outputs := make([]ModeOutput, 1)
			findings := make([]Finding, tc.findings)
			for i := range findings {
				findings[i] = Finding{Finding: "F" + string(rune('0'+i))}
			}
			outputs[0] = ModeOutput{
				ModeID:      "mode-a",
				Thesis:      "Test",
				TopFindings: findings,
			}

			budgetState := &BudgetState{
				TotalSpent:    tc.tokens,
				TotalLimit:    100000,
				PerAgentSpent: map[string]int{"mode-a": tc.tokens},
			}

			_ = metrics.ComputeFromOutputs(outputs, budgetState, nil)
			report := metrics.GetReport()

			if report.BudgetEfficiency == nil {
				t.Fatal("BudgetEfficiency should be present")
			}
			if report.BudgetEfficiency.EfficiencyRating != tc.expectedRating {
				t.Errorf("expected rating %s, got %s (findings/k: %.2f)",
					tc.expectedRating, report.BudgetEfficiency.EfficiencyRating,
					report.BudgetEfficiency.FindingsPerKTok)
			}
		})
	}
}

func TestObservabilityMetrics_SuggestionsGenerated(t *testing.T) {
	metrics := NewObservabilityMetrics(nil)

	// Create outputs with high redundancy
	outputs := []ModeOutput{
		{
			ModeID: "mode-a",
			Thesis: "Same thesis",
			TopFindings: []Finding{
				{Finding: "Shared finding"},
				{Finding: "Shared finding 2"},
			},
		},
		{
			ModeID: "mode-b",
			Thesis: "Same thesis",
			TopFindings: []Finding{
				{Finding: "Shared finding"},
				{Finding: "Shared finding 2"},
			},
		},
	}

	_ = metrics.ComputeFromOutputs(outputs, nil, nil)
	report := metrics.GetReport()

	if len(report.Suggestions) == 0 {
		t.Error("expected suggestions for high redundancy outputs")
	}

	foundRedundancySuggestion := false
	for _, s := range report.Suggestions {
		if strings.Contains(strings.ToLower(s), "redundan") {
			foundRedundancySuggestion = true
			break
		}
	}
	if !foundRedundancySuggestion {
		t.Error("expected redundancy-related suggestion")
	}
}

func TestObservabilityMetrics_ComputedAtTimestamp(t *testing.T) {
	metrics := NewObservabilityMetrics(nil)
	before := time.Now().UTC()

	outputs := []ModeOutput{
		{ModeID: "mode-a", Thesis: "A", TopFindings: []Finding{{Finding: "F1"}}},
	}

	_ = metrics.ComputeFromOutputs(outputs, nil, nil)
	after := time.Now().UTC()

	if metrics.ComputedAt.Before(before) || metrics.ComputedAt.After(after) {
		t.Error("ComputedAt should be set to current time during computation")
	}
}
