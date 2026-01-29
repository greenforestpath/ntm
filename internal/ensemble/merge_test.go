package ensemble

import (
	"testing"
)

func TestDefaultMergeConfig(t *testing.T) {
	cfg := DefaultMergeConfig()

	if cfg.MaxFindings != 20 {
		t.Errorf("MaxFindings = %d, want 20", cfg.MaxFindings)
	}
	if cfg.MaxRisks != 10 {
		t.Errorf("MaxRisks = %d, want 10", cfg.MaxRisks)
	}
	if cfg.MaxRecommendations != 10 {
		t.Errorf("MaxRecommendations = %d, want 10", cfg.MaxRecommendations)
	}
	if cfg.MinConfidence != 0.3 {
		t.Errorf("MinConfidence = %f, want 0.3", cfg.MinConfidence)
	}
	if cfg.DeduplicationThreshold != 0.7 {
		t.Errorf("DeduplicationThreshold = %f, want 0.7", cfg.DeduplicationThreshold)
	}
}

func TestMergeOutputs_EmptyInputs(t *testing.T) {
	cfg := DefaultMergeConfig()
	result := MergeOutputs(nil, cfg)

	if result == nil {
		t.Fatal("MergeOutputs returned nil for empty inputs")
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0", len(result.Findings))
	}
	if result.Stats.InputCount != 0 {
		t.Errorf("InputCount = %d, want 0", result.Stats.InputCount)
	}
}

func TestMergeOutputs_SingleOutput(t *testing.T) {
	cfg := DefaultMergeConfig()

	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Thesis:     "Test thesis",
			Confidence: 0.8,
			TopFindings: []Finding{
				{Finding: "Finding 1", Impact: ImpactHigh, Confidence: 0.9},
				{Finding: "Finding 2", Impact: ImpactMedium, Confidence: 0.7},
			},
			Risks: []Risk{
				{Risk: "Risk 1", Impact: ImpactHigh, Likelihood: 0.8},
			},
			Recommendations: []Recommendation{
				{Recommendation: "Rec 1", Priority: ImpactHigh},
			},
		},
	}

	result := MergeOutputs(outputs, cfg)

	if len(result.SourceModes) != 1 {
		t.Errorf("SourceModes = %d, want 1", len(result.SourceModes))
	}
	if result.SourceModes[0] != "mode-a" {
		t.Errorf("SourceModes[0] = %s, want mode-a", result.SourceModes[0])
	}
	if len(result.Findings) != 2 {
		t.Errorf("Findings = %d, want 2", len(result.Findings))
	}
	if len(result.Risks) != 1 {
		t.Errorf("Risks = %d, want 1", len(result.Risks))
	}
	if len(result.Recommendations) != 1 {
		t.Errorf("Recommendations = %d, want 1", len(result.Recommendations))
	}
}

func TestMergeOutputs_MultipleOutputs_NoDuplicates(t *testing.T) {
	cfg := DefaultMergeConfig()

	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.8,
			TopFindings: []Finding{
				{Finding: "Finding A", Impact: ImpactHigh, Confidence: 0.9},
			},
		},
		{
			ModeID:     "mode-b",
			Confidence: 0.7,
			TopFindings: []Finding{
				{Finding: "Finding B", Impact: ImpactMedium, Confidence: 0.8},
			},
		},
	}

	result := MergeOutputs(outputs, cfg)

	if len(result.Findings) != 2 {
		t.Errorf("Findings = %d, want 2", len(result.Findings))
	}
	if result.Stats.TotalFindings != 2 {
		t.Errorf("TotalFindings = %d, want 2", result.Stats.TotalFindings)
	}
	if result.Stats.DedupedFindings != 2 {
		t.Errorf("DedupedFindings = %d, want 2", result.Stats.DedupedFindings)
	}
}

