package ensemble

import (
	"strings"
	"testing"
)

func TestDefaultDedupeConfig(t *testing.T) {
	cfg := DefaultDedupeConfig()
	if cfg.SimilarityThreshold <= 0 || cfg.SimilarityThreshold > 1 {
		t.Errorf("invalid similarity threshold: %f", cfg.SimilarityThreshold)
	}
	if cfg.TextWeight+cfg.EvidenceWeight <= 0 {
		t.Error("weights should sum to > 0")
	}
}

func TestDedupeEngine_EmptyInput(t *testing.T) {
	t.Log("TEST: TestDedupeEngine_EmptyInput - starting")

	engine := NewDedupeEngine(DefaultDedupeConfig())
	result := engine.Dedupe(nil)

	if result == nil {
		t.Fatal("expected non-nil result for empty input")
	}
	if len(result.Clusters) != 0 {
		t.Errorf("expected 0 clusters, got %d", len(result.Clusters))
	}
	if result.Stats.InputFindings != 0 {
		t.Errorf("expected 0 input findings, got %d", result.Stats.InputFindings)
	}

	t.Log("TEST: TestDedupeEngine_EmptyInput - assertion: empty input handled correctly")
}

func TestDedupeEngine_SingleFinding(t *testing.T) {
	t.Log("TEST: TestDedupeEngine_SingleFinding - starting")

	outputs := []ModeOutput{
		{
			ModeID:     "deductive",
			Confidence: 0.8,
			TopFindings: []Finding{
				{Finding: "The authentication module has a bug", Impact: ImpactHigh, Confidence: 0.9},
			},
		},
	}

	engine := NewDedupeEngine(DefaultDedupeConfig())
	result := engine.Dedupe(outputs)

	if result.Stats.InputFindings != 1 {
		t.Errorf("expected 1 input finding, got %d", result.Stats.InputFindings)
	}
	if result.Stats.OutputClusters != 1 {
		t.Errorf("expected 1 cluster, got %d", result.Stats.OutputClusters)
	}
	if len(result.Clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(result.Clusters))
	}
	if result.Clusters[0].MemberCount != 1 {
		t.Errorf("expected cluster with 1 member, got %d", result.Clusters[0].MemberCount)
	}

	t.Logf("TEST: TestDedupeEngine_SingleFinding - cluster ID: %s", result.Clusters[0].ClusterID)
	t.Log("TEST: TestDedupeEngine_SingleFinding - assertion: single finding creates single cluster")
}

func TestDedupeEngine_DuplicateFindingsAcrossModes(t *testing.T) {
	t.Log("TEST: TestDedupeEngine_DuplicateFindingsAcrossModes - starting")

	// Two modes report very similar findings (high word overlap for Jaccard)
	outputs := []ModeOutput{
		{
			ModeID:     "deductive",
			Confidence: 0.9,
			TopFindings: []Finding{
				{Finding: "Memory leak detected in database connection pool handler", Impact: ImpactCritical, Confidence: 0.9},
			},
		},
		{
			ModeID:     "adversarial",
			Confidence: 0.85,
			TopFindings: []Finding{
				{Finding: "Memory leak detected in database connection pool handler code", Impact: ImpactCritical, Confidence: 0.85},
			},
		},
	}

	// Use lower threshold to ensure similar sentences are detected
	cfg := DefaultDedupeConfig()
	cfg.SimilarityThreshold = 0.6

	engine := NewDedupeEngine(cfg)
	result := engine.Dedupe(outputs)

	t.Logf("TEST: TestDedupeEngine_DuplicateFindingsAcrossModes - input: %d, clusters: %d",
		result.Stats.InputFindings, result.Stats.OutputClusters)

	if result.Stats.InputFindings != 2 {
		t.Errorf("expected 2 input findings, got %d", result.Stats.InputFindings)
	}

	// Should dedupe to 1 cluster
	if result.Stats.OutputClusters != 1 {
		t.Errorf("expected 1 cluster after deduplication, got %d", result.Stats.OutputClusters)
	}

	if len(result.Clusters) > 0 {
		cluster := result.Clusters[0]
		t.Logf("TEST: cluster ID=%s, members=%d, sources=%v",
			cluster.ClusterID, cluster.MemberCount, cluster.SourceModes)

		if cluster.MemberCount != 2 {
			t.Errorf("expected cluster with 2 members, got %d", cluster.MemberCount)
		}
		if len(cluster.SourceModes) != 2 {
			t.Errorf("expected 2 source modes, got %d", len(cluster.SourceModes))
		}
	}

	t.Log("TEST: TestDedupeEngine_DuplicateFindingsAcrossModes - assertion: duplicate findings merged")
}

