package ensemble

import (
	"context"
	"encoding/json"
	"testing"
)

func TestStream_ChunkOrdering(t *testing.T) {
	input := map[string]any{"mode": "mode-a"}
	logTestStartStream(t, input)

	synth, err := NewSynthesizer(SynthesisConfig{Strategy: StrategyManual})
	assertNoErrorStream(t, "new synthesizer", err)

	chunks, errs := synth.StreamSynthesize(context.Background(), sampleSynthesisInput())
	var collected []SynthesisChunk
	for chunk := range chunks {
		collected = append(collected, chunk)
	}
	if err := <-errs; err != nil {
		assertNoErrorStream(t, "stream error", err)
	}
	logTestResultStream(t, collected)

	assertTrueStream(t, "chunks collected", len(collected) > 0)
	for i, chunk := range collected {
		assertEqualStream(t, "index ordering", chunk.Index, i+1)
	}
	assertEqualStream(t, "last chunk type", collected[len(collected)-1].Type, ChunkComplete)
}

func TestStream_CancelResume(t *testing.T) {
	input := map[string]any{"mode": "mode-a", "cancel": true}
	logTestStartStream(t, input)

	synth, err := NewSynthesizer(SynthesisConfig{Strategy: StrategyManual})
	assertNoErrorStream(t, "new synthesizer", err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	chunks, errs := synth.StreamSynthesize(ctx, sampleSynthesisInput())

	for range chunks {
		// Should not receive chunks on a canceled context.
	}
	err = <-errs
	logTestResultStream(t, err)
	assertTrueStream(t, "context canceled", err == context.Canceled)
}

func TestStream_JSONLEvents(t *testing.T) {
	input := map[string]any{"mode": "mode-a", "format": "jsonl"}
	logTestStartStream(t, input)

	synth, err := NewSynthesizer(SynthesisConfig{Strategy: StrategyManual})
	assertNoErrorStream(t, "new synthesizer", err)

	chunks, errs := synth.StreamSynthesize(context.Background(), sampleSynthesisInput())
	var lines [][]byte
	for chunk := range chunks {
		data, err := json.Marshal(chunk)
		assertNoErrorStream(t, "marshal chunk", err)
		lines = append(lines, data)
	}
	if err := <-errs; err != nil {
		assertNoErrorStream(t, "stream error", err)
	}
	logTestResultStream(t, len(lines))

	for _, line := range lines {
		var decoded SynthesisChunk
		assertNoErrorStream(t, "unmarshal chunk", json.Unmarshal(line, &decoded))
	}
}

func sampleSynthesisInput() *SynthesisInput {
	return &SynthesisInput{
		OriginalQuestion: "What is the architecture?",
		Outputs: []ModeOutput{
			{
				ModeID:     "mode-a",
				Thesis:     "Architecture uses services",
				Confidence: 0.8,
				TopFindings: []Finding{
					{Finding: "Uses REST", Impact: ImpactMedium, Confidence: 0.9},
				},
				Risks: []Risk{
					{Risk: "Latency", Impact: ImpactMedium, Likelihood: 0.5},
				},
				Recommendations: []Recommendation{
					{Recommendation: "Add caching", Priority: ImpactHigh},
				},
				QuestionsForUser: []Question{
					{Question: "Which services are critical?"},
				},
			},
		},
	}
}

func logTestStartStream(t *testing.T, input any) {
	t.Helper()
	t.Logf("TEST: %s - starting with input: %v", t.Name(), input)
}

func logTestResultStream(t *testing.T, result any) {
	t.Helper()
	t.Logf("TEST: %s - got result: %v", t.Name(), result)
}

func assertNoErrorStream(t *testing.T, desc string, err error) {
	t.Helper()
	t.Logf("TEST: %s - assertion: %s", t.Name(), desc)
	if err != nil {
		t.Fatalf("%s: %v", desc, err)
	}
}

func assertTrueStream(t *testing.T, desc string, ok bool) {
	t.Helper()
	t.Logf("TEST: %s - assertion: %s", t.Name(), desc)
	if !ok {
		t.Fatalf("assertion failed: %s", desc)
	}
}

func assertEqualStream(t *testing.T, desc string, got, want any) {
	t.Helper()
	t.Logf("TEST: %s - assertion: %s", t.Name(), desc)
	if got != want {
		t.Fatalf("%s: got %v want %v", desc, got, want)
	}
}
