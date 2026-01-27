package ensemble

import (
	"strings"
	"testing"
	"time"
)

func TestDisagreementAuditor_AuditConflicts(t *testing.T) {
	outputs := []ModeOutput{
		{
			ModeID:          "deductive",
			Thesis:          "Root cause is a missing nil check in handler X.",
			TopFindings:     []Finding{{Finding: "Nil deref in handler", Impact: ImpactHigh, Confidence: 0.7}},
			Risks:           []Risk{{Risk: "Crash on startup", Impact: ImpactCritical, Likelihood: 0.6}},
			Recommendations: []Recommendation{{Recommendation: "Add nil guard", Priority: ImpactHigh}},
			Confidence:      0.8,
			GeneratedAt:     time.Now().UTC(),
		},
		{
			ModeID:          "counterfactual",
			Thesis:          "Root cause is an upstream config mismatch.",
			TopFindings:     []Finding{{Finding: "Config mismatch", Impact: ImpactHigh, Confidence: 0.6}},
			Risks:           []Risk{{Risk: "Silent misconfig", Impact: ImpactMedium, Likelihood: 0.4}},
			Recommendations: []Recommendation{{Recommendation: "Validate config", Priority: ImpactMedium}},
			Confidence:      0.6,
			GeneratedAt:     time.Now().UTC(),
		},
	}

	auditor := NewDisagreementAuditor(outputs, nil)
	report, err := auditor.Audit()
	if err != nil {
		t.Fatalf("Audit error: %v", err)
	}
	if report == nil {
		t.Fatal("expected report, got nil")
	}
	if len(report.Conflicts) == 0 {
		t.Fatal("expected conflicts, got none")
	}
	if len(report.ModeDisagreements) == 0 {
		t.Fatal("expected mode disagreements map")
	}
	if len(report.ResolutionSuggestions) == 0 {
		t.Fatal("expected resolution suggestions")
	}
}

func TestDisagreementAuditor_EmptyOutputs(t *testing.T) {
	auditor := NewDisagreementAuditor(nil, nil)
	if _, err := auditor.Audit(); err == nil {
		t.Fatal("expected error for empty outputs")
	}
}

func TestDisagreementAuditor_GeneratePrompt(t *testing.T) {
	outputs := []ModeOutput{{
		ModeID:      "deductive",
		Thesis:      "Same",
		TopFindings: []Finding{{Finding: "Fact", Impact: ImpactLow, Confidence: 0.5}},
		Confidence:  0.5,
		GeneratedAt: time.Now().UTC(),
	}}
	auditor := NewDisagreementAuditor(outputs, &SynthesisResult{Summary: "summary"})
	prompt := auditor.GeneratePrompt()
	if !strings.Contains(prompt, "DISAGREEMENT AUDITOR") {
		t.Fatalf("prompt missing header")
	}
	if !strings.Contains(prompt, "Output Format") {
		t.Fatalf("prompt missing output format section")
	}
}

func TestDisagreementAuditor_SuggestResolutions(t *testing.T) {
	outputs := []ModeOutput{
		{ModeID: "a", Thesis: "Alpha", TopFindings: []Finding{{Finding: "one", Impact: ImpactLow, Confidence: 0.5}}, Confidence: 0.5, GeneratedAt: time.Now().UTC()},
		{ModeID: "b", Thesis: "Beta", TopFindings: []Finding{{Finding: "two", Impact: ImpactLow, Confidence: 0.5}}, Confidence: 0.5, GeneratedAt: time.Now().UTC()},
	}
	auditor := NewDisagreementAuditor(outputs, nil)
	suggestions := auditor.SuggestResolutions()
	if len(suggestions) == 0 {
		t.Fatal("expected suggestions")
	}
}

func TestPositionsDiverge_SimilarPositions(t *testing.T) {
	positions := []ConflictPosition{
		{ModeID: "a", Position: "Root cause is missing nil check"},
		{ModeID: "b", Position: "Root cause: missing nil check"},
	}
	if positionsDiverge(positions) {
		t.Fatal("expected positions not to diverge")
	}
}

func TestFormatHelpers_NilAndEmpty(t *testing.T) {
	if got := formatOutputs(nil); got != "[]" {
		t.Fatalf("formatOutputs(nil) = %q, want []", got)
	}
	if got := formatSynthesis(nil); got != "{}" {
		t.Fatalf("formatSynthesis(nil) = %q, want {}", got)
	}
}

func TestResolveModeByCode(t *testing.T) {
	catalog := testModeCatalog(t)
	mode, err := resolveModeByCode("A1", catalog)
	if err != nil {
		t.Fatalf("resolveModeByCode error: %v", err)
	}
	if mode == nil || mode.ID != "deductive" {
		t.Fatalf("resolveModeByCode got %v, want deductive", mode)
	}
	if _, err := resolveModeByCode("A9", catalog); err == nil {
		t.Fatal("expected error for out-of-range code")
	}
}

func TestParseModeCode_Invalid(t *testing.T) {
	if _, _, ok := parseModeCode("invalid"); ok {
		t.Fatal("expected parseModeCode to fail")
	}
	if _, _, ok := parseModeCode("A"); ok {
		t.Fatal("expected parseModeCode to fail on short code")
	}
}
