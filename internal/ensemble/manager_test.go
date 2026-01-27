//go:build ensemble_experimental
// +build ensemble_experimental

package ensemble

import (
	"testing"

	"github.com/Dicklesworthstone/ntm/internal/tmux"
)

func TestResolveEnsembleConfig_PresetOverrides(t *testing.T) {
	catalog := testModeCatalog(t)
	preset := EnsemblePreset{
		Name:  "diagnosis",
		Modes: []ModeRef{ModeRefFromID("deductive"), ModeRefFromID("abductive")},
		Synthesis: SynthesisConfig{
			Strategy: StrategyConsensus,
		},
		Budget: BudgetConfig{
			MaxTokensPerMode: 1111,
			MaxTotalTokens:   50000,
		},
	}
	registry := NewEnsembleRegistry([]EnsemblePreset{preset}, catalog)

	cfg := &EnsembleConfig{
		SessionName: "demo",
		Question:    "What is broken?",
		Ensemble:    "diagnosis",
		Synthesis:   SynthesisConfig{Strategy: StrategyCreative},
		Budget:      BudgetConfig{MaxTokensPerMode: 3333},
	}

	modeIDs, resolved, explicitSpecs, err := resolveEnsembleConfig(cfg, catalog, registry)
	if err != nil {
		t.Fatalf("resolveEnsembleConfig error: %v", err)
	}
	if len(modeIDs) != 2 {
		t.Fatalf("modeIDs len = %d, want 2", len(modeIDs))
	}
	if explicitSpecs != nil {
		t.Fatalf("explicit specs should be nil for preset path, got %v", explicitSpecs)
	}
	if resolved.presetName != "diagnosis" {
		t.Fatalf("presetName = %q, want diagnosis", resolved.presetName)
	}
	if resolved.synthesis.Strategy != StrategyCreative {
		t.Fatalf("synthesis strategy = %q, want %q", resolved.synthesis.Strategy, StrategyCreative)
	}
	if resolved.budget.MaxTokensPerMode != 3333 {
		t.Fatalf("budget MaxTokensPerMode = %d, want 3333", resolved.budget.MaxTokensPerMode)
	}
	if resolved.budget.MaxTotalTokens != 50000 {
		t.Fatalf("budget MaxTotalTokens = %d, want 50000", resolved.budget.MaxTotalTokens)
	}
}

func TestResolveEnsembleConfig_ExplicitSpecs(t *testing.T) {
	catalog := testModeCatalog(t)
	registry := NewEnsembleRegistry(nil, catalog)

	cfg := &EnsembleConfig{
		SessionName: "demo",
		Question:    "Test explicit",
		Assignment:  assignmentExplicit,
		Modes:       []string{"deductive:cc,abductive:cod"},
	}

	modeIDs, resolved, explicitSpecs, err := resolveEnsembleConfig(cfg, catalog, registry)
	if err != nil {
		t.Fatalf("resolveEnsembleConfig error: %v", err)
	}
	if len(modeIDs) != 2 {
		t.Fatalf("modeIDs len = %d, want 2", len(modeIDs))
	}
	if len(explicitSpecs) != 2 {
		t.Fatalf("explicit specs len = %d, want 2", len(explicitSpecs))
	}
	if resolved.presetName != "" {
		t.Fatalf("presetName = %q, want empty", resolved.presetName)
	}
}

func TestResolveEnsembleConfig_AdvancedModeRejected(t *testing.T) {
	catalog := testModeCatalog(t)
	preset := EnsemblePreset{
		Name:  "advanced-demo",
		Modes: []ModeRef{ModeRefFromID("advanced-mode"), ModeRefFromID("deductive")},
	}
	registry := NewEnsembleRegistry([]EnsemblePreset{preset}, catalog)

	cfg := &EnsembleConfig{
		SessionName:   "demo",
		Question:      "Test advanced",
		Ensemble:      "advanced-demo",
		AllowAdvanced: false,
	}

	_, _, _, err := resolveEnsembleConfig(cfg, catalog, registry)
	if err == nil {
		t.Fatal("expected error when advanced modes not allowed")
	}
}

func TestBuildPaneSpecs_DefaultsAndValidation(t *testing.T) {
	cfg := &EnsembleConfig{ProjectDir: "/tmp"}

	panes, err := buildPaneSpecs(cfg, 2)
	if err != nil {
		t.Fatalf("buildPaneSpecs error: %v", err)
	}
	if len(panes) != 2 {
		t.Fatalf("panes len = %d, want 2", len(panes))
	}
	for i, pane := range panes {
		if pane.AgentType != string(tmux.AgentClaude) {
			t.Errorf("pane[%d] AgentType = %q, want %q", i, pane.AgentType, tmux.AgentClaude)
		}
		if pane.Project != "/tmp" {
			t.Errorf("pane[%d] Project = %q, want /tmp", i, pane.Project)
		}
	}

	cfg.AgentMix = map[string]int{"cc": 1}
	_, err = buildPaneSpecs(cfg, 2)
	if err == nil {
		t.Fatal("expected error when agent mix insufficient")
	}

	cfg.AgentMix = map[string]int{"": 1}
	_, err = buildPaneSpecs(cfg, 1)
	if err == nil {
		t.Fatal("expected error for empty agent type")
	}
}

func TestNormalizeExplicitSpecs_RejectsDuplicates(t *testing.T) {
	catalog := testModeCatalog(t)
	_, err := normalizeExplicitSpecs([]string{"deductive:cc", "deductive:cod"}, catalog)
	if err == nil {
		t.Fatal("expected error for duplicate mode in explicit specs")
	}
}

func TestValidateResolvedConfig_BudgetDefaults(t *testing.T) {
	catalog := testModeCatalog(t)
	resolved := resolvedEnsembleConfig{
		synthesis: DefaultSynthesisConfig(),
		budget:    BudgetConfig{MaxTokensPerMode: 4000, MaxTotalTokens: 50000},
	}
	modeIDs := []string{"deductive", "abductive"}

	if err := validateResolvedConfig(&resolved, modeIDs, catalog, true); err != nil {
		t.Fatalf("validateResolvedConfig error: %v", err)
	}
}

func TestNormalizeAssignment_Defaults(t *testing.T) {
	if got := normalizeAssignment(""); got != assignmentAffinity {
		t.Fatalf("normalizeAssignment(\"\") = %q, want %q", got, assignmentAffinity)
	}
	if got := normalizeAssignment("round-robin"); got != assignmentRoundRobin {
		t.Fatalf("normalizeAssignment(\"round-robin\") = %q, want %q", got, assignmentRoundRobin)
	}
	if got := normalizeAssignment("explicit"); got != assignmentExplicit {
		t.Fatalf("normalizeAssignment(\"explicit\") = %q, want %q", got, assignmentExplicit)
	}
}

func TestBuildPaneTargetMap_UsesTitleAndID(t *testing.T) {
	panes := []tmux.Pane{
		{ID: "%1", Title: "pane-1", Index: 1},
		{ID: "%2", Title: "pane-2", Index: 2},
	}

	targets := buildPaneTargetMap("session", panes)
	if targets["pane-1"] != "%1" {
		t.Fatalf("target for pane-1 = %q, want %q", targets["pane-1"], "%1")
	}
	if targets["%2"] != "%2" {
		t.Fatalf("target for %%2 = %q, want %q", targets["%2"], "%2")
	}
}
