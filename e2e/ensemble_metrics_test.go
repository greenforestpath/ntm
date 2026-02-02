//go:build e2e
// +build e2e

// Package e2e contains end-to-end tests for NTM observability metrics.
// These tests use deterministic fixtures (no model calls).
package e2e

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/Dicklesworthstone/ntm/internal/ensemble"
)

func TestE2E_Metrics_CoverageMap(t *testing.T) {
	SkipIfShort(t)

	catalog := testMetricsCatalog(t)
	coverage := ensemble.NewCoverageMap(catalog)
	coverage.RecordMode("formal")
	coverage.RecordMode("ampliative")

	report := coverage.CalculateCoverage()
	t.Logf("modes used=%v", []string{"formal", "ampliative"})
	t.Logf("coverage overall=%.3f blind_spots=%v", report.Overall, report.BlindSpots)

	expectedOverall := 2.0 / float64(len(ensemble.AllCategories()))
	if math.Abs(report.Overall-expectedOverall) > 0.0001 {
		t.Fatalf("overall coverage %.3f != %.3f", report.Overall, expectedOverall)
	}
	if !containsCategory(report.BlindSpots, ensemble.CategoryUncertainty) {
		t.Fatalf("expected Uncertainty to be a blind spot, got %v", report.BlindSpots)
	}
}

func TestE2E_Metrics_RedundancyScore_LowRedundancy(t *testing.T) {
	SkipIfShort(t)

	outputs := []ensemble.ModeOutput{
		modeOutput("formal", "Thesis A", []ensemble.Finding{finding("Finding A", "file_a.go:10")}),
		modeOutput("ampliative", "Thesis B", []ensemble.Finding{finding("Finding B", "file_b.go:12")}),
	}
	t.Logf("low redundancy inputs: formal=%d findings, ampliative=%d findings", len(outputs[0].TopFindings), len(outputs[1].TopFindings))

	analysis := ensemble.CalculateRedundancy(outputs)
	t.Logf("low redundancy score=%.3f", analysis.OverallScore)

	if analysis.OverallScore >= 0.3 {
		t.Fatalf("expected low redundancy (<0.3), got %.3f", analysis.OverallScore)
	}
}

func TestE2E_Metrics_RedundancyScore_HighRedundancy(t *testing.T) {
	SkipIfShort(t)

	shared := finding("Shared finding", "file_shared.go:42")
	outputs := []ensemble.ModeOutput{
		modeOutput("formal", "Thesis A", []ensemble.Finding{shared, finding("Extra", "file_extra.go:1")}),
		modeOutput("ampliative", "Thesis B", []ensemble.Finding{shared}),
	}
	t.Logf("high redundancy inputs: formal=%d findings, ampliative=%d findings", len(outputs[0].TopFindings), len(outputs[1].TopFindings))

	analysis := ensemble.CalculateRedundancy(outputs)
	highPairs := analysis.GetHighRedundancyPairs(0.6)
	analysis.Recommendations = analysis.SuggestReplacements(testMetricsCatalog(t))

	t.Logf("high redundancy score=%.3f pairs=%v", analysis.OverallScore, highPairs)

	if analysis.OverallScore < 0.6 {
		t.Fatalf("expected high redundancy (>=0.6), got %.3f", analysis.OverallScore)
	}
	if len(highPairs) == 0 {
		t.Fatalf("expected high redundancy pairs, got none")
	}
}

func TestE2E_Metrics_FindingsVelocity(t *testing.T) {
	SkipIfShort(t)

	tracker := ensemble.NewVelocityTracker()
	tracker.RecordOutput("formal", modeOutput("formal", "Thesis A", []ensemble.Finding{finding("A1", "file.go:1"), finding("A2", "file.go:2")}), 1000)
	tracker.RecordOutput("ampliative", modeOutput("ampliative", "Thesis B", []ensemble.Finding{finding("B1", "file.go:3")}), 500)

	report := tracker.CalculateVelocity()
	t.Logf("velocity inputs: formal tokens=1000 findings=2, ampliative tokens=500 findings=1")
	t.Logf("velocity overall=%.3f per_mode=%v low=%v high=%v", report.Overall, report.PerMode, report.LowPerformers, report.HighPerformers)

	if report.Overall <= 0 {
		t.Fatalf("expected positive overall velocity, got %.3f", report.Overall)
	}
	if len(report.PerMode) != 2 {
		t.Fatalf("expected 2 per-mode entries, got %d", len(report.PerMode))
	}
	if report.PerMode[0].Velocity == 0 {
		t.Fatalf("expected non-zero velocity for mode %s", report.PerMode[0].ModeID)
	}
}

func TestE2E_Metrics_ConflictDensity(t *testing.T) {
	SkipIfShort(t)

	t.Run("AuditReport", func(t *testing.T) {
		report := &ensemble.AuditReport{
			Conflicts: []ensemble.DetailedConflict{
				{
					Topic: "Dependency choice",
					Positions: []ensemble.ConflictPosition{
						{ModeID: "formal", Position: "Use interface"},
						{ModeID: "ampliative", Position: "Use concrete"},
					},
					Severity: ensemble.ConflictHigh,
				},
				{
					Topic: "Dependency choice",
					Positions: []ensemble.ConflictPosition{
						{ModeID: "formal", Position: "Use interface"},
						{ModeID: "ampliative", Position: "Use concrete"},
					},
					Severity: ensemble.ConflictHigh,
				},
			},
		}

		tracker := ensemble.NewConflictTracker()
		tracker.FromAudit(report)
		density := tracker.GetDensity(1)

		t.Logf("audit conflicts=%d high_pairs=%v source=%s", density.TotalConflicts, density.HighConflictPairs, density.Source)

		if density.TotalConflicts != 2 {
			t.Fatalf("expected 2 conflicts, got %d", density.TotalConflicts)
		}
		if len(density.HighConflictPairs) != 1 {
			t.Fatalf("expected one high conflict pair, got %v", density.HighConflictPairs)
		}
	})

	t.Run("Fallback", func(t *testing.T) {
		outputs := []ensemble.ModeOutput{
			modeOutput("formal", "Use interface", []ensemble.Finding{finding("A1", "file.go:1")}),
			modeOutput("ampliative", "Use concrete", []ensemble.Finding{finding("B1", "file.go:2")}),
		}

		tracker := ensemble.NewConflictTracker()
		tracker.DetectConflicts(outputs)
		density := tracker.GetDensity(1)

		t.Logf("fallback conflicts=%d source=%s", density.TotalConflicts, density.Source)

		if density.Source != "fallback" {
			t.Fatalf("expected fallback source, got %s", density.Source)
		}
		if density.TotalConflicts == 0 {
			t.Fatal("expected at least one conflict from fallback")
		}
	})
}