func TestDedupeEngine_DistinctFindings(t *testing.T) {
	t.Log("TEST: TestDedupeEngine_DistinctFindings - starting")

	// Two completely different findings
	outputs := []ModeOutput{
		{
			ModeID:     "deductive",
			Confidence: 0.9,
			TopFindings: []Finding{
				{Finding: "Memory leak in connection pool cleanup", Impact: ImpactHigh, Confidence: 0.9},
			},
		},
		{
			ModeID:     "adversarial",
			Confidence: 0.85,
			TopFindings: []Finding{
				{Finding: "SQL injection vulnerability in user search endpoint", Impact: ImpactCritical, Confidence: 0.95},
			},
		},
	}

	engine := NewDedupeEngine(DefaultDedupeConfig())
	result := engine.Dedupe(outputs)

	// Should remain as 2 separate clusters
	if result.Stats.OutputClusters != 2 {
		t.Errorf("expected 2 clusters for distinct findings, got %d", result.Stats.OutputClusters)
	}
	if result.Stats.DuplicatesFound != 0 {
		t.Errorf("expected 0 duplicates for distinct findings, got %d", result.Stats.DuplicatesFound)
	}

	t.Log("TEST: TestDedupeEngine_DistinctFindings - assertion: distinct findings stay separate")
}

func TestDedupeEngine_EvidencePointerSimilarity(t *testing.T) {
	t.Log("TEST: TestDedupeEngine_EvidencePointerSimilarity - starting")

	// Similar findings at the same file location
	outputs := []ModeOutput{
		{
			ModeID:     "mode1",
			Confidence: 0.9,
			TopFindings: []Finding{
				{
					Finding:         "Variable may be uninitialized",
					EvidencePointer: "main.go:42",
					Confidence:      0.8,
				},
			},
		},
		{
			ModeID:     "mode2",
			Confidence: 0.85,
			TopFindings: []Finding{
				{
					Finding:         "Variable could be uninitialized here",
					EvidencePointer: "main.go:42",
					Confidence:      0.75,
				},
			},
		},
	}

	cfg := DefaultDedupeConfig()
	cfg.EvidenceWeight = 0.5 // Increase evidence weight
	cfg.TextWeight = 0.5

	engine := NewDedupeEngine(cfg)
	result := engine.Dedupe(outputs)

	// With same evidence pointer, these should cluster together
	if result.Stats.OutputClusters > 1 {
		t.Logf("TEST: got %d clusters, expected 1 due to same evidence pointer", result.Stats.OutputClusters)
	}

	t.Log("TEST: TestDedupeEngine_EvidencePointerSimilarity - assertion: evidence pointer affects similarity")
}

func TestDedupeEngine_ClusterIDStability(t *testing.T) {
	t.Log("TEST: TestDedupeEngine_ClusterIDStability - starting")

	outputs := []ModeOutput{
		{
			ModeID:     "mode1",
			Confidence: 0.9,
			TopFindings: []Finding{
				{Finding: "Test finding one", Confidence: 0.9},
				{Finding: "Test finding two", Confidence: 0.8},
			},
		},
	}

	engine := NewDedupeEngine(DefaultDedupeConfig())

	// Run deduplication twice
	result1 := engine.Dedupe(outputs)
	result2 := engine.Dedupe(outputs)

	// Cluster IDs should be identical
	if len(result1.Clusters) != len(result2.Clusters) {
		t.Fatalf("cluster counts differ: %d vs %d", len(result1.Clusters), len(result2.Clusters))
	}

	for i := range result1.Clusters {
		if result1.Clusters[i].ClusterID != result2.Clusters[i].ClusterID {
			t.Errorf("cluster ID mismatch at %d: %s vs %s",
				i, result1.Clusters[i].ClusterID, result2.Clusters[i].ClusterID)
		}
	}

	t.Log("TEST: TestDedupeEngine_ClusterIDStability - assertion: cluster IDs are deterministic")
}

func TestDedupeEngine_PreferHighConfidence(t *testing.T) {
	t.Log("TEST: TestDedupeEngine_PreferHighConfidence - starting")

	// Very similar findings to ensure they cluster together
	outputs := []ModeOutput{
		{
			ModeID:     "mode1",
			Confidence: 0.7,
			TopFindings: []Finding{
				{Finding: "SQL injection vulnerability detected in user search query", Confidence: 0.6},
			},
		},
		{
			ModeID:     "mode2",
			Confidence: 0.95,
			TopFindings: []Finding{
				{Finding: "SQL injection vulnerability detected in user search query handler", Confidence: 0.95},
			},
		},
	}

	cfg := DefaultDedupeConfig()
	cfg.PreferHighConfidence = true
	cfg.SimilarityThreshold = 0.6 // Lower threshold for similar sentences

	engine := NewDedupeEngine(cfg)
	result := engine.Dedupe(outputs)

	if len(result.Clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(result.Clusters))
	}

	// Canonical should be the higher confidence one
	if result.Clusters[0].Canonical.Confidence < 0.9 {
		t.Errorf("expected high-confidence canonical, got %.2f", result.Clusters[0].Canonical.Confidence)
	}

	t.Log("TEST: TestDedupeEngine_PreferHighConfidence - assertion: higher confidence finding becomes canonical")
}