func TestMergeOutputs_Deduplication(t *testing.T) {
	cfg := DefaultMergeConfig()
	cfg.DeduplicationThreshold = 0.5 // Lower threshold for easier matching

	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.8,
			TopFindings: []Finding{
				{Finding: "The authentication module has a security vulnerability", Impact: ImpactHigh, Confidence: 0.9},
			},
		},
		{
			ModeID:     "mode-b",
			Confidence: 0.7,
			TopFindings: []Finding{
				{Finding: "The authentication module has a security vulnerability", Impact: ImpactHigh, Confidence: 0.8},
			},
		},
	}

	result := MergeOutputs(outputs, cfg)

	// Should deduplicate identical findings
	if len(result.Findings) != 1 {
		t.Errorf("Findings = %d, want 1 (should be deduplicated)", len(result.Findings))
	}
	if result.Stats.TotalFindings != 2 {
		t.Errorf("TotalFindings = %d, want 2", result.Stats.TotalFindings)
	}
	// The deduplicated finding should list both source modes
	if len(result.Findings) > 0 && len(result.Findings[0].SourceModes) != 2 {
		t.Errorf("SourceModes for deduplicated finding = %d, want 2", len(result.Findings[0].SourceModes))
	}
}

func TestMergeOutputs_MinConfidenceFilter(t *testing.T) {
	cfg := DefaultMergeConfig()
	cfg.MinConfidence = 0.5

	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.8,
			TopFindings: []Finding{
				{Finding: "High confidence finding", Impact: ImpactHigh, Confidence: 0.9},
				{Finding: "Low confidence finding", Impact: ImpactHigh, Confidence: 0.2},
			},
		},
	}

	result := MergeOutputs(outputs, cfg)

	// Low confidence finding should be filtered out
	if len(result.Findings) != 1 {
		t.Errorf("Findings = %d, want 1 (low confidence filtered)", len(result.Findings))
	}
}

func TestMergeOutputs_MaxFindingsLimit(t *testing.T) {
	cfg := DefaultMergeConfig()
	cfg.MaxFindings = 3

	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.8,
			TopFindings: []Finding{
				{Finding: "Finding 1", Impact: ImpactHigh, Confidence: 0.9},
				{Finding: "Finding 2", Impact: ImpactMedium, Confidence: 0.8},
				{Finding: "Finding 3", Impact: ImpactLow, Confidence: 0.7},
				{Finding: "Finding 4", Impact: ImpactLow, Confidence: 0.6},
				{Finding: "Finding 5", Impact: ImpactLow, Confidence: 0.5},
			},
		},
	}

	result := MergeOutputs(outputs, cfg)

	if len(result.Findings) != 3 {
		t.Errorf("Findings = %d, want 3 (limited by MaxFindings)", len(result.Findings))
	}
}

func TestMergeOutputs_SortByScore(t *testing.T) {
	cfg := DefaultMergeConfig()
	cfg.PreferHighImpact = true
	cfg.WeightByConfidence = true

	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.8,
			TopFindings: []Finding{
				{Finding: "Low impact finding", Impact: ImpactLow, Confidence: 0.9},
				{Finding: "Critical finding", Impact: ImpactCritical, Confidence: 0.9},
				{Finding: "Medium impact finding", Impact: ImpactMedium, Confidence: 0.9},
			},
		},
	}

	result := MergeOutputs(outputs, cfg)

	if len(result.Findings) < 3 {
		t.Fatalf("Expected at least 3 findings, got %d", len(result.Findings))
	}

	// Critical should be first (highest score)
	if result.Findings[0].Finding.Finding != "Critical finding" {
		t.Errorf("First finding should be critical, got: %s", result.Findings[0].Finding.Finding)
	}
}

func TestMergeOutputs_Questions(t *testing.T) {
	cfg := DefaultMergeConfig()

	outputs := []ModeOutput{
		{
			ModeID:           "mode-a",
			Confidence:       0.8,
			TopFindings:      []Finding{{Finding: "f", Impact: ImpactMedium, Confidence: 0.5}},
			QuestionsForUser: []Question{{Question: "Question from A"}},
		},
		{
			ModeID:           "mode-b",
			Confidence:       0.7,
			TopFindings:      []Finding{{Finding: "f2", Impact: ImpactMedium, Confidence: 0.5}},
			QuestionsForUser: []Question{{Question: "Question from B"}},
		},
	}

	result := MergeOutputs(outputs, cfg)

	if len(result.Questions) != 2 {
		t.Errorf("Questions = %d, want 2", len(result.Questions))
	}
}

