package ensemble

import (
	"errors"
	"testing"
	"time"
)

func TestPipelineStage_String(t *testing.T) {
	tests := []struct {
		stage PipelineStage
		want  string
	}{
		{StageIntake, "intake"},
		{StageModeRun, "mode_run"},
		{StageSynthesis, "synthesis"},
		{StageComplete, "complete"},
		{StageFailed, "failed"},
	}

	for _, tt := range tests {
		if got := tt.stage.String(); got != tt.want {
			t.Errorf("PipelineStage(%q).String() = %q, want %q", tt.stage, got, tt.want)
		}
	}
}

func TestEnsembleResult_Success(t *testing.T) {
	tests := []struct {
		name   string
		result *EnsembleResult
		want   bool
	}{
		{
			name:   "nil result",
			result: nil,
			want:   false,
		},
		{
			name: "complete without error",
			result: &EnsembleResult{
				Stage: StageComplete,
			},
			want: true,
		},
		{
			name: "complete with error",
			result: &EnsembleResult{
				Stage: StageComplete,
				Error: "something failed",
			},
			want: false,
		},
		{
			name: "failed stage",
			result: &EnsembleResult{
				Stage: StageFailed,
			},
			want: false,
		},
		{
			name: "in progress",
			result: &EnsembleResult{
				Stage: StageModeRun,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Success(); got != tt.want {
				t.Errorf("Success() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnsembleResult_Duration(t *testing.T) {
	t.Run("nil result", func(t *testing.T) {
		var r *EnsembleResult
		if got := r.Duration(); got != 0 {
			t.Errorf("Duration() = %v, want 0", got)
		}
	})

	t.Run("completed result", func(t *testing.T) {
		start := time.Now().Add(-5 * time.Minute)
		end := start.Add(3 * time.Minute)
		r := &EnsembleResult{
			StartedAt:   start,
			CompletedAt: end,
		}
		got := r.Duration()
		want := 3 * time.Minute
		if got != want {
			t.Errorf("Duration() = %v, want %v", got, want)
		}
	})

	t.Run("in progress result", func(t *testing.T) {
		start := time.Now().Add(-1 * time.Minute)
		r := &EnsembleResult{
			StartedAt: start,
		}
		got := r.Duration()
		// Should be roughly 1 minute (allow some tolerance)
		if got < 59*time.Second || got > 61*time.Second {
			t.Errorf("Duration() = %v, expected ~1 minute", got)
		}
	})
}

func TestSynthesisReport_Validate(t *testing.T) {
	tests := []struct {
		name    string
		report  *SynthesisReport
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil report",
			report:  nil,
			wantErr: true,
			errMsg:  "synthesis report is nil",
		},
		{
			name: "missing thesis",
			report: &SynthesisReport{
				TopFindings: []Finding{{Finding: "test finding", Impact: ImpactHigh, Confidence: 0.8}},
				Confidence:  0.8,
			},
			wantErr: true,
			errMsg:  "consolidated_thesis is required",
		},
		{
			name: "missing findings",
			report: &SynthesisReport{
				ConsolidatedThesis: "A thesis",
				Confidence:         0.8,
			},
			wantErr: true,
			errMsg:  "at least one finding is required",
		},
		{
			name: "invalid confidence",
			report: &SynthesisReport{
				ConsolidatedThesis: "A thesis",
				TopFindings:        []Finding{{Finding: "test finding", Impact: ImpactHigh, Confidence: 0.8}},
				Confidence:         1.5, // Invalid
			},
			wantErr: true,
			errMsg:  "invalid confidence",
		},
		{
			name: "valid report",
			report: &SynthesisReport{
				ConsolidatedThesis: "A thesis",
				TopFindings:        []Finding{{Finding: "test finding", Impact: ImpactHigh, Confidence: 0.8}},
				Confidence:         0.8,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.report.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg && !contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestPipelineError(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := NewPipelineError(StageIntake, "failed to generate context", cause)

		if err.Stage != StageIntake {
			t.Errorf("Stage = %v, want %v", err.Stage, StageIntake)
		}

		got := err.Error()
		want := "stage intake: failed to generate context: underlying error"
		if got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}

		if unwrapped := errors.Unwrap(err); unwrapped != cause {
			t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
		}
	})

	t.Run("without cause", func(t *testing.T) {
		err := NewPipelineError(StageModeRun, "spawn failed", nil)

		got := err.Error()
		want := "stage mode_run: spawn failed"
		if got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}

		if unwrapped := errors.Unwrap(err); unwrapped != nil {
			t.Errorf("Unwrap() = %v, want nil", unwrapped)
		}
	})
}

// TestValidatePipelineConfig is in pipeline_experimental_test.go
// since EnsembleConfig requires the ensemble_experimental build tag.

func TestDefaultRunConfig(t *testing.T) {
	cfg := DefaultRunConfig()

	if cfg.SkipStage1 {
		t.Error("SkipStage1 should default to false")
	}
	if cfg.SkipStage3 {
		t.Error("SkipStage3 should default to false")
	}
	if cfg.CollectTimeout != 10*time.Minute {
		t.Errorf("CollectTimeout = %v, want 10m", cfg.CollectTimeout)
	}
}

func TestDefaultOutputCollectorConfig(t *testing.T) {
	cfg := DefaultOutputCollectorConfig()

	if cfg.PollInterval != 5*time.Second {
		t.Errorf("PollInterval = %v, want 5s", cfg.PollInterval)
	}
	if cfg.Timeout != 10*time.Minute {
		t.Errorf("Timeout = %v, want 10m", cfg.Timeout)
	}
	if cfg.RequireAll {
		t.Error("RequireAll should default to false")
	}
	if cfg.MinOutputs != 1 {
		t.Errorf("MinOutputs = %d, want 1", cfg.MinOutputs)
	}
}

func TestStage2Result_Fields(t *testing.T) {
	result := Stage2Result{
		SessionName:    "test-session",
		ModesAttempted: 5,
		ModesSucceeded: 3,
		ModesFailed:    2,
		Duration:       30 * time.Second,
		EarlyStopped:   true,
		StopReason:     "consensus reached",
	}

	if result.SessionName != "test-session" {
		t.Errorf("SessionName = %q, want %q", result.SessionName, "test-session")
	}
	if result.ModesAttempted != 5 {
		t.Errorf("ModesAttempted = %d, want 5", result.ModesAttempted)
	}
	if result.ModesSucceeded != 3 {
		t.Errorf("ModesSucceeded = %d, want 3", result.ModesSucceeded)
	}
	if result.ModesFailed != 2 {
		t.Errorf("ModesFailed = %d, want 2", result.ModesFailed)
	}
	if !result.EarlyStopped {
		t.Error("EarlyStopped should be true")
	}
}

func TestPipelineMetrics_Fields(t *testing.T) {
	metrics := PipelineMetrics{
		IntakeDuration:     5 * time.Second,
		ModeRunDuration:    2 * time.Minute,
		SynthesisDuration:  30 * time.Second,
		ContextTokens:      500,
		ModeTokens:         8000,
		SynthesisTokens:    1000,
		TotalTokens:        9500,
		ModesAttempted:     5,
		ModesSucceeded:     4,
		ModesFailed:        1,
		EarlyStopTriggered: true,
		EarlyStopReason:    "budget exceeded",
	}

	if metrics.IntakeDuration != 5*time.Second {
		t.Errorf("IntakeDuration = %v, want 5s", metrics.IntakeDuration)
	}
	if metrics.TotalTokens != 9500 {
		t.Errorf("TotalTokens = %d, want 9500", metrics.TotalTokens)
	}
	if !metrics.EarlyStopTriggered {
		t.Error("EarlyStopTriggered should be true")
	}
}

func TestSynthesisAgreement_Fields(t *testing.T) {
	agreement := SynthesisAgreement{
		Finding:      "Memory leak in handler",
		ModeIDs:      []string{"deductive", "systems-thinking", "failure-mode"},
		Confidence:   0.9,
		EvidenceRefs: []string{"finding-1", "finding-2"},
	}

	if agreement.Finding != "Memory leak in handler" {
		t.Errorf("Finding = %q, want %q", agreement.Finding, "Memory leak in handler")
	}
	if len(agreement.ModeIDs) != 3 {
		t.Errorf("len(ModeIDs) = %d, want 3", len(agreement.ModeIDs))
	}
	if agreement.Confidence != 0.9 {
		t.Errorf("Confidence = %v, want 0.9", agreement.Confidence)
	}
}

func TestSynthesisDisagreement_Fields(t *testing.T) {
	disagreement := SynthesisDisagreement{
		Topic: "Root cause of performance issue",
		Positions: []DisagreementPosition{
			{ModeID: "deductive", Position: "Database queries", Confidence: 0.7},
			{ModeID: "systems-thinking", Position: "Network latency", Confidence: 0.6},
		},
		Resolution:       "Database queries (higher confidence)",
		ResolutionMethod: "confidence_voting",
	}

	if disagreement.Topic != "Root cause of performance issue" {
		t.Errorf("Topic = %q", disagreement.Topic)
	}
	if len(disagreement.Positions) != 2 {
		t.Errorf("len(Positions) = %d, want 2", len(disagreement.Positions))
	}
	if disagreement.ResolutionMethod != "confidence_voting" {
		t.Errorf("ResolutionMethod = %q, want %q", disagreement.ResolutionMethod, "confidence_voting")
	}
}

func TestModeContribution_Fields(t *testing.T) {
	contrib := ModeContribution{
		ModeID:              "systems-thinking",
		FindingsContributed: 5,
		UniqueInsights:      2,
		AgreementCount:      3,
		DisagreementCount:   1,
		OverallWeight:       0.85,
	}

	if contrib.ModeID != "systems-thinking" {
		t.Errorf("ModeID = %q", contrib.ModeID)
	}
	if contrib.FindingsContributed != 5 {
		t.Errorf("FindingsContributed = %d, want 5", contrib.FindingsContributed)
	}
	if contrib.OverallWeight != 0.85 {
		t.Errorf("OverallWeight = %v, want 0.85", contrib.OverallWeight)
	}
}

func TestAuditEntry_Fields(t *testing.T) {
	now := time.Now()
	entry := AuditEntry{
		Timestamp: now,
		Action:    "merge_findings",
		Details:   "Merged 3 similar findings",
		ModeID:    "deductive",
	}

	if entry.Timestamp != now {
		t.Errorf("Timestamp mismatch")
	}
	if entry.Action != "merge_findings" {
		t.Errorf("Action = %q", entry.Action)
	}
	if entry.ModeID != "deductive" {
		t.Errorf("ModeID = %q", entry.ModeID)
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