func TestE2E_Metrics_PostRunReport(t *testing.T) {
	SkipIfShort(t)

	catalog := testMetricsCatalog(t)
	coverage := ensemble.NewCoverageMap(catalog)
	coverage.RecordMode("formal")
	coverage.RecordMode("ampliative")
	coverageReport := coverage.CalculateCoverage()

	outputs := []ensemble.ModeOutput{
		modeOutput("formal", "Thesis A", []ensemble.Finding{finding("Finding A", "file_a.go:10")}),
		modeOutput("ampliative", "Thesis B", []ensemble.Finding{finding("Finding B", "file_b.go:12")}),
	}
	redun := ensemble.CalculateRedundancy(outputs)

	tracker := ensemble.NewVelocityTracker()
	tracker.RecordOutput("formal", outputs[0], 900)
	tracker.RecordOutput("ampliative", outputs[1], 700)
	velocity := tracker.CalculateVelocity()

	conflictTracker := ensemble.NewConflictTracker()
	conflictTracker.DetectConflicts(outputs)
	conflicts := conflictTracker.GetDensity(1)

	report := buildMetricsReport(coverageReport, redun, velocity, conflicts)
	t.Logf("post-run report preview: %s", truncateString(report, 200))

	for _, marker := range []string{"Category Coverage", "Redundancy Analysis", "Findings Velocity", "Conflicts"} {
		if !strings.Contains(report, marker) {
			t.Fatalf("expected report to include %q, got:\n%s", marker, report)
		}
	}
}

func buildMetricsReport(
	coverage *ensemble.CoverageReport,
	redundancy *ensemble.RedundancyAnalysis,
	velocity *ensemble.VelocityReport,
	conflicts *ensemble.ConflictDensity,
) string {
	var b strings.Builder
	if coverage != nil {
		covered := 0
		for _, category := range ensemble.AllCategories() {
			if cov, ok := coverage.PerCategory[category]; ok && len(cov.UsedModes) > 0 {
				covered++
			}
		}
		fmt.Fprintf(&b, "Category Coverage: %.2f (%d/%d categories)\n", coverage.Overall, covered, len(ensemble.AllCategories()))
		if len(coverage.Suggestions) > 0 {
			fmt.Fprintf(&b, "Coverage Suggestions: %s\n", strings.Join(coverage.Suggestions, "; "))
		}
	}
	if redundancy != nil {
		b.WriteString(redundancy.Render())
		b.WriteString("\n")
	}
	if velocity != nil {
		fmt.Fprintf(&b, "Findings Velocity: %.2f\n", velocity.Overall)
		if len(velocity.Suggestions) > 0 {
			fmt.Fprintf(&b, "Velocity Suggestions: %s\n", strings.Join(velocity.Suggestions, "; "))
		}
	}
	if conflicts != nil {
		fmt.Fprintf(&b, "Conflicts: %d detected\n", conflicts.TotalConflicts)
	}
	return b.String()
}

func testMetricsCatalog(t *testing.T) *ensemble.ModeCatalog {
	modes := []ensemble.ReasoningMode{
		{
			ID:        "formal",
			Code:      "A1",
			Name:      "Formal Mode",
			Category:  ensemble.CategoryFormal,
			Tier:      ensemble.TierCore,
			ShortDesc: "formal test",
		},
		{
			ID:        "ampliative",
			Code:      "B1",
			Name:      "Ampliative Mode",
			Category:  ensemble.CategoryAmpliative,
			Tier:      ensemble.TierCore,
			ShortDesc: "ampliative test",
		},
		{
			ID:        "uncertainty",
			Code:      "C1",
			Name:      "Uncertainty Mode",
			Category:  ensemble.CategoryUncertainty,
			Tier:      ensemble.TierCore,
			ShortDesc: "uncertainty test",
		},
	}

	catalog, err := ensemble.NewModeCatalog(modes, "test")
	if err != nil {
		if t != nil {
			t.Fatalf("NewModeCatalog: %v", err)
		}
		return nil
	}
	return catalog
}

func modeOutput(modeID, thesis string, findings []ensemble.Finding) ensemble.ModeOutput {
	return ensemble.ModeOutput{
		ModeID:      modeID,
		Thesis:      thesis,
		TopFindings: findings,
	}
}

func finding(text, evidence string) ensemble.Finding {
	return ensemble.Finding{
		Finding:         text,
		Impact:          ensemble.ImpactLow,
		Confidence:      0.5,
		EvidencePointer: evidence,
	}
}

func containsCategory(categories []ensemble.ModeCategory, target ensemble.ModeCategory) bool {
	for _, category := range categories {
		if category == target {
			return true
		}
	}
	return false
}