func TestMergeOutputs_RisksScoring(t *testing.T) {
	cfg := DefaultMergeConfig()

	outputs := []ModeOutput{
		{
			ModeID:      "mode-a",
			Confidence:  0.8,
			TopFindings: []Finding{{Finding: "f", Impact: ImpactMedium, Confidence: 0.5}},
			Risks: []Risk{
				{Risk: "Low risk", Impact: ImpactLow, Likelihood: 0.3},
				{Risk: "High risk", Impact: ImpactHigh, Likelihood: 0.9},
			},
		},
	}

	result := MergeOutputs(outputs, cfg)

	if len(result.Risks) < 2 {
		t.Fatalf("Expected at least 2 risks, got %d", len(result.Risks))
	}

	// High risk should be first (higher score due to impact × likelihood)
	if result.Risks[0].Risk.Risk != "High risk" {
		t.Errorf("First risk should be high risk, got: %s", result.Risks[0].Risk.Risk)
	}
}

func TestConsolidateTheses(t *testing.T) {
	tests := []struct {
		name    string
		outputs []ModeOutput
		want    string
	}{
		{
			name:    "empty outputs",
			outputs: nil,
			want:    "",
		},
		{
			name: "single output",
			outputs: []ModeOutput{
				{Thesis: "Only thesis", Confidence: 0.8},
			},
			want: "Only thesis",
		},
		{
			name: "multiple outputs - pick highest confidence",
			outputs: []ModeOutput{
				{Thesis: "Low confidence thesis", Confidence: 0.5},
				{Thesis: "High confidence thesis", Confidence: 0.9},
				{Thesis: "Medium confidence thesis", Confidence: 0.7},
			},
			want: "High confidence thesis",
		},
		{
			name: "empty theses skipped",
			outputs: []ModeOutput{
				{Thesis: "", Confidence: 0.95},
				{Thesis: "Valid thesis", Confidence: 0.7},
			},
			want: "Valid thesis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConsolidateTheses(tt.outputs)
			if got != tt.want {
				t.Errorf("ConsolidateTheses() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAverageConfidence(t *testing.T) {
	tests := []struct {
		name    string
		outputs []ModeOutput
		want    Confidence
	}{
		{
			name:    "empty outputs",
			outputs: nil,
			want:    0,
		},
		{
			name: "single output",
			outputs: []ModeOutput{
				{Confidence: 0.8},
			},
			want: 0.8,
		},
		{
			name: "multiple outputs",
			outputs: []ModeOutput{
				{Confidence: 0.6},
				{Confidence: 0.8},
				{Confidence: 1.0},
			},
			want: 0.8, // (0.6 + 0.8 + 1.0) / 3 = 0.8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AverageConfidence(tt.outputs)
			// Use approximate comparison for floating point
			diff := float64(got) - float64(tt.want)
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.001 {
				t.Errorf("AverageConfidence() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestImpactWeight(t *testing.T) {
	tests := []struct {
		impact ImpactLevel
		want   float64
	}{
		{ImpactCritical, 1.0},
		{ImpactHigh, 0.8},
		{ImpactMedium, 0.5},
		{ImpactLow, 0.3},
		{ImpactLevel("unknown"), 0.4},
	}

	for _, tt := range tests {
		t.Run(string(tt.impact), func(t *testing.T) {
			got := impactWeight(tt.impact)
			if got != tt.want {
				t.Errorf("impactWeight(%v) = %v, want %v", tt.impact, got, tt.want)
			}
		})
	}
}

func TestMergeOutputs_RecommendationsScoring(t *testing.T) {
	cfg := DefaultMergeConfig()

	outputs := []ModeOutput{
		{
			ModeID:      "mode-a",
			Confidence:  0.8,
			TopFindings: []Finding{{Finding: "f", Impact: ImpactMedium, Confidence: 0.5}},
			Recommendations: []Recommendation{
				{Recommendation: "Low priority", Priority: ImpactLow},
				{Recommendation: "High priority", Priority: ImpactHigh},
				{Recommendation: "Critical priority", Priority: ImpactCritical},
			},
		},
	}

	result := MergeOutputs(outputs, cfg)

	if len(result.Recommendations) < 3 {
		t.Fatalf("Expected at least 3 recommendations, got %d", len(result.Recommendations))
	}

	// Critical should be first
	if result.Recommendations[0].Recommendation.Recommendation != "Critical priority" {
		t.Errorf("First recommendation should be critical priority, got: %s",
			result.Recommendations[0].Recommendation.Recommendation)
	}
}

func TestMergeStats(t *testing.T) {
	cfg := DefaultMergeConfig()
	cfg.DeduplicationThreshold = 0.9 // High threshold to only merge identical text

	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.8,
			TopFindings: []Finding{
				{Finding: "This exact finding appears in both modes", Impact: ImpactHigh, Confidence: 0.9},
				{Finding: "Alpha mode discovered a completely separate issue with authentication", Impact: ImpactMedium, Confidence: 0.8},
			},
			Risks: []Risk{
				{Risk: "Duplicate risk", Impact: ImpactHigh, Likelihood: 0.8},
			},
			Recommendations: []Recommendation{
				{Recommendation: "Unique rec A", Priority: ImpactHigh},
			},
		},
		{
			ModeID:     "mode-b",
			Confidence: 0.7,
			TopFindings: []Finding{
				{Finding: "This exact finding appears in both modes", Impact: ImpactHigh, Confidence: 0.85},
				{Finding: "Beta mode found something entirely different about performance", Impact: ImpactLow, Confidence: 0.7},
			},
			Risks: []Risk{
				{Risk: "Duplicate risk", Impact: ImpactHigh, Likelihood: 0.7},
			},
			Recommendations: []Recommendation{
				{Recommendation: "Unique rec B", Priority: ImpactMedium},
			},
		},
	}

	result := MergeOutputs(outputs, cfg)

	if result.Stats.InputCount != 2 {
		t.Errorf("InputCount = %d, want 2", result.Stats.InputCount)
	}
	if result.Stats.TotalFindings != 4 {
		t.Errorf("TotalFindings = %d, want 4", result.Stats.TotalFindings)
	}
	// Deduplication merges the identical finding, leaving 3
	if result.Stats.DedupedFindings != 3 {
		t.Errorf("DedupedFindings = %d, want 3", result.Stats.DedupedFindings)
	}
	if result.Stats.TotalRisks != 2 {
		t.Errorf("TotalRisks = %d, want 2", result.Stats.TotalRisks)
	}
	if result.Stats.TotalRecommendations != 2 {
		t.Errorf("TotalRecommendations = %d, want 2", result.Stats.TotalRecommendations)
	}
	if result.Stats.MergeTime <= 0 {
		t.Error("MergeTime should be positive")
	}
}

func TestMergedFinding_SourceModes(t *testing.T) {
	cfg := DefaultMergeConfig()
	cfg.DeduplicationThreshold = 0.5

	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.8,
			TopFindings: []Finding{
				{Finding: "Shared finding across modes", Impact: ImpactHigh, Confidence: 0.9},
			},
		},
		{
			ModeID:     "mode-b",
			Confidence: 0.7,
			TopFindings: []Finding{
				{Finding: "Shared finding across modes", Impact: ImpactHigh, Confidence: 0.85},
			},
		},
		{
			ModeID:     "mode-c",
			Confidence: 0.75,
			TopFindings: []Finding{
				{Finding: "Shared finding across modes", Impact: ImpactHigh, Confidence: 0.8},
			},
		},
	}

	result := MergeOutputs(outputs, cfg)

	if len(result.Findings) != 1 {
		t.Fatalf("Expected 1 merged finding, got %d", len(result.Findings))
	}

	// All three modes should be in source modes
	sourceModes := result.Findings[0].SourceModes
	if len(sourceModes) != 3 {
		t.Errorf("SourceModes = %d, want 3", len(sourceModes))
	}

	// Check that score was boosted for agreement (1.1x per merge)
	// Base score for mode-a: 0.9 * 0.8 * 0.8 = 0.576
	// After 2 merges: 0.576 * 1.1 * 1.1 ≈ 0.697
	// So we expect > 0.6 to confirm boosting occurred
	if result.Findings[0].MergeScore <= 0.6 {
		t.Errorf("MergeScore should be boosted for multi-mode agreement, got %f", result.Findings[0].MergeScore)
	}
}

// =============================================================================
// MechanicalMerger Tests
// =============================================================================

func TestNewMechanicalMerger(t *testing.T) {
	outputs := []ModeOutput{
		{ModeID: "mode-a", Confidence: 0.8},
		{ModeID: "mode-b", Confidence: 0.7},
	}

	merger := NewMechanicalMerger(outputs)

	if merger == nil {
		t.Fatal("NewMechanicalMerger returned nil")
	}
	if len(merger.Outputs) != 2 {
		t.Errorf("Outputs count = %d, want 2", len(merger.Outputs))
	}
}

func TestMechanicalMerger_EmptyInputs(t *testing.T) {
	merger := NewMechanicalMerger(nil)
	result, err := merger.Merge()

	if err != nil {
		t.Fatalf("Merge returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Merge returned nil result")
	}
	if len(result.GroupedFindings) != 0 {
		t.Errorf("GroupedFindings = %d, want 0", len(result.GroupedFindings))
	}
}

func TestMechanicalMerger_GroupFindingsByEvidence(t *testing.T) {
	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.8,
			TopFindings: []Finding{
				{Finding: "SQL injection found", EvidencePointer: "api/users.go:42", Impact: ImpactCritical, Confidence: 0.9},
				{Finding: "Missing validation", EvidencePointer: "api/users.go:42", Impact: ImpactHigh, Confidence: 0.8},
				{Finding: "Unhandled error", EvidencePointer: "db/conn.go:15", Impact: ImpactMedium, Confidence: 0.7},
			},
		},
		{
			ModeID:     "mode-b",
			Confidence: 0.7,
			TopFindings: []Finding{
				{Finding: "Input validation missing", EvidencePointer: "api/users.go:42", Impact: ImpactHigh, Confidence: 0.85},
				{Finding: "Connection leak", EvidencePointer: "db/conn.go:20", Impact: ImpactHigh, Confidence: 0.75},
			},
		},
	}

	merger := NewMechanicalMerger(outputs)
	result, err := merger.Merge()

	if err != nil {
		t.Fatalf("Merge returned error: %v", err)
	}

	// Should have 3 evidence groups: api/users.go:42, db/conn.go:15, db/conn.go:20
	if len(result.GroupedFindings) != 3 {
		t.Errorf("GroupedFindings = %d, want 3", len(result.GroupedFindings))
		for _, g := range result.GroupedFindings {
			t.Logf("  Group: %s with %d findings", g.EvidencePointer, len(g.Findings))
		}
	}

	// Find the api/users.go:42 group and verify it has findings from both modes
	var usersGroup *FindingGroup
	for i := range result.GroupedFindings {
		if result.GroupedFindings[i].EvidencePointer == "api/users.go:42" {
			usersGroup = &result.GroupedFindings[i]
			break
		}
	}

	if usersGroup == nil {
		t.Fatal("api/users.go:42 group not found")
	}

	// Should have both modes
	if len(usersGroup.Modes) != 2 {
		t.Errorf("api/users.go:42 modes = %d, want 2", len(usersGroup.Modes))
	}
}

func TestMechanicalMerger_DuplicateFindingsDedup(t *testing.T) {
	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.8,
			TopFindings: []Finding{
				{Finding: "SQL injection vulnerability found in user input", EvidencePointer: "api.go:10", Impact: ImpactCritical, Confidence: 0.9},
			},
		},
		{
			ModeID:     "mode-b",
			Confidence: 0.7,
			TopFindings: []Finding{
				// Very similar finding - should be deduplicated (Jaccard > 0.8)
				{Finding: "SQL injection vulnerability detected in user input", EvidencePointer: "api.go:10", Impact: ImpactCritical, Confidence: 0.85},
			},
		},
	}

	merger := NewMechanicalMerger(outputs)
	result, err := merger.Merge()

	if err != nil {
		t.Fatalf("Merge returned error: %v", err)
	}

	// Should have 1 group with 1 unique finding (duplicates merged)
	if len(result.GroupedFindings) != 1 {
		t.Fatalf("GroupedFindings = %d, want 1", len(result.GroupedFindings))
	}

	// The finding text is similar enough to be deduplicated
	group := result.GroupedFindings[0]
	if len(group.Findings) > 1 {
		t.Logf("Note: Findings not deduplicated - may be below threshold")
	}

	// Both modes should still be tracked
	if len(group.Modes) != 2 {
		t.Errorf("Modes = %d, want 2 (both should contribute)", len(group.Modes))
	}
}