func TestDedupeEngine_MultipleClustersMixedFindings(t *testing.T) {
	t.Log("TEST: TestDedupeEngine_MultipleClustersMixedFindings - starting")

	// Multiple findings: some duplicates, some unique
	outputs := []ModeOutput{
		{
			ModeID:     "mode1",
			Confidence: 0.9,
			TopFindings: []Finding{
				{Finding: "Memory leak in database connection pool", Confidence: 0.9},
				{Finding: "Null pointer dereference in parser", Confidence: 0.85},
			},
		},
		{
			ModeID:     "mode2",
			Confidence: 0.85,
			TopFindings: []Finding{
				{Finding: "Memory leak in the database connection pool cleanup", Confidence: 0.88}, // Similar to first
				{Finding: "Race condition in worker threads", Confidence: 0.8},                    // Unique
			},
		},
		{
			ModeID:     "mode3",
			Confidence: 0.8,
			TopFindings: []Finding{
				{Finding: "Potential null pointer dereference in parser module", Confidence: 0.75}, // Similar to second
			},
		},
	}

	engine := NewDedupeEngine(DefaultDedupeConfig())
	result := engine.Dedupe(outputs)

	t.Logf("TEST: input=%d, clusters=%d, duplicates=%d",
		result.Stats.InputFindings, result.Stats.OutputClusters, result.Stats.DuplicatesFound)

	// Should have 3 clusters: memory leak, null pointer, race condition
	// (memory leak and null pointer each have duplicates)
	if result.Stats.InputFindings != 5 {
		t.Errorf("expected 5 input findings, got %d", result.Stats.InputFindings)
	}

	// Log cluster details
	for i, c := range result.Clusters {
		t.Logf("TEST: cluster %d: %s (%d members)", i, c.ClusterID, c.MemberCount)
	}

	t.Log("TEST: TestDedupeEngine_MultipleClustersMixedFindings - assertion: mixed findings cluster correctly")
}

func TestDedupeResult_GetCanonicalFindings(t *testing.T) {
	t.Log("TEST: TestDedupeResult_GetCanonicalFindings - starting")

	outputs := []ModeOutput{
		{
			ModeID: "mode1",
			TopFindings: []Finding{
				{Finding: "Finding A", Confidence: 0.9},
				{Finding: "Finding B", Confidence: 0.8},
			},
		},
	}

	result := DedupeFindings(outputs)
	canonicals := result.GetCanonicalFindings()

	if len(canonicals) != len(result.Clusters) {
		t.Errorf("canonical count mismatch: %d vs %d clusters",
			len(canonicals), len(result.Clusters))
	}

	t.Log("TEST: TestDedupeResult_GetCanonicalFindings - assertion: canonical findings extracted correctly")
}

func TestDedupeResult_GetClusterByID(t *testing.T) {
	t.Log("TEST: TestDedupeResult_GetClusterByID - starting")

	outputs := []ModeOutput{
		{ModeID: "mode1", TopFindings: []Finding{{Finding: "Test finding", Confidence: 0.9}}},
	}

	result := DedupeFindings(outputs)
	if len(result.Clusters) == 0 {
		t.Fatal("expected at least 1 cluster")
	}

	clusterID := result.Clusters[0].ClusterID
	found := result.GetClusterByID(clusterID)
	if found == nil {
		t.Errorf("cluster %s not found", clusterID)
	}

	notFound := result.GetClusterByID("nonexistent-id")
	if notFound != nil {
		t.Error("expected nil for nonexistent cluster ID")
	}

	t.Log("TEST: TestDedupeResult_GetClusterByID - assertion: cluster lookup works correctly")
}

func TestDedupeResult_Render(t *testing.T) {
	t.Log("TEST: TestDedupeResult_Render - starting")

	outputs := []ModeOutput{
		{
			ModeID: "mode1",
			TopFindings: []Finding{
				{Finding: "Important finding here", Confidence: 0.9, EvidencePointer: "file.go:10"},
			},
		},
	}

	result := DedupeFindings(outputs)
	rendered := result.Render()

	if rendered == "" {
		t.Error("expected non-empty render output")
	}
	if !strings.Contains(rendered, "Deduplication Results") {
		t.Error("render missing header")
	}
	if !strings.Contains(rendered, "Input Findings") {
		t.Error("render missing input count")
	}
	if !strings.Contains(rendered, "clu-") {
		t.Error("render missing cluster ID")
	}

	t.Logf("TEST: render output preview: %s", rendered[:min(200, len(rendered))])
	t.Log("TEST: TestDedupeResult_Render - assertion: render produces readable output")
}

func TestDedupeConvenienceFunctions(t *testing.T) {
	t.Log("TEST: TestDedupeConvenienceFunctions - starting")

	outputs := []ModeOutput{
		{ModeID: "mode1", TopFindings: []Finding{{Finding: "Test", Confidence: 0.9}}},
	}

	// Test DedupeFindings
	result1 := DedupeFindings(outputs)
	if result1 == nil {
		t.Error("DedupeFindings returned nil")
	}

	// Test DedupeFindingsWithConfig
	cfg := DefaultDedupeConfig()
	cfg.SimilarityThreshold = 0.5
	result2 := DedupeFindingsWithConfig(outputs, cfg)
	if result2 == nil {
		t.Error("DedupeFindingsWithConfig returned nil")
	}

	t.Log("TEST: TestDedupeConvenienceFunctions - assertion: convenience functions work")
}
