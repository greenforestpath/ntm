package ensemble

import (
	"reflect"
	"testing"
)

func TestExport_SingleFinding(t *testing.T) {
	input := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.9,
			TopFindings: []Finding{
				{Finding: "Single finding", Impact: ImpactHigh, Confidence: 0.8},
			},
		},
	}
	logTestStartExport(t, input)

	merged := MergeOutputs(input, DefaultMergeConfig())
	logTestResultExport(t, merged)

	assertTrueExport(t, "merged output present", merged != nil)
	assertEqualExport(t, "findings count", len(merged.Findings), 1)
	assertEqualExport(t, "stats total findings", merged.Stats.TotalFindings, 1)
}

func TestExport_BatchMode(t *testing.T) {
	input := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.9,
			TopFindings: []Finding{
				{Finding: "Finding A", Impact: ImpactHigh, Confidence: 0.8},
			},
		},
		{
			ModeID:     "mode-b",
			Confidence: 0.7,
			TopFindings: []Finding{
				{Finding: "Finding B", Impact: ImpactMedium, Confidence: 0.7},
			},
		},
	}
	logTestStartExport(t, input)

	merged := MergeOutputs(input, DefaultMergeConfig())
	logTestResultExport(t, merged)

	assertEqualExport(t, "input count", merged.Stats.InputCount, 2)
	assertTrueExport(t, "batch findings collected", len(merged.Findings) >= 2)
}

func TestExport_DryRun(t *testing.T) {
	input := []ModeOutput{
		{
			ModeID:     "mode-a",
			Confidence: 0.9,
			TopFindings: []Finding{
				{Finding: "Stable finding", Impact: ImpactMedium, Confidence: 0.7},
			},
		},
	}
	original := deepCopyOutputs(input)
	logTestStartExport(t, map[string]any{"outputs": input, "original": original})

	_ = MergeOutputs(input, DefaultMergeConfig())
	logTestResultExport(t, input)

	assertTrueExport(t, "merge does not mutate input", reflect.DeepEqual(input, original))
}

func TestExport_PriorityMapping(t *testing.T) {
	input := []ImpactLevel{ImpactCritical, ImpactHigh, ImpactMedium, ImpactLow}
	logTestStartExport(t, input)

	critical := impactWeight(ImpactCritical)
	high := impactWeight(ImpactHigh)
	medium := impactWeight(ImpactMedium)
	low := impactWeight(ImpactLow)
	logTestResultExport(t, map[string]float64{"critical": critical, "high": high, "medium": medium, "low": low})

	assertTrueExport(t, "critical > high", critical > high)
	assertTrueExport(t, "high > medium", high > medium)
	assertTrueExport(t, "medium > low", medium > low)
}

func deepCopyOutputs(outputs []ModeOutput) []ModeOutput {
	copyOutputs := make([]ModeOutput, len(outputs))
	for i := range outputs {
		copyOutputs[i] = outputs[i]
		if outputs[i].TopFindings != nil {
			copyOutputs[i].TopFindings = append([]Finding(nil), outputs[i].TopFindings...)
		}
		if outputs[i].Risks != nil {
			copyOutputs[i].Risks = append([]Risk(nil), outputs[i].Risks...)
		}
		if outputs[i].Recommendations != nil {
			copyOutputs[i].Recommendations = append([]Recommendation(nil), outputs[i].Recommendations...)
		}
	}
	return copyOutputs
}

func logTestStartExport(t *testing.T, input any) {
	t.Helper()
	t.Logf("TEST: %s - starting with input: %v", t.Name(), input)
}

func logTestResultExport(t *testing.T, result any) {
	t.Helper()
	t.Logf("TEST: %s - got result: %v", t.Name(), result)
}

func assertTrueExport(t *testing.T, desc string, ok bool) {
	t.Helper()
	t.Logf("TEST: %s - assertion: %s", t.Name(), desc)
	if !ok {
		t.Fatalf("assertion failed: %s", desc)
	}
}

func assertEqualExport(t *testing.T, desc string, got, want any) {
	t.Helper()
	t.Logf("TEST: %s - assertion: %s", t.Name(), desc)
	if got != want {
		t.Fatalf("%s: got %v want %v", desc, got, want)
	}
}