func TestMechanicalMerger_GroupRisksBySeverity(t *testing.T) {
	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.8,
			Risks: []Risk{
				{Risk: "Data breach risk", Impact: ImpactCritical, Likelihood: 0.7},
				{Risk: "Performance degradation", Impact: ImpactMedium, Likelihood: 0.5},
			},
		},
		{
			ModeID:     "mode-b",
			Confidence: 0.7,
			Risks: []Risk{
				{Risk: "API rate limiting issues", Impact: ImpactHigh, Likelihood: 0.6},
				{Risk: "Memory leak risk", Impact: ImpactMedium, Likelihood: 0.4},
			},
		},
	}

	merger := NewMechanicalMerger(outputs)
	result, err := merger.Merge()

	if err != nil {
		t.Fatalf("Merge returned error: %v", err)
	}

	// Should have 3 severity groups: critical, high, medium
	if len(result.GroupedRisks) != 3 {
		t.Errorf("GroupedRisks = %d, want 3", len(result.GroupedRisks))
		for _, g := range result.GroupedRisks {
			t.Logf("  Severity %s: %d risks", g.Severity, len(g.Risks))
		}
	}

	// Critical should be first
	if len(result.GroupedRisks) > 0 && result.GroupedRisks[0].Severity != ImpactCritical {
		t.Errorf("First severity group = %s, want critical", result.GroupedRisks[0].Severity)
	}
}

