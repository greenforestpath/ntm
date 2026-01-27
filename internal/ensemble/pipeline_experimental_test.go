//go:build ensemble_experimental
// +build ensemble_experimental

package ensemble

import "testing"

func TestValidatePipelineConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *EnsembleConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
		},
		{
			name:    "missing session name",
			cfg:     &EnsembleConfig{Question: "test?", Ensemble: "test"},
			wantErr: true,
		},
		{
			name:    "missing question",
			cfg:     &EnsembleConfig{SessionName: "test", Ensemble: "test"},
			wantErr: true,
		},
		{
			name:    "missing modes and ensemble",
			cfg:     &EnsembleConfig{SessionName: "test", Question: "test?"},
			wantErr: true,
		},
		{
			name:    "both modes and ensemble",
			cfg:     &EnsembleConfig{SessionName: "test", Question: "test?", Ensemble: "test", Modes: []string{"A1"}},
			wantErr: true,
		},
		{
			name:    "valid with ensemble",
			cfg:     &EnsembleConfig{SessionName: "test", Question: "test?", Ensemble: "test"},
			wantErr: false,
		},
		{
			name:    "valid with modes",
			cfg:     &EnsembleConfig{SessionName: "test", Question: "test?", Modes: []string{"A1", "B1"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePipelineConfig(tt.cfg)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEnsembleManager_RunStage1_NilConfig(t *testing.T) {
	m := &EnsembleManager{}
	_, err := m.RunStage1(nil, nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestEnsembleManager_RunStage2_NilConfig(t *testing.T) {
	m := &EnsembleManager{}
	_, err := m.RunStage2(nil, nil, nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestEnsembleManager_RunStage3_NilConfig(t *testing.T) {
	m := &EnsembleManager{}
	_, err := m.RunStage3(nil, nil, nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestEnsembleManager_RunStage3_EmptyOutputs(t *testing.T) {
	m := &EnsembleManager{}
	cfg := &EnsembleConfig{
		SessionName: "test",
		Question:    "test?",
		Ensemble:    "test",
	}
	_, err := m.RunStage3(nil, cfg, []ModeOutput{})
	if err == nil {
		t.Error("expected error for empty outputs")
	}
}
