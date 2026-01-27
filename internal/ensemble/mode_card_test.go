package ensemble

import (
	"strings"
	"testing"
)

func TestNewModeCard(t *testing.T) {
	tests := []struct {
		name     string
		mode     *ReasoningMode
		wantNil  bool
		checkID  string
	}{
		{
			name:    "nil mode",
			mode:    nil,
			wantNil: true,
		},
		{
			name: "valid mode",
			mode: &ReasoningMode{
				ID:          "test-mode",
				Code:        "T1",
				Name:        "Test Mode",
				Category:    CategoryFormal,
				Tier:        TierCore,
				Icon:        "🧪",
				ShortDesc:   "A test mode",
				Description: "Longer description",
				BestFor:     []string{"Testing"},
			},
			wantNil: false,
			checkID: "test-mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := NewModeCard(tt.mode)
			if tt.wantNil {
				if card != nil {
					t.Error("expected nil card")
				}
				return
			}
			if card == nil {
				t.Fatal("expected non-nil card")
			}
			if card.ModeID != tt.checkID {
				t.Errorf("ModeID = %q, want %q", card.ModeID, tt.checkID)
			}
			if card.Code != tt.mode.Code {
				t.Errorf("Code = %q, want %q", card.Code, tt.mode.Code)
			}
			if card.Name != tt.mode.Name {
				t.Errorf("Name = %q, want %q", card.Name, tt.mode.Name)
			}
		})
	}
}

func TestModeCatalog_GetModeCard(t *testing.T) {
	catalog, err := LoadModeCatalog()
	if err != nil {
		t.Fatalf("LoadModeCatalog: %v", err)
	}

	tests := []struct {
		name    string
		modeRef string
		wantErr bool
		checkID string
	}{
		{
			name:    "by ID",
			modeRef: "deductive",
			wantErr: false,
			checkID: "deductive",
		},
		{
			name:    "by code",
			modeRef: "A1",
			wantErr: false,
			checkID: "deductive",
		},
		{
			name:    "not found",
			modeRef: "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card, err := catalog.GetModeCard(tt.modeRef)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetModeCard: %v", err)
			}
			if card.ModeID != tt.checkID {
				t.Errorf("ModeID = %q, want %q", card.ModeID, tt.checkID)
			}
			// Check that card has examples
			if len(card.Examples) == 0 {
				t.Error("expected card to have examples")
			}
			// Check typical cost
			if card.TypicalCost == 0 {
				t.Error("expected non-zero TypicalCost")
			}
		})
	}
}

func TestModeCatalog_GetModeCard_NilCatalog(t *testing.T) {
	var catalog *ModeCatalog
	_, err := catalog.GetModeCard("deductive")
	if err == nil {
		t.Error("expected error for nil catalog")
	}
}

func TestGenerateExamples(t *testing.T) {
	tests := []struct {
		name      string
		category  ModeCategory
		wantCount int
	}{
		{name: "nil mode", category: "", wantCount: 0},
		{name: "Formal", category: CategoryFormal, wantCount: 2},
		{name: "Ampliative", category: CategoryAmpliative, wantCount: 2},
		{name: "Uncertainty", category: CategoryUncertainty, wantCount: 2},
		{name: "Causal", category: CategoryCausal, wantCount: 2},
		{name: "Practical", category: CategoryPractical, wantCount: 2},
		{name: "Strategic", category: CategoryStrategic, wantCount: 2},
		{name: "Dialectical", category: CategoryDialectical, wantCount: 2},
		{name: "unknown", category: ModeCategory("Unknown"), wantCount: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mode *ReasoningMode
			if tt.category != "" {
				mode = &ReasoningMode{
					ID:       "test",
					Name:     "Test Mode",
					Category: tt.category,
				}
			}
			examples := generateExamples(mode)
			if len(examples) != tt.wantCount {
				t.Errorf("got %d examples, want %d", len(examples), tt.wantCount)
			}
		})
	}
}

