//go:build ensemble_experimental
// +build ensemble_experimental

package ensemble

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func TestEstimateEnsemble_OverBudgetWarningAndCap(t *testing.T) {
	mode := ReasoningMode{
		ID:          "formal-expensive",
		Code:        "A1",
		Name:        "Formal Expensive",
		Category:    CategoryFormal,
		Tier:        TierCore,
		ShortDesc:   "Short",
		Description: "Long description",
		Outputs:     "Outputs",
	}

	catalog, err := NewModeCatalog([]ReasoningMode{mode}, "1.0")
	if err != nil {
		t.Fatalf("NewModeCatalog: %v", err)
	}

	manager := &EnsembleManager{
		Catalog: catalog,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	cfg := &EnsembleConfig{
		Question: "Test question",
		Modes:    []string{mode.ID},
		Budget: BudgetConfig{
			MaxTokensPerMode: 500,
			MaxTotalTokens:   600,
		},
	}

	opts := EstimateOptions{
		ContextPack: &ContextPack{TokenEstimate: 100},
	}

	estimate, err := manager.EstimateEnsemble(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("EstimateEnsemble: %v", err)
	}
	if len(estimate.Modes) != 1 {
		t.Fatalf("expected 1 mode estimate, got %d", len(estimate.Modes))
	}
	if estimate.Modes[0].OutputTokens != 500 {
		t.Fatalf("output tokens = %d, want 500", estimate.Modes[0].OutputTokens)
	}
	if !estimate.OverBudget {
		t.Fatalf("expected over budget")
	}
	if len(estimate.Warnings) == 0 {
		t.Fatalf("expected warnings for budget overage")
	}
}

func TestEstimateEnsemble_AlternativesSuggested(t *testing.T) {
	expensive := ReasoningMode{
		ID:          "formal-exp",
		Code:        "A1",
		Name:        "Formal Experimental",
		Category:    CategoryFormal,
		Tier:        TierExperimental,
		ShortDesc:   "Short",
		Description: "Long description",
		Outputs:     "Outputs",
	}
	cheaper := ReasoningMode{
		ID:          "formal-core",
		Code:        "A2",
		Name:        "Formal Core",
		Category:    CategoryFormal,
		Tier:        TierCore,
		ShortDesc:   "Short",
		Description: "Long description",
		Outputs:     "Outputs",
	}

	catalog, err := NewModeCatalog([]ReasoningMode{expensive, cheaper}, "1.0")
	if err != nil {
		t.Fatalf("NewModeCatalog: %v", err)
	}

	manager := &EnsembleManager{
		Catalog: catalog,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	cfg := &EnsembleConfig{
		Question:      "Test question",
		Modes:         []string{expensive.ID},
		AllowAdvanced: true,
		Budget: BudgetConfig{
			MaxTokensPerMode: 5000,
			MaxTotalTokens:   1000,
		},
	}

	opts := EstimateOptions{
		DisableContext: true,
	}

	estimate, err := manager.EstimateEnsemble(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("EstimateEnsemble: %v", err)
	}
	if len(estimate.Modes) != 1 {
		t.Fatalf("expected 1 mode estimate, got %d", len(estimate.Modes))
	}
	alts := estimate.Modes[0].Alternatives
	if len(alts) == 0 {
		t.Fatalf("expected alternatives for over-budget estimate")
	}
	if alts[0].ID != cheaper.ID {
		t.Fatalf("expected alternative %s, got %s", cheaper.ID, alts[0].ID)
	}
}