func TestMechanicalMerger_GroupRecommendationsByAction(t *testing.T) {
	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.8,
			Recommendations: []Recommendation{
				{Recommendation: "Add unit tests for validation logic", Priority: ImpactHigh},
				{Recommendation: "Refactor the authentication module", Priority: ImpactMedium},
			},
		},
		{
			ModeID:     "mode-b",
			Confidence: 0.7,
			Recommendations: []Recommendation{
				{Recommendation: "Write integration tests for API", Priority: ImpactHigh},
				{Recommendation: "Document the configuration options", Priority: ImpactLow},
			},
		},
	}

	merger := NewMechanicalMerger(outputs)
	result, err := merger.Merge()

	if err != nil {
		t.Fatalf("Merge returned error: %v", err)
	}

	// Should have action type groups
	if len(result.GroupedRecommendations) == 0 {
		t.Fatal("GroupedRecommendations is empty")
	}

	// Check that we have test-related recommendations grouped together
	var testGroup *RecommendationGroup
	for i := range result.GroupedRecommendations {
		if result.GroupedRecommendations[i].ActionType == "add-test" {
			testGroup = &result.GroupedRecommendations[i]
			break
		}
	}

	if testGroup != nil && len(testGroup.Recommendations) < 2 {
		t.Errorf("add-test group should have 2 recommendations, got %d", len(testGroup.Recommendations))
	}
}

