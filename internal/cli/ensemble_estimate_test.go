package cli

import (
	"strings"
	"testing"

	"github.com/Dicklesworthstone/ntm/internal/ensemble"
)

func TestBuildEnsembleEstimate_FormalCore(t *testing.T) {
	catalog := testEstimateCatalog(t)
	budget := ensemble.BudgetConfig{
		MaxTokensPerMode: 6000,
		MaxTotalTokens:   8000,
	}
	input := ensemble.EstimateInput{
		ModeIDs:       []string{"formal-mode"},
		Question:      "Test question",
		Budget:        budget,
		AllowAdvanced: true,
	}

	out, err := buildEnsembleEstimate(catalog, input, ensemble.EstimateOptions{DisableContext: true}, 0)
	if err != nil {
		t.Fatalf("buildEnsembleEstimate error: %v", err)
	}
	if len(out.Modes) != 1 {
		t.Fatalf("expected 1 mode, got %d", len(out.Modes))
	}
	if out.Modes[0].ModeID != "formal-mode" {
		t.Fatalf("expected mode formal-mode, got %s", out.Modes[0].ModeID)
	}
	if out.Modes[0].TokenEstimate < 3000 {
		t.Fatalf("formal core estimate = %d, want >= 3000", out.Modes[0].TokenEstimate)
	}
}

func TestBuildEnsembleEstimate_Warnings(t *testing.T) {
	catalog := testEstimateCatalog(t)
	budget := ensemble.BudgetConfig{
		MaxTokensPerMode: 4000,
		MaxTotalTokens:   4000,
	}

	input := ensemble.EstimateInput{
		ModeIDs:       []string{"formal-mode", "practical-mode"},
		Question:      "Test question",
		Budget:        budget,
		AllowAdvanced: true,
	}
	out, err := buildEnsembleEstimate(catalog, input, ensemble.EstimateOptions{DisableContext: true}, 0)
	if err != nil {
		t.Fatalf("buildEnsembleEstimate error: %v", err)
	}
	if !hasWarning(out.Warnings, "exceed budget") {
		t.Fatalf("expected budget exceed warning, got %v", out.Warnings)
	}
}

func TestResolveModeIDs_ByCode(t *testing.T) {
	catalog := testEstimateCatalog(t)
	ids, err := resolveModeIDs([]string{"A1", "practical-mode"}, catalog)
	if err != nil {
		t.Fatalf("resolveModeIDs error: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 mode IDs, got %d", len(ids))
	}
	if ids[0] != "formal-mode" {
		t.Fatalf("expected A1 to resolve to formal-mode, got %s", ids[0])
	}
}

func hasWarning(warnings []string, needle string) bool {
	for _, w := range warnings {
		if strings.Contains(w, needle) {
			return true
		}
	}
	return false
}

func testEstimateCatalog(t *testing.T) *ensemble.ModeCatalog {
	t.Helper()
	modes := []ensemble.ReasoningMode{
		{
			ID:          "formal-mode",
			Code:        "A1",
			Name:        "Formal Mode",
			Category:    ensemble.CategoryFormal,
			Tier:        ensemble.TierCore,
			ShortDesc:   "Formal reasoning",
			Description: "Formal reasoning description",
			Outputs:     "Proofs",
		},
		{
			ID:          "practical-mode",
			Code:        "G1",
			Name:        "Practical Mode",
			Category:    ensemble.CategoryPractical,
			Tier:        ensemble.TierCore,
			ShortDesc:   "Practical reasoning",
			Description: "Practical reasoning description",
			Outputs:     "Recommendations",
		},
		{
			ID:          "practical-adv",
			Code:        "G2",
			Name:        "Practical Advanced",
			Category:    ensemble.CategoryPractical,
			Tier:        ensemble.TierAdvanced,
			ShortDesc:   "Advanced practical reasoning",
			Description: "Advanced practical reasoning description",
			Outputs:     "Recommendations",
		},
	}

	catalog, err := ensemble.NewModeCatalog(modes, "test")
	if err != nil {
		t.Fatalf("NewModeCatalog error: %v", err)
	}
	return catalog
}
