package ensemble

import (
	"testing"
	"time"
)

func TestModeCategory_IsValid(t *testing.T) {
	tests := []struct {
		cat   ModeCategory
		valid bool
	}{
		{CategoryFormal, true},
		{CategoryAmpliative, true},
		{CategoryUncertainty, true},
		{CategoryVagueness, true},
		{CategoryChange, true},
		{CategoryCausal, true},
		{CategoryPractical, true},
		{CategoryStrategic, true},
		{CategoryDialectical, true},
		{CategoryModal, true},
		{CategoryDomain, true},
		{CategoryMeta, true},
		{ModeCategory("invalid"), false},
		{ModeCategory(""), false},
	}

	for _, tt := range tests {
		if got := tt.cat.IsValid(); got != tt.valid {
			t.Errorf("ModeCategory(%q).IsValid() = %v, want %v", tt.cat, got, tt.valid)
		}
	}
}

func TestValidateModeID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"deductive", false},
		{"bayesian-inference", false},
		{"foo-bar-baz", false},
		{"a", false},
		{"a1", false},
		{"", true},                       // empty
		{"Deductive", true},              // uppercase
		{"123abc", true},                 // starts with number
		{"-invalid", true},               // starts with hyphen
		{"has spaces", true},             // contains spaces
		{"has_underscore", true},         // contains underscore
		{string(make([]byte, 65)), true}, // too long
	}

	for _, tt := range tests {
		err := ValidateModeID(tt.id)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateModeID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
		}
	}
}

func TestReasoningMode_Validate(t *testing.T) {
	validMode := ReasoningMode{
		ID:        "deductive",
		Name:      "Deductive Logic",
		Category:  CategoryFormal,
		ShortDesc: "Derive conclusions from premises",
	}

	if err := validMode.Validate(); err != nil {
		t.Errorf("valid mode should pass validation: %v", err)
	}

	// Test missing ID
	noID := validMode
	noID.ID = ""
	if err := noID.Validate(); err == nil {
		t.Error("mode without ID should fail validation")
	}

	// Test invalid ID
	invalidID := validMode
	invalidID.ID = "INVALID"
	if err := invalidID.Validate(); err == nil {
		t.Error("mode with invalid ID should fail validation")
	}

	// Test missing name
	noName := validMode
	noName.Name = ""
	if err := noName.Validate(); err == nil {
		t.Error("mode without name should fail validation")
	}

	// Test invalid category
	invalidCat := validMode
	invalidCat.Category = "invalid"
	if err := invalidCat.Validate(); err == nil {
		t.Error("mode with invalid category should fail validation")
	}

	// Test long short_desc
	longDesc := validMode
	longDesc.ShortDesc = string(make([]byte, 100))
	if err := longDesc.Validate(); err == nil {
		t.Error("mode with short_desc > 80 chars should fail validation")
	}
}

func TestAssignmentStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   AssignmentStatus
		terminal bool
	}{
		{AssignmentPending, false},
		{AssignmentInjecting, false},
		{AssignmentActive, false},
		{AssignmentDone, true},
		{AssignmentError, true},
	}

	for _, tt := range tests {
		if got := tt.status.IsTerminal(); got != tt.terminal {
			t.Errorf("AssignmentStatus(%q).IsTerminal() = %v, want %v", tt.status, got, tt.terminal)
		}
	}
}

func TestEnsembleStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   EnsembleStatus
		terminal bool
	}{
		{EnsembleSpawning, false},
		{EnsembleInjecting, false},
		{EnsembleActive, false},
		{EnsembleSynthesizing, false},
		{EnsembleComplete, true},
		{EnsembleError, true},
	}

	for _, tt := range tests {
		if got := tt.status.IsTerminal(); got != tt.terminal {
			t.Errorf("EnsembleStatus(%q).IsTerminal() = %v, want %v", tt.status, got, tt.terminal)
		}
	}
}

func TestSynthesisStrategy_IsValid(t *testing.T) {
	tests := []struct {
		strategy SynthesisStrategy
		valid    bool
	}{
		{StrategyConsensus, true},
		{StrategyDebate, true},
		{StrategyWeighted, true},
		{StrategySequential, true},
		{StrategyBestOf, true},
		{SynthesisStrategy("invalid"), false},
		{SynthesisStrategy(""), false},
	}

	for _, tt := range tests {
		if got := tt.strategy.IsValid(); got != tt.valid {
			t.Errorf("SynthesisStrategy(%q).IsValid() = %v, want %v", tt.strategy, got, tt.valid)
		}
	}
}

func TestEnsemblePreset_Validate(t *testing.T) {
	catalog := []ReasoningMode{
		{ID: "deductive", Name: "Deductive", Category: CategoryFormal, ShortDesc: "Test"},
		{ID: "bayesian", Name: "Bayesian", Category: CategoryUncertainty, ShortDesc: "Test"},
	}

	validPreset := EnsemblePreset{
		Name:              "test-preset",
		Description:       "A test preset",
		Modes:             []string{"deductive", "bayesian"},
		SynthesisStrategy: StrategyConsensus,
	}

	if err := validPreset.Validate(catalog); err != nil {
		t.Errorf("valid preset should pass validation: %v", err)
	}

	// Test missing name
	noName := validPreset
	noName.Name = ""
	if err := noName.Validate(catalog); err == nil {
		t.Error("preset without name should fail validation")
	}

	// Test no modes
	noModes := validPreset
	noModes.Modes = nil
	if err := noModes.Validate(catalog); err == nil {
		t.Error("preset without modes should fail validation")
	}

	// Test invalid strategy
	invalidStrategy := validPreset
	invalidStrategy.SynthesisStrategy = "invalid"
	if err := invalidStrategy.Validate(catalog); err == nil {
		t.Error("preset with invalid strategy should fail validation")
	}

	// Test missing mode
	missingMode := validPreset
	missingMode.Modes = []string{"deductive", "nonexistent"}
	if err := missingMode.Validate(catalog); err == nil {
		t.Error("preset referencing nonexistent mode should fail validation")
	}
}