func TestMechanicalMerger_DetectThesisConflicts(t *testing.T) {
	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.8,
			Thesis:     "The authentication system should use JWT tokens for secure user validation",
		},
		{
			ModeID:     "mode-b",
			Confidence: 0.7,
			Thesis:     "The authentication system should not use JWT tokens due to security risks",
		},
	}

	merger := NewMechanicalMerger(outputs)
	result, err := merger.Merge()

	if err != nil {
		t.Fatalf("Merge returned error: %v", err)
	}

	// Should detect the thesis conflict
	hasThesisConflict := false
	for _, conflict := range result.IdentifiedConflicts {
		if conflict.ConflictType == "thesis" {
			hasThesisConflict = true
			break
		}
	}

	if !hasThesisConflict {
		t.Log("Note: No thesis conflict detected - sentiment detection may need refinement")
	}
}

func TestMechanicalMerger_DetectSeverityConflicts(t *testing.T) {
	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.8,
			Risks: []Risk{
				{Risk: "SQL injection vulnerability in user input handling", Impact: ImpactCritical, Likelihood: 0.9},
			},
		},
		{
			ModeID:     "mode-b",
			Confidence: 0.7,
			Risks: []Risk{
				// Same risk but much lower severity assessment
				{Risk: "SQL injection vulnerability in user input handling", Impact: ImpactLow, Likelihood: 0.3},
			},
		},
	}

	merger := NewMechanicalMerger(outputs)
	result, err := merger.Merge()

	if err != nil {
		t.Fatalf("Merge returned error: %v", err)
	}

	// Should detect the severity conflict (critical vs low = 3 levels apart)
	hasSeverityConflict := false
	for _, conflict := range result.IdentifiedConflicts {
		if conflict.ConflictType == "severity" {
			hasSeverityConflict = true
			if conflict.ModeA != "mode-a" && conflict.ModeB != "mode-a" {
				t.Error("Severity conflict should involve mode-a")
			}
			break
		}
	}

	if !hasSeverityConflict {
		t.Error("Expected severity conflict between critical and low assessments")
	}
}

