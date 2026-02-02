package ensemble

import "testing"

func TestGenerateFindingID_Deterministic(t *testing.T) {
	first := GenerateFindingID("mode-a", "Same finding")
	second := GenerateFindingID("mode-a", "Same finding")
	if first != second {
		t.Fatalf("GenerateFindingID should be deterministic, got %q and %q", first, second)
	}

	other := GenerateFindingID("mode-b", "Same finding")
	if other == first {
		t.Fatalf("GenerateFindingID should differ across modes, got %q for both", other)
	}
}

func TestMergeOutputsWithProvenance_RecordsMerge(t *testing.T) {
	tracker := NewProvenanceTracker("question", []string{"mode-a", "mode-b"})
	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Thesis:     "A",
			Confidence: 0.9,
			TopFindings: []Finding{
				{Finding: "Shared finding", Impact: ImpactHigh, Confidence: 0.9},
			},
		},
		{
			ModeID:     "mode-b",
			Thesis:     "B",
			Confidence: 0.8,
			TopFindings: []Finding{
				{Finding: "Shared finding", Impact: ImpactHigh, Confidence: 0.8},
			},
		},
	}

	merged := MergeOutputsWithProvenance(outputs, DefaultMergeConfig(), tracker)
	if len(merged.Findings) != 1 {
		t.Fatalf("expected 1 merged finding, got %d", len(merged.Findings))
	}

	primaryID := GenerateFindingID("mode-a", "Shared finding")
	secondaryID := GenerateFindingID("mode-b", "Shared finding")

	primary, ok := tracker.GetChain(primaryID)
	if !ok {
		t.Fatalf("expected primary chain %s", primaryID)
	}
	if !containsStringProvenance(primary.MergedFrom, secondaryID) {
		t.Fatalf("expected primary chain to merge %s, got %+v", secondaryID, primary.MergedFrom)
	}

	mergedChain, ok := tracker.GetChain(secondaryID)
	if !ok {
		t.Fatalf("expected merged chain %s", secondaryID)
	}
	if mergedChain.MergedInto != primaryID {
		t.Fatalf("expected merged chain to point to %s, got %s", primaryID, mergedChain.MergedInto)
	}

	if merged.Findings[0].ProvenanceID != primaryID {
		t.Fatalf("expected merged provenance id %s, got %s", primaryID, merged.Findings[0].ProvenanceID)
	}
}

func TestSynthesizer_RecordsSynthesisCitations(t *testing.T) {
	tracker := NewProvenanceTracker("question", []string{"mode-a"})
	synth, err := NewSynthesizer(SynthesisConfig{Strategy: StrategyManual})
	if err != nil {
		t.Fatalf("NewSynthesizer error: %v", err)
	}

	outputs := []ModeOutput{
		{
			ModeID:     "mode-a",
			Thesis:     "A",
			Confidence: 0.9,
			TopFindings: []Finding{
				{Finding: "Cited finding", Impact: ImpactHigh, Confidence: 0.9},
			},
		},
	}

	_, err = synth.Synthesize(&SynthesisInput{
		Outputs:          outputs,
		OriginalQuestion: "question",
		Provenance:       tracker,
	})
	if err != nil {
		t.Fatalf("Synthesize error: %v", err)
	}

	findingID := GenerateFindingID("mode-a", "Cited finding")
	chain, ok := tracker.GetChain(findingID)
	if !ok {
		t.Fatalf("expected chain %s", findingID)
	}
	if len(chain.SynthesisCitations) == 0 {
		t.Fatalf("expected synthesis citations to be recorded")
	}
}