func TestModeCatalog(t *testing.T) {
	modes := []ReasoningMode{
		{ID: "deductive", Name: "Deductive Logic", Category: CategoryFormal, ShortDesc: "Derive conclusions", BestFor: []string{"proofs"}},
		{ID: "bayesian", Name: "Bayesian Inference", Category: CategoryUncertainty, ShortDesc: "Probabilistic reasoning", BestFor: []string{"prediction"}},
		{ID: "causal-inference", Name: "Causal Inference", Category: CategoryCausal, ShortDesc: "Find causes", BestFor: []string{"debugging"}},
	}

	cat, err := NewModeCatalog(modes, "1.0.0")
	if err != nil {
		t.Fatalf("NewModeCatalog failed: %v", err)
	}

	// Test Count
	if got := cat.Count(); got != 3 {
		t.Errorf("Count() = %d, want 3", got)
	}

	// Test Version
	if got := cat.Version(); got != "1.0.0" {
		t.Errorf("Version() = %q, want %q", got, "1.0.0")
	}

	// Test GetMode
	mode := cat.GetMode("deductive")
	if mode == nil {
		t.Error("GetMode(deductive) returned nil")
	} else if mode.Name != "Deductive Logic" {
		t.Errorf("GetMode(deductive).Name = %q, want %q", mode.Name, "Deductive Logic")
	}

	// Test GetMode nonexistent
	if cat.GetMode("nonexistent") != nil {
		t.Error("GetMode(nonexistent) should return nil")
	}

	// Test ListModes
	all := cat.ListModes()
	if len(all) != 3 {
		t.Errorf("ListModes() returned %d modes, want 3", len(all))
	}

	// Test ListByCategory
	formal := cat.ListByCategory(CategoryFormal)
	if len(formal) != 1 {
		t.Errorf("ListByCategory(Formal) returned %d modes, want 1", len(formal))
	}

	// Test SearchModes
	found := cat.SearchModes("logic")
	if len(found) != 1 {
		t.Errorf("SearchModes(logic) returned %d modes, want 1", len(found))
	}

	// Test search in BestFor
	foundBestFor := cat.SearchModes("proofs")
	if len(foundBestFor) != 1 {
		t.Errorf("SearchModes(proofs) returned %d modes, want 1", len(foundBestFor))
	}
}

func TestModeCatalog_DuplicateID(t *testing.T) {
	modes := []ReasoningMode{
		{ID: "deductive", Name: "Deductive Logic", Category: CategoryFormal, ShortDesc: "Test"},
		{ID: "deductive", Name: "Duplicate", Category: CategoryFormal, ShortDesc: "Test"},
	}

	_, err := NewModeCatalog(modes, "1.0.0")
	if err == nil {
		t.Error("NewModeCatalog should fail with duplicate IDs")
	}
}

func TestModeCatalog_InvalidMode(t *testing.T) {
	modes := []ReasoningMode{
		{ID: "INVALID", Name: "Invalid", Category: CategoryFormal, ShortDesc: "Test"},
	}

	_, err := NewModeCatalog(modes, "1.0.0")
	if err == nil {
		t.Error("NewModeCatalog should fail with invalid mode")
	}
}

func TestModeAssignment_Fields(t *testing.T) {
	now := time.Now()
	completed := now.Add(time.Hour)

	assignment := ModeAssignment{
		ModeID:      "deductive",
		PaneName:    "myproject__cc_1",
		AgentType:   "cc",
		Status:      AssignmentActive,
		OutputPath:  "/tmp/output.txt",
		AssignedAt:  now,
		CompletedAt: &completed,
		Error:       "",
	}

	if assignment.ModeID != "deductive" {
		t.Errorf("ModeID = %q, want %q", assignment.ModeID, "deductive")
	}
	if assignment.Status != AssignmentActive {
		t.Errorf("Status = %q, want %q", assignment.Status, AssignmentActive)
	}
}

func TestEnsembleSession_Fields(t *testing.T) {
	now := time.Now()

	session := EnsembleSession{
		SessionName:       "myproject",
		Question:          "What is the best approach?",
		PresetUsed:        "architecture-review",
		Assignments:       []ModeAssignment{},
		Status:            EnsembleActive,
		SynthesisStrategy: StrategyConsensus,
		CreatedAt:         now,
	}

	if session.Status != EnsembleActive {
		t.Errorf("Status = %q, want %q", session.Status, EnsembleActive)
	}
	if session.SynthesisStrategy != StrategyConsensus {
		t.Errorf("SynthesisStrategy = %q, want %q", session.SynthesisStrategy, StrategyConsensus)
	}
}

func TestAllCategories(t *testing.T) {
	cats := AllCategories()
	if len(cats) != 12 {
		t.Errorf("AllCategories() returned %d categories, want 12", len(cats))
	}

	// All should be valid
	for _, cat := range cats {
		if !cat.IsValid() {
			t.Errorf("AllCategories() returned invalid category %q", cat)
		}
	}
}