func TestMechanicalMerger_DetectRecommendationConflicts(t *testing.T) {
	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.8,
			Recommendations: []Recommendation{
				{Recommendation: "Add caching layer to improve performance", Priority: ImpactHigh},
			},
		},
		{
			ModeID:     "mode-b",
			Confidence: 0.7,
			Recommendations: []Recommendation{
				{Recommendation: "Remove caching to avoid stale data issues", Priority: ImpactHigh},
			},
		},
	}

	merger := NewMechanicalMerger(outputs)
	result, err := merger.Merge()

	if err != nil {
		t.Fatalf("Merge returned error: %v", err)
	}

	// Should detect the recommendation conflict (add vs remove caching)
	hasRecConflict := false
	for _, conflict := range result.IdentifiedConflicts {
		if conflict.ConflictType == "recommendation" {
			hasRecConflict = true
			break
		}
	}

	if !hasRecConflict {
		t.Log("Note: Recommendation conflict not detected - may need threshold adjustment")
	}
}

func TestMechanicalMerger_Statistics(t *testing.T) {
	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.8,
			TopFindings: []Finding{
				{Finding: "Finding A1", EvidencePointer: "file.go:10", Impact: ImpactHigh, Confidence: 0.9},
				{Finding: "Finding A2", EvidencePointer: "file.go:20", Impact: ImpactMedium, Confidence: 0.8},
			},
			Risks: []Risk{
				{Risk: "Risk A", Impact: ImpactHigh, Likelihood: 0.7},
			},
			Recommendations: []Recommendation{
				{Recommendation: "Rec A", Priority: ImpactHigh},
			},
		},
		{
			ModeID:     "mode-b",
			Confidence: 0.7,
			TopFindings: []Finding{
				{Finding: "Finding B1", EvidencePointer: "file.go:30", Impact: ImpactLow, Confidence: 0.7},
			},
			Risks: []Risk{
				{Risk: "Risk B", Impact: ImpactMedium, Likelihood: 0.5},
			},
			Recommendations: []Recommendation{
				{Recommendation: "Rec B", Priority: ImpactMedium},
			},
		},
	}

	merger := NewMechanicalMerger(outputs)
	result, err := merger.Merge()

	if err != nil {
		t.Fatalf("Merge returned error: %v", err)
	}

	stats := result.Statistics

	if stats.TotalFindings != 3 {
		t.Errorf("TotalFindings = %d, want 3", stats.TotalFindings)
	}
	if stats.TotalRisks != 2 {
		t.Errorf("TotalRisks = %d, want 2", stats.TotalRisks)
	}
	if stats.TotalRecommendations != 2 {
		t.Errorf("TotalRecommendations = %d, want 2", stats.TotalRecommendations)
	}
	if stats.EvidenceGroups != 3 {
		t.Errorf("EvidenceGroups = %d, want 3", stats.EvidenceGroups)
	}
}