func TestEstimateTypicalCost(t *testing.T) {
	tests := []struct {
		name     string
		category ModeCategory
		tier     ModeTier
		wantMin  int
		wantMax  int
	}{
		{name: "nil mode", category: "", tier: "", wantMin: 0, wantMax: 0},
		{name: "Formal core", category: CategoryFormal, tier: TierCore, wantMin: 3000, wantMax: 3000},
		{name: "Meta core", category: CategoryMeta, tier: TierCore, wantMin: 2500, wantMax: 2500},
		{name: "Strategic core", category: CategoryStrategic, tier: TierCore, wantMin: 2500, wantMax: 2500},
		{name: "Dialectical core", category: CategoryDialectical, tier: TierCore, wantMin: 2800, wantMax: 2800},
		{name: "default advanced", category: CategoryCausal, tier: TierAdvanced, wantMin: 2400, wantMax: 2400},
		{name: "default experimental", category: CategoryCausal, tier: TierExperimental, wantMin: 3000, wantMax: 3000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mode *ReasoningMode
			if tt.category != "" {
				mode = &ReasoningMode{
					ID:       "test",
					Category: tt.category,
					Tier:     tt.tier,
				}
			}
			cost := estimateTypicalCost(mode)
			if cost < tt.wantMin || cost > tt.wantMax {
				t.Errorf("cost = %d, want between %d and %d", cost, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestFindComplements(t *testing.T) {
	catalog, err := LoadModeCatalog()
	if err != nil {
		t.Fatalf("LoadModeCatalog: %v", err)
	}

	tests := []struct {
		name     string
		modeID   string
		wantSome bool
	}{
		{name: "deductive (Formal)", modeID: "deductive", wantSome: true},
		{name: "inductive (Ampliative)", modeID: "inductive", wantSome: true},
		{name: "root-cause (Causal)", modeID: "root-cause", wantSome: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := catalog.GetMode(tt.modeID)
			if mode == nil {
				t.Fatalf("mode %q not found", tt.modeID)
			}
			complements := findComplements(catalog, mode)
			if tt.wantSome && len(complements) == 0 {
				t.Error("expected some complements")
			}
		})
	}
}

func TestFindComplements_NilInputs(t *testing.T) {
	catalog, _ := LoadModeCatalog()

	// Nil catalog
	complements := findComplements(nil, &ReasoningMode{})
	if complements != nil {
		t.Error("expected nil for nil catalog")
	}

	// Nil mode
	complements = findComplements(catalog, nil)
	if complements != nil {
		t.Error("expected nil for nil mode")
	}
}

func TestFormatCard(t *testing.T) {
	tests := []struct {
		name     string
		card     *ModeCard
		contains []string
	}{
		{
			name:     "nil card",
			card:     nil,
			contains: nil,
		},
		{
			name: "full card",
			card: &ModeCard{
				ModeID:         "test-mode",
				Code:           "T1",
				Name:           "Test Mode",
				Category:       CategoryFormal,
				Tier:           TierCore,
				ShortDesc:      "A test mode for testing",
				Description:    "This is a longer description.",
				Differentiator: "What makes it unique",
				BestFor:        []string{"Testing", "Validation"},
				Examples:       []string{"Example prompt 1"},
				Outputs:        "Test outputs",
				FailureModes:   []string{"Can fail if X"},
				TypicalCost:    2500,
				Complements:    []string{"mode-a", "mode-b"},
			},
			contains: []string{
				"Test Mode (T1)",
				"ID: test-mode",
				"Category: Formal",
				"Tier: core",
				"A test mode for testing",
				"What makes it unique",
				"Testing",
				"Validation",
				"Example prompt 1",
				"Test outputs",
				"Can fail if X",
				"~2500",
				"mode-a, mode-b",
			},
		},
		{
			name: "card with icon",
			card: &ModeCard{
				ModeID:    "icon-mode",
				Code:      "I1",
				Name:      "Icon Mode",
				Icon:      "🔬",
				Category:  CategoryMeta,
				Tier:      TierAdvanced,
				ShortDesc: "Has an icon",
			},
			contains: []string{"🔬 Icon Mode (I1)"},
		},
		{
			name: "card without icon uses default",
			card: &ModeCard{
				ModeID:    "no-icon",
				Code:      "N1",
				Name:      "No Icon Mode",
				Category:  CategoryPractical,
				Tier:      TierCore,
				ShortDesc: "No icon provided",
			},
			contains: []string{"📋 No Icon Mode (N1)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatCard(tt.card)
			if tt.card == nil {
				if output != "" {
					t.Errorf("expected empty string for nil card, got %q", output)
				}
				return
			}
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q:\n%s", want, output)
				}
			}
		})
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		width  int
		prefix string
		want   string
	}{
		{
			name:   "empty text",
			text:   "",
			width:  70,
			prefix: "   ",
			want:   "",
		},
		{
			name:   "short text",
			text:   "Hello world",
			width:  70,
			prefix: "   ",
			want:   "   Hello world",
		},
		{
			name:   "text that wraps",
			text:   "This is a very long sentence that should wrap at the specified width boundary",
			width:  30,
			prefix: "",
			want:   "This is a very long sentence\nthat should wrap at the\nspecified width boundary",
		},
		{
			name:   "text with prefix",
			text:   "Word1 Word2 Word3 Word4",
			width:  15,
			prefix: ">> ",
			want:   ">> Word1 Word2\n>> Word3 Word4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapText(tt.text, tt.width, tt.prefix)
			if got != tt.want {
				t.Errorf("wrapText() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}
