package cli

import (
	"testing"
)

func TestParsePaneName(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		want    *PaneInfo
		wantErr bool
	}{
		{
			name:  "basic cc pane without variant",
			title: "myproject__cc_1",
			want: &PaneInfo{
				Session: "myproject",
				Type:    AgentTypeClaude,
				Index:   1,
				Variant: "",
			},
		},
		{
			name:  "cc pane with model variant",
			title: "myproject__cc_1_opus",
			want: &PaneInfo{
				Session: "myproject",
				Type:    AgentTypeClaude,
				Index:   1,
				Variant: "opus",
			},
		},
		{
			name:  "cc pane with persona variant",
			title: "myproject__cc_2_architect",
			want: &PaneInfo{
				Session: "myproject",
				Type:    AgentTypeClaude,
				Index:   2,
				Variant: "architect",
			},
		},
		{
			name:  "codex pane with model variant",
			title: "backend__cod_3_gpt4",
			want: &PaneInfo{
				Session: "backend",
				Type:    AgentTypeCodex,
				Index:   3,
				Variant: "gpt4",
			},
		},
		{
			name:  "gemini pane without variant",
			title: "frontend__gmi_1",
			want: &PaneInfo{
				Session: "frontend",
				Type:    AgentTypeGemini,
				Index:   1,
				Variant: "",
			},
		},
		{
			name:  "session name with underscores",
			title: "my_cool_project__cc_1_sonnet",
			want: &PaneInfo{
				Session: "my_cool_project",
				Type:    AgentTypeClaude,
				Index:   1,
				Variant: "sonnet",
			},
		},
		{
			name:    "invalid - no double underscore",
			title:   "myproject_cc_1",
			wantErr: true,
		},
		{
			name:    "invalid - no index",
			title:   "myproject__cc",
			wantErr: true,
		},
		{
			name:    "invalid - empty string",
			title:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePaneName(tt.title)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePaneName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Session != tt.want.Session {
				t.Errorf("Session = %q, want %q", got.Session, tt.want.Session)
			}
			if got.Type != tt.want.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.want.Type)
			}
			if got.Index != tt.want.Index {
				t.Errorf("Index = %d, want %d", got.Index, tt.want.Index)
			}
			if got.Variant != tt.want.Variant {
				t.Errorf("Variant = %q, want %q", got.Variant, tt.want.Variant)
			}
		})
	}
}

func TestFormatPaneName(t *testing.T) {
	tests := []struct {
		name      string
		session   string
		agentType AgentType
		index     int
		variant   string
		want      string
	}{
		{
			name:      "without variant",
			session:   "myproject",
			agentType: AgentTypeClaude,
			index:     1,
			variant:   "",
			want:      "myproject__cc_1",
		},
		{
			name:      "with model variant",
			session:   "myproject",
			agentType: AgentTypeClaude,
			index:     2,
			variant:   "opus",
			want:      "myproject__cc_2_opus",
		},
		{
			name:      "codex with variant",
			session:   "backend",
			agentType: AgentTypeCodex,
			index:     1,
			variant:   "gpt4",
			want:      "backend__cod_1_gpt4",
		},
		{
			name:      "gemini without variant",
			session:   "frontend",
			agentType: AgentTypeGemini,
			index:     3,
			variant:   "",
			want:      "frontend__gmi_3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPaneName(tt.session, tt.agentType, tt.index, tt.variant)
			if got != tt.want {
				t.Errorf("FormatPaneName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPaneInfo_HasVariant(t *testing.T) {
	tests := []struct {
		name    string
		variant string
		want    bool
	}{
		{"empty variant", "", false},
		{"with variant", "opus", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PaneInfo{Variant: tt.variant}
			if got := p.HasVariant(); got != tt.want {
				t.Errorf("HasVariant() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPaneInfo_MatchesVariant(t *testing.T) {
	tests := []struct {
		name    string
		variant string
		filter  string
		want    bool
	}{
		{"empty filter matches all", "opus", "", true},
		{"empty filter matches empty variant", "", "", true},
		{"exact match", "opus", "opus", true},
		{"no match", "opus", "sonnet", false},
		{"filter on empty variant", "", "opus", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PaneInfo{Variant: tt.variant}
			if got := p.MatchesVariant(tt.filter); got != tt.want {
				t.Errorf("MatchesVariant(%q) = %v, want %v", tt.filter, got, tt.want)
			}
		})
	}
}

func TestResolveVariant(t *testing.T) {
	tests := []struct {
		name    string
		persona string
		model   string
		want    string
	}{
		{"persona takes precedence", "architect", "opus", "architect"},
		{"model when no persona", "", "opus", "opus"},
		{"empty when neither", "", "", ""},
		{"persona only", "reviewer", "", "reviewer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveVariant(tt.persona, tt.model)
			if got != tt.want {
				t.Errorf("ResolveVariant(%q, %q) = %q, want %q", tt.persona, tt.model, got, tt.want)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that Format -> Parse -> Format produces the same result
	tests := []struct {
		session   string
		agentType AgentType
		index     int
		variant   string
	}{
		{"myproject", AgentTypeClaude, 1, ""},
		{"myproject", AgentTypeClaude, 2, "opus"},
		{"backend", AgentTypeCodex, 1, "gpt4"},
		{"frontend", AgentTypeGemini, 5, "reviewer"},
	}

	for _, tt := range tests {
		title := FormatPaneName(tt.session, tt.agentType, tt.index, tt.variant)
		parsed, err := ParsePaneName(title)
		if err != nil {
			t.Errorf("ParsePaneName(%q) failed: %v", title, err)
			continue
		}

		// Format again and compare
		roundTripped := FormatPaneName(parsed.Session, parsed.Type, parsed.Index, parsed.Variant)
		if roundTripped != title {
			t.Errorf("Round trip failed: %q -> %q", title, roundTripped)
		}
	}
}