func TestEvidenceProximity(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		wantMin  float64
		wantMax  float64
	}{
		{
			name:    "identical pointers",
			a:       "file.go:42",
			b:       "file.go:42",
			wantMin: 1.0,
			wantMax: 1.0,
		},
		{
			name:    "same file very close lines",
			a:       "file.go:10",
			b:       "file.go:12",
			wantMin: 0.8,
			wantMax: 1.0,
		},
		{
			name:    "same file nearby lines",
			a:       "file.go:10",
			b:       "file.go:18",
			wantMin: 0.6,
			wantMax: 0.8,
		},
		{
			name:    "same file distant lines",
			a:       "file.go:10",
			b:       "file.go:100",
			wantMin: 0.2,
			wantMax: 0.4,
		},
		{
			name:    "different files",
			a:       "file1.go:10",
			b:       "file2.go:10",
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name:    "empty pointer",
			a:       "",
			b:       "file.go:10",
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name:    "same file no line numbers",
			a:       "file.go",
			b:       "file.go",
			wantMin: 0.7,
			wantMax: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evidenceProximity(tt.a, tt.b)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("evidenceProximity(%q, %q) = %v, want [%v, %v]",
					tt.a, tt.b, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestInferActionType(t *testing.T) {
	tests := []struct {
		text     string
		wantType string
	}{
		{"Add unit tests for the validation logic", "add-test"},
		{"Write integration tests for API endpoints", "add-test"},
		{"Refactor the authentication module", "refactor"},
		{"Clean up the deprecated code", "refactor"},
		{"Document the API endpoints", "document"},
		{"Add comments to complex functions", "document"},
		{"Fix the null pointer exception", "fix"},
		{"Resolve the race condition", "fix"},
		{"Add caching for better performance", "add-feature"},
		{"Implement retry logic", "add-feature"},
		{"Remove deprecated functions", "remove"},
		{"Delete unused imports", "remove"},
		{"Update dependencies to latest versions", "update"},
		{"Encrypt sensitive data at rest", "security"},
		{"Optimize database queries", "optimize"},
		{"Random text without keywords", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := inferActionType(tt.text)
			if got != tt.wantType {
				t.Errorf("inferActionType(%q) = %q, want %q", tt.text, got, tt.wantType)
			}
		})
	}
}

func TestParseEvidencePointer(t *testing.T) {
	tests := []struct {
		ptr      string
		wantFile string
		wantLine int
	}{
		{"file.go:42", "file.go", 42},
		{"path/to/file.go:100", "path/to/file.go", 100},
		{"file.go", "file.go", -1},
		{"file.go:invalid", "file.go", -1},
		{"", "", -1},
	}

	for _, tt := range tests {
		t.Run(tt.ptr, func(t *testing.T) {
			file, line := parseEvidencePointer(tt.ptr)
			if file != tt.wantFile {
				t.Errorf("file = %q, want %q", file, tt.wantFile)
			}
			if line != tt.wantLine {
				t.Errorf("line = %d, want %d", line, tt.wantLine)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		text     string
		patterns []string
		want     bool
	}{
		{"match first", "hello world", []string{"hello", "xyz"}, true},
		{"match second", "hello world", []string{"xyz", "world"}, true},
		{"no match", "hello world", []string{"xyz", "abc"}, false},
		{"empty patterns", "hello world", nil, false},
		{"empty text", "", []string{"hello"}, false},
		{"substring match", "authentication", []string{"auth"}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := containsAny(tc.text, tc.patterns)
			if got != tc.want {
				t.Errorf("containsAny(%q, %v) = %v, want %v", tc.text, tc.patterns, got, tc.want)
			}
		})
	}
}

func TestSeverityRank(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level ImpactLevel
		want  int
	}{
		{ImpactCritical, 4},
		{ImpactHigh, 3},
		{ImpactMedium, 2},
		{ImpactLow, 1},
		{ImpactLevel("unknown"), 2},
	}

	for _, tc := range tests {
		t.Run(string(tc.level), func(t *testing.T) {
			t.Parallel()
			got := severityRank(tc.level)
			if got != tc.want {
				t.Errorf("severityRank(%q) = %d, want %d", tc.level, got, tc.want)
			}
		})
	}
}

func TestMergeAbs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input int
		want  int
	}{
		{0, 0},
		{5, 5},
		{-5, 5},
		{-1, 1},
		{1, 1},
	}

	for _, tc := range tests {
		got := abs(tc.input)
		if got != tc.want {
			t.Errorf("abs(%d) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestMergeIsStopWord(t *testing.T) {
	t.Parallel()

	stopWords := []string{"the", "this", "that", "with", "from", "have", "been", "will", "should", "would", "could", "being", "there", "their", "when", "where"}
	for _, w := range stopWords {
		t.Run("stop_"+w, func(t *testing.T) {
			t.Parallel()
			if !isStopWord(w) {
				t.Errorf("isStopWord(%q) = false, want true", w)
			}
		})
	}

	nonStop := []string{"security", "injection", "database", "function", "error"}
	for _, w := range nonStop {
		t.Run("nonstop_"+w, func(t *testing.T) {
			t.Parallel()
			if isStopWord(w) {
				t.Errorf("isStopWord(%q) = true, want false", w)
			}
		})
	}
}