func containsStringProvenance(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func TestProvenanceChain_IsActive(t *testing.T) {
	t.Parallel()

	active := &ProvenanceChain{FindingID: "f1", MergedInto: ""}
	if !active.IsActive() {
		t.Error("chain with empty MergedInto should be active")
	}

	merged := &ProvenanceChain{FindingID: "f2", MergedInto: "f1"}
	if merged.IsActive() {
		t.Error("chain with non-empty MergedInto should not be active")
	}
}

func TestProvenanceTracker_ContextHash(t *testing.T) {
	t.Parallel()

	tracker := NewProvenanceTracker("test question", []string{"mode-a", "mode-b"})
	hash := tracker.ContextHash()
	if hash == "" {
		t.Error("ContextHash() should not be empty")
	}
	if len(hash) != 16 {
		t.Errorf("ContextHash() length = %d, want 16", len(hash))
	}

	// Same input should produce same hash
	tracker2 := NewProvenanceTracker("test question", []string{"mode-a", "mode-b"})
	if tracker.ContextHash() != tracker2.ContextHash() {
		t.Error("same inputs should produce same context hash")
	}

	// Different input should produce different hash
	tracker3 := NewProvenanceTracker("different question", []string{"mode-a"})
	if tracker.ContextHash() == tracker3.ContextHash() {
		t.Error("different inputs should produce different context hash")
	}
}

func TestProvenanceTracker_CountAndActiveCount(t *testing.T) {
	t.Parallel()

	tracker := NewProvenanceTracker("q", []string{"m1", "m2"})

	// Empty tracker
	if tracker.Count() != 0 {
		t.Errorf("Count() = %d, want 0", tracker.Count())
	}
	if tracker.ActiveCount() != 0 {
		t.Errorf("ActiveCount() = %d, want 0", tracker.ActiveCount())
	}

	// Add some chains via RecordDiscovery
	tracker.RecordDiscovery("m1", Finding{Finding: "Finding A", Impact: ImpactHigh, Confidence: 0.9})
	tracker.RecordDiscovery("m2", Finding{Finding: "Finding B", Impact: ImpactMedium, Confidence: 0.8})
	if tracker.Count() != 2 {
		t.Errorf("Count() = %d, want 2", tracker.Count())
	}
	if tracker.ActiveCount() != 2 {
		t.Errorf("ActiveCount() = %d, want 2", tracker.ActiveCount())
	}
}

func TestProvenanceTracker_Stats(t *testing.T) {
	t.Parallel()

	tracker := NewProvenanceTracker("q", []string{"m1", "m2"})

	tracker.RecordDiscovery("m1", Finding{Finding: "Finding A", Impact: ImpactHigh, Confidence: 0.9})
	tracker.RecordDiscovery("m2", Finding{Finding: "Finding B", Impact: ImpactMedium, Confidence: 0.8})

	stats := tracker.Stats()
	if stats.TotalFindings != 2 {
		t.Errorf("TotalFindings = %d, want 2", stats.TotalFindings)
	}
	if stats.ActiveFindings != 2 {
		t.Errorf("ActiveFindings = %d, want 2", stats.ActiveFindings)
	}
	if stats.MergedFindings != 0 {
		t.Errorf("MergedFindings = %d, want 0", stats.MergedFindings)
	}
	if stats.ModeBreakdown["m1"] != 1 {
		t.Errorf("ModeBreakdown[m1] = %d, want 1", stats.ModeBreakdown["m1"])
	}
	if stats.ModeBreakdown["m2"] != 1 {
		t.Errorf("ModeBreakdown[m2] = %d, want 1", stats.ModeBreakdown["m2"])
	}
}

func TestProvenanceTracker_Export(t *testing.T) {
	t.Parallel()

	tracker := NewProvenanceTracker("q", []string{"m1"})
	tracker.RecordDiscovery("m1", Finding{Finding: "Finding", Impact: ImpactHigh, Confidence: 0.9})

	data, err := tracker.Export()
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}
	if len(data) == 0 {
		t.Error("Export() should not return empty data")
	}
	// Verify it's valid JSON containing expected keys
	json := string(data)
	if !containsStringProvenance([]string{json}, "context_hash") {
		// Use a simpler check
		found := false
		for i := 0; i <= len(json)-12; i++ {
			if json[i:i+12] == "context_hash" {
				found = true
				break
			}
		}
		if !found {
			t.Error("exported JSON should contain context_hash")
		}
	}
}
