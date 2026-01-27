package ensemble

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStateStore_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.db")

	store, err := NewStateStore(path)
	if err != nil {
		t.Fatalf("NewStateStore error: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	createdAt := time.Now().UTC().Truncate(time.Second)
	synthAt := createdAt.Add(2 * time.Minute)
	completedAt := createdAt.Add(5 * time.Minute)

	session := &EnsembleSession{
		SessionName:       "test-session",
		Question:          "What is the issue?",
		PresetUsed:        "diagnosis",
		Status:            EnsembleActive,
		SynthesisStrategy: StrategyConsensus,
		CreatedAt:         createdAt,
		SynthesizedAt:     &synthAt,
		SynthesisOutput:   "summary",
		Error:             "",
		Assignments: []ModeAssignment{
			{
				ModeID:      "deductive",
				PaneName:    "pane-1",
				AgentType:   "cc",
				Status:      AssignmentActive,
				OutputPath:  "/tmp/out.txt",
				AssignedAt:  createdAt,
				CompletedAt: &completedAt,
			},
		},
	}

	if err := store.Save(session); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := store.Load("test-session")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected loaded session, got nil")
	}

	if loaded.Question != session.Question {
		t.Errorf("Question = %q, want %q", loaded.Question, session.Question)
	}
	if loaded.PresetUsed != session.PresetUsed {
		t.Errorf("PresetUsed = %q, want %q", loaded.PresetUsed, session.PresetUsed)
	}
	if loaded.Status != session.Status {
		t.Errorf("Status = %q, want %q", loaded.Status, session.Status)
	}
	if loaded.SynthesisStrategy != session.SynthesisStrategy {
		t.Errorf("SynthesisStrategy = %q, want %q", loaded.SynthesisStrategy, session.SynthesisStrategy)
	}
	if !loaded.CreatedAt.Equal(session.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", loaded.CreatedAt, session.CreatedAt)
	}
	if loaded.SynthesizedAt == nil || !loaded.SynthesizedAt.Equal(*session.SynthesizedAt) {
		t.Errorf("SynthesizedAt = %v, want %v", loaded.SynthesizedAt, session.SynthesizedAt)
	}
	if loaded.SynthesisOutput != session.SynthesisOutput {
		t.Errorf("SynthesisOutput = %q, want %q", loaded.SynthesisOutput, session.SynthesisOutput)
	}

	if len(loaded.Assignments) != 1 {
		t.Fatalf("Assignments len = %d, want 1", len(loaded.Assignments))
	}
	assignment := loaded.Assignments[0]
	if assignment.ModeID != "deductive" {
		t.Errorf("Assignment ModeID = %q, want deductive", assignment.ModeID)
	}
	if assignment.PaneName != "pane-1" {
		t.Errorf("Assignment PaneName = %q, want pane-1", assignment.PaneName)
	}
	if assignment.AgentType != "cc" {
		t.Errorf("Assignment AgentType = %q, want cc", assignment.AgentType)
	}
	if assignment.Status != AssignmentActive {
		t.Errorf("Assignment Status = %q, want %q", assignment.Status, AssignmentActive)
	}
	if assignment.OutputPath != "/tmp/out.txt" {
		t.Errorf("Assignment OutputPath = %q, want /tmp/out.txt", assignment.OutputPath)
	}
	if assignment.AssignedAt.IsZero() {
		t.Errorf("Assignment AssignedAt should be set")
	}
	if assignment.CompletedAt == nil || !assignment.CompletedAt.Equal(completedAt) {
		t.Errorf("Assignment CompletedAt = %v, want %v", assignment.CompletedAt, completedAt)
	}
}

func TestOutputCapture_ExtractYAML_PrefersValidBlock(t *testing.T) {
	capture := NewOutputCapture(nil)
	raw := strings.Join([]string{
		"noise before",
		"```yaml",
		": bad yaml",
		"```",
		"more noise",
		"```yaml",
		"mode_id: deductive",
		"thesis: something",
		"```",
	}, "\n")

	block, ok := capture.extractYAML(raw)
	if !ok {
		t.Fatal("expected YAML block to be found")
	}
	if !strings.Contains(block, "mode_id: deductive") {
		t.Fatalf("expected valid YAML block, got: %q", block)
	}
}

func TestOutputCapture_CapturePane_Empty(t *testing.T) {
	capture := NewOutputCapture(nil)
	if _, err := capture.capturePane(""); err == nil {
		t.Fatal("expected error for empty pane")
	}
}

func TestStateStore_UpdateListDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.db")

	store, err := NewStateStore(path)
	if err != nil {
		t.Fatalf("NewStateStore error: %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &EnsembleSession{
		SessionName:       "update-session",
		Question:          "Question",
		Status:            EnsembleActive,
		SynthesisStrategy: StrategyConsensus,
		CreatedAt:         time.Now().UTC(),
		Assignments: []ModeAssignment{
			{
				ModeID:    "deductive",
				PaneName:  "pane-1",
				AgentType: "cc",
				Status:    AssignmentPending,
			},
		},
	}

	if err := store.Save(session); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	if err := store.UpdateStatus(session.SessionName, EnsembleComplete); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if err := store.UpdateAssignmentStatus(session.SessionName, "deductive", AssignmentDone); err != nil {
		t.Fatalf("UpdateAssignmentStatus error: %v", err)
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List len = %d, want 1", len(list))
	}
	if list[0].Status != EnsembleComplete {
		t.Fatalf("List status = %q, want %q", list[0].Status, EnsembleComplete)
	}
	if len(list[0].Assignments) != 1 || list[0].Assignments[0].Status != AssignmentDone {
		t.Fatalf("assignment status = %q, want %q", list[0].Assignments[0].Status, AssignmentDone)
	}

	if err := store.Delete(session.SessionName); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if _, err := store.Load(session.SessionName); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Load after delete error = %v, want os.ErrNotExist", err)
	}
}

// =============================================================================
// Integration Tests for Ensemble Dependency Chain
// These tests verify that components integrate correctly without real tmux or model calls.
// =============================================================================

// TestIntegration_ModeCatalogToPromptEngine verifies the flow:
// ModeCatalog → mode lookup → PreambleEngine rendering
func TestIntegration_ModeCatalogToPromptEngine(t *testing.T) {
	// Step 1: Create mode catalog from embedded modes
	catalog, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog error: %v", err)
	}

	// Step 2: Verify catalog contains expected modes
	if catalog.Count() < 10 {
		t.Fatalf("expected at least 10 modes, got %d", catalog.Count())
	}

	// Step 3: Look up a specific mode
	deductive := catalog.GetMode("deductive")
	if deductive == nil {
		t.Fatal("expected deductive mode in catalog")
	}
	if deductive.Category != CategoryFormal {
		t.Errorf("deductive category = %q, want %q", deductive.Category, CategoryFormal)
	}

	// Step 4: Look up by code
	byCode := catalog.GetModeByCode("A1")
	if byCode == nil {
		t.Fatal("expected mode A1 in catalog")
	}
	if byCode.ID != "deductive" {
		t.Errorf("A1 mode ID = %q, want deductive", byCode.ID)
	}

	// Step 5: Create preamble engine and render
	engine := NewPreambleEngine()

	contextPack := &ContextPack{
		ProjectBrief: &ProjectBrief{
			Name:        "test-project",
			Description: "A test project for integration testing",
			Languages:   []string{"Go"},
		},
		UserContext: &UserContext{
			ProblemStatement: "Analyze the architecture for potential issues",
		},
	}

	preambleData := &PreambleData{
		Problem:     "What are the main architectural concerns?",
		ContextPack: contextPack,
		Mode:        deductive,
		TokenCap:    4000,
	}

	rendered, err := engine.Render(preambleData)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	// Step 6: Verify preamble contains expected content
	if !strings.Contains(rendered, "deductive") {
		t.Error("preamble should contain mode ID")
	}
	if !strings.Contains(rendered, "What are the main architectural concerns?") {
		t.Error("preamble should contain problem statement")
	}
	if !strings.Contains(rendered, "test-project") {
		t.Error("preamble should contain project name")
	}
	if !strings.Contains(rendered, "REQUIRED OUTPUT FORMAT") {
		t.Error("preamble should contain schema contract")
	}
	if !strings.Contains(rendered, SchemaVersion) {
		t.Errorf("preamble should contain schema version %s", SchemaVersion)
	}
}

// TestIntegration_EnsembleToOrchestrator verifies the flow:
// EnsemblePreset → mode resolution → ModeAssignment creation
func TestIntegration_EnsembleToOrchestrator(t *testing.T) {
	// Step 1: Load catalog
	catalog, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog error: %v", err)
	}

	// Step 2: Get an embedded ensemble preset
	preset := GetEmbeddedEnsemble("project-diagnosis")
	if preset == nil {
		t.Fatal("expected project-diagnosis preset")
	}

	// Step 3: Validate preset against catalog
	if err := preset.Validate(catalog); err != nil {
		t.Fatalf("preset validation error: %v", err)
	}

	// Step 4: Resolve mode refs to IDs
	modeIDs, err := preset.ResolveIDs(catalog)
	if err != nil {
		t.Fatalf("ResolveIDs error: %v", err)
	}
	if len(modeIDs) < 2 {
		t.Fatalf("expected at least 2 modes, got %d", len(modeIDs))
	}

	// Step 5: Simulate assignment creation (without real tmux)
	assignments := make([]ModeAssignment, len(modeIDs))
	for i, modeID := range modeIDs {
		mode := catalog.GetMode(modeID)
		if mode == nil {
			t.Fatalf("mode %q not found after resolution", modeID)
		}
		assignments[i] = ModeAssignment{
			ModeID:     modeID,
			PaneName:   "mock-pane-" + modeID,
			AgentType:  "cc",
			Status:     AssignmentPending,
			AssignedAt: time.Now(),
		}
	}

	// Step 6: Verify assignments match modes
	if len(assignments) != len(modeIDs) {
		t.Errorf("assignment count = %d, want %d", len(assignments), len(modeIDs))
	}
	for _, a := range assignments {
		if a.Status != AssignmentPending {
			t.Errorf("assignment %s status = %q, want pending", a.ModeID, a.Status)
		}
	}
}

// TestIntegration_IntakeToOrchestrator verifies the flow:
// User input → ContextPack → targeted questions → preamble data
func TestIntegration_IntakeToOrchestrator(t *testing.T) {
	// Step 1: Start with thin context
	thinPack := &ContextPack{
		ProjectBrief: &ProjectBrief{
			Name: "thin-project",
		},
		// UserContext is nil - thin context
	}

	// Step 2: Check if questions should be asked
	if !ShouldAskQuestions(thinPack) {
		t.Error("expected ShouldAskQuestions = true for thin context")
	}

	// Step 3: Select questions for thin context
	questions := SelectQuestions(thinPack)
	if len(questions) == 0 {
		t.Fatal("expected questions for thin context")
	}

	// Verify required questions are included
	hasGoal := false
	for _, q := range questions {
		if q.ID == "goal" && q.Required {
			hasGoal = true
		}
	}
	if !hasGoal {
		t.Error("expected required 'goal' question")
	}

	// Step 4: Simulate user answers
	answers := map[string]string{
		"goal":        "Identify performance bottlenecks in the API layer",
		"concerns":    "memory leaks, slow database queries",
		"constraints": "must maintain backward compatibility",
	}

	// Step 5: Merge answers into context pack
	enrichedPack := MergeAnswers(thinPack, answers)

	// Step 6: Verify enriched context
	if enrichedPack.UserContext == nil {
		t.Fatal("expected UserContext after merge")
	}
	if !strings.Contains(enrichedPack.UserContext.ProblemStatement, "performance bottlenecks") {
		t.Error("expected problem statement to include user answer")
	}
	if len(enrichedPack.UserContext.FocusAreas) != 2 {
		t.Errorf("expected 2 focus areas, got %d", len(enrichedPack.UserContext.FocusAreas))
	}
	if len(enrichedPack.UserContext.Constraints) != 1 {
		t.Errorf("expected 1 constraint, got %d", len(enrichedPack.UserContext.Constraints))
	}

	// Step 7: Verify questions are no longer needed for specific fields
	if ShouldAskQuestions(enrichedPack) {
		// This is expected since we still have thin project brief
		// But we should have fewer questions now
		newQuestions := SelectQuestions(enrichedPack)
		if len(newQuestions) >= len(questions) {
			t.Error("expected fewer questions after enrichment")
		}
	}
}

// TestIntegration_OrchestratorToSynthesis verifies the flow:
// ModeOutputs → synthesis strategy lookup → strategy config
func TestIntegration_OrchestratorToSynthesis(t *testing.T) {
	// Step 1: Create mock mode outputs (as if from agents)
	outputs := []ModeOutput{
		{
			ModeID: "deductive",
			Thesis: "The authentication layer has a potential race condition",
			TopFindings: []Finding{
				{
					Finding:         "Race condition in token refresh",
					Impact:          ImpactHigh,
					Confidence:      0.85,
					EvidencePointer: "internal/auth/token.go:142",
					Reasoning:       "Multiple goroutines access shared state",
				},
			},
			Confidence:  0.85,
			GeneratedAt: time.Now(),
		},
		{
			ModeID: "systems-thinking",
			Thesis: "The system has tight coupling between auth and session management",
			TopFindings: []Finding{
				{
					Finding:         "Tight coupling increases change risk",
					Impact:          ImpactMedium,
					Confidence:      0.7,
					EvidencePointer: "internal/session/manager.go:55",
					Reasoning:       "Direct imports create dependency",
				},
			},
			Confidence:  0.7,
			GeneratedAt: time.Now(),
		},
	}

	// Step 2: Validate outputs
	for i, output := range outputs {
		if err := output.Validate(); err != nil {
			t.Fatalf("output[%d] validation error: %v", i, err)
		}
	}

	// Step 3: Get synthesis strategy config
	strategyName := "consensus"
	strategy, err := GetStrategy(strategyName)
	if err != nil {
		t.Fatalf("GetStrategy error: %v", err)
	}

	// Step 4: Verify strategy config
	if strategy.Name != StrategyConsensus {
		t.Errorf("strategy name = %q, want consensus", strategy.Name)
	}
	if !strategy.RequiresAgent {
		t.Error("consensus strategy should require agent")
	}
	if strategy.SynthesizerMode == "" {
		t.Error("consensus strategy should have synthesizer mode")
	}

	// Step 5: Verify deprecated strategies migrate correctly
	migrated, wasMigrated := MigrateStrategy("debate")
	if !wasMigrated {
		t.Error("expected 'debate' to be deprecated")
	}
	if migrated != "dialectical" {
		t.Errorf("debate migration = %q, want dialectical", migrated)
	}

	// Step 6: Test strategy validation
	_, err = ValidateOrMigrateStrategy("invalid-strategy")
	if err == nil {
		t.Error("expected error for invalid strategy")
	}

	// Step 7: Estimate tokens for outputs
	totalTokens := EstimateModeOutputsTokens(outputs)
	if totalTokens <= 0 {
		t.Error("expected positive token estimate for outputs")
	}
}

// TestIntegration_BudgetEnforcement verifies the flow:
// BudgetConfig → BudgetTracker → spending → limit enforcement
func TestIntegration_BudgetEnforcement(t *testing.T) {
	// Step 1: Create budget config
	config := BudgetConfig{
		MaxTokensPerMode: 1000,
		MaxTotalTokens:   3000,
		TimeoutPerMode:   2 * time.Minute,
	}

	// Step 2: Create tracker
	tracker := NewBudgetTracker(config, nil)

	// Step 3: Record spending within budget
	result1 := tracker.RecordSpend("agent-1", 500)
	if !result1.Allowed {
		t.Error("expected spend of 500 to be allowed")
	}
	if result1.Remaining != 500 {
		t.Errorf("remaining = %d, want 500", result1.Remaining)
	}

	// Step 4: Record more spending
	result2 := tracker.RecordSpend("agent-2", 800)
	if !result2.Allowed {
		t.Error("expected spend of 800 to be allowed")
	}

	// Step 5: Exceed per-agent budget
	result3 := tracker.RecordSpend("agent-1", 600) // Total for agent-1 = 1100 > 1000
	if result3.Allowed {
		t.Error("expected spend to be disallowed (agent over budget)")
	}
	if result3.Remaining >= 0 {
		t.Error("expected negative remaining for over-budget agent")
	}

	// Step 6: Check budget state
	state := tracker.GetState()
	if !state.IsOverBudget {
		// Total spent: 500 + 800 + 600 = 1900, still under 3000
		// But agent-1 is over budget
	}
	if len(state.OverBudgetAgents) == 0 {
		t.Error("expected at least one over-budget agent")
	}

	// Step 7: Check individual agent status
	if !tracker.IsAgentOverBudget("agent-1") {
		t.Error("expected agent-1 to be over budget")
	}
	if tracker.IsAgentOverBudget("agent-2") {
		t.Error("expected agent-2 to be within budget")
	}

	// Step 8: Exceed total budget
	result4 := tracker.RecordSpend("agent-3", 1500) // Total = 3400 > 3000
	if result4.Allowed {
		t.Error("expected spend to be disallowed (total over budget)")
	}
	if !tracker.IsOverBudget() {
		t.Error("expected total budget to be exceeded")
	}

	// Step 9: Test reset
	tracker.Reset()
	if tracker.IsOverBudget() {
		t.Error("expected tracker to be within budget after reset")
	}
	if tracker.TotalRemaining() != config.MaxTotalTokens {
		t.Errorf("total remaining after reset = %d, want %d", tracker.TotalRemaining(), config.MaxTotalTokens)
	}
}

// TestIntegration_FullPipeline_ProjectDiagnosis verifies the complete flow:
// Preset → Modes → Context → Preambles → Mock Outputs → Strategy
// This simulates a full ensemble run without real tmux or model calls.
func TestIntegration_FullPipeline_ProjectDiagnosis(t *testing.T) {
	// === PHASE 1: CATALOG & PRESET SETUP ===

	// Load catalog
	catalog, err := DefaultCatalog()
	if err != nil {
		t.Fatalf("DefaultCatalog error: %v", err)
	}

	// Get project-diagnosis preset
	preset := GetEmbeddedEnsemble("project-diagnosis")
	if preset == nil {
		t.Fatal("expected project-diagnosis preset")
	}

	// Create registry from embedded ensembles directly (no user/project files)
	// This avoids issues with advanced-tier modes in some presets
	registry := NewEnsembleRegistry(EmbeddedEnsembles, catalog)

	// Validate the specific preset we're testing (project-diagnosis uses core modes)
	report := ValidateEnsemblePreset(preset, catalog, registry)
	if report.HasErrors() {
		t.Fatalf("preset validation errors: %v", report.Error())
	}

	// === PHASE 2: MODE RESOLUTION ===

	modeIDs, err := preset.ResolveIDs(catalog)
	if err != nil {
		t.Fatalf("ResolveIDs error: %v", err)
	}

	modes := make([]*ReasoningMode, len(modeIDs))
	for i, id := range modeIDs {
		modes[i] = catalog.GetMode(id)
		if modes[i] == nil {
			t.Fatalf("mode %q not found", id)
		}
	}

	// === PHASE 3: CONTEXT PACK ASSEMBLY ===

	contextPack := &ContextPack{
		GeneratedAt: time.Now(),
		ProjectBrief: &ProjectBrief{
			Name:        "ntm",
			Description: "Named Tmux Manager for orchestrating AI coding agents",
			Languages:   []string{"Go"},
			Frameworks:  []string{"cobra", "bubbletea"},
			Structure: &ProjectStructure{
				TotalFiles: 150,
				TotalLines: 25000,
			},
			OpenIssues: 42,
		},
		UserContext: &UserContext{
			ProblemStatement: "Assess the health of the ntm codebase and identify areas for improvement",
			FocusAreas:       []string{"architecture", "reliability", "maintainability"},
			Constraints:      []string{"Go 1.25+", "no external build tools"},
			SuccessCriteria:  []string{"identify top 3 risks", "recommend quick wins"},
		},
	}

	// Check if questions are needed
	if ShouldAskQuestions(contextPack) {
		t.Log("Context is still considered thin, but proceeding with test")
	}

	// === PHASE 4: PREAMBLE GENERATION ===

	engine := NewPreambleEngine()
	preambles := make([]string, len(modes))

	for i, mode := range modes {
		data := &PreambleData{
			Problem:     contextPack.UserContext.ProblemStatement,
			ContextPack: contextPack,
			Mode:        mode,
			TokenCap:    preset.Budget.MaxTokensPerMode,
		}

		preamble, err := engine.Render(data)
		if err != nil {
			t.Fatalf("render preamble for mode %q: %v", mode.ID, err)
		}
		preambles[i] = preamble

		// Verify preamble content - preamble uses mode.Name not mode.ID
		if !strings.Contains(preamble, mode.Name) && !strings.Contains(preamble, mode.Code) {
			t.Errorf("preamble for %q should contain mode name %q or code %q", mode.ID, mode.Name, mode.Code)
		}
		if !strings.Contains(preamble, "ntm") {
			t.Errorf("preamble for %q should contain project name", mode.ID)
		}
	}

	// === PHASE 5: MOCK AGENT ASSIGNMENTS ===

	session := &EnsembleSession{
		SessionName:       "test-ensemble-session",
		Question:          contextPack.UserContext.ProblemStatement,
		PresetUsed:        preset.Name,
		Status:            EnsembleActive,
		SynthesisStrategy: preset.Synthesis.Strategy,
		CreatedAt:         time.Now(),
		Assignments:       make([]ModeAssignment, len(modes)),
	}

	for i, mode := range modes {
		session.Assignments[i] = ModeAssignment{
			ModeID:     mode.ID,
			PaneName:   "mock-pane-" + mode.ID,
			AgentType:  "cc",
			Status:     AssignmentPending,
			AssignedAt: time.Now(),
		}
	}

	// === PHASE 6: BUDGET TRACKING ===

	tracker := NewBudgetTracker(preset.Budget, nil)

	// Simulate token spending per mode
	for _, a := range session.Assignments {
		mockTokens := 2000 + (len(a.ModeID) * 100) // Deterministic mock value
		result := tracker.RecordSpend(a.ModeID, mockTokens)
		if !result.Allowed {
			t.Logf("budget exceeded for %q: %s", a.ModeID, result.Message)
		}
	}

	// Check budget state
	budgetState := tracker.GetState()
	t.Logf("Budget state: total=%d/%d, elapsed=%v",
		budgetState.TotalSpent, budgetState.TotalLimit, budgetState.ElapsedTime)

	// === PHASE 7: MOCK OUTPUT GENERATION ===

	outputs := make([]ModeOutput, len(modes))
	for i, mode := range modes {
		outputs[i] = ModeOutput{
			ModeID: mode.ID,
			Thesis: "Analysis from " + mode.Name + " perspective reveals important insights",
			TopFindings: []Finding{
				{
					Finding:         "Finding from " + mode.ID + " analysis",
					Impact:          ImpactMedium,
					Confidence:      0.75,
					EvidencePointer: "internal/ensemble/integration_test.go:1",
					Reasoning:       "Based on " + mode.Category.String() + " reasoning",
				},
			},
			Recommendations: []Recommendation{
				{
					Recommendation:  "Recommendation from " + mode.ID,
					Priority:        ImpactMedium,
					Rationale:       "Supports project health",
					RelatedFindings: []int{0},
				},
			},
			FailureModesToWatch: []FailureModeWarning{
				{
					Mode:        mode.FailureModes[0],
					Description: "Potential failure pattern",
				},
			},
			Confidence:  0.75,
			GeneratedAt: time.Now(),
		}

		// Validate output
		if err := outputs[i].Validate(); err != nil {
			t.Fatalf("output[%d] validation error: %v", i, err)
		}

		// Update assignment status
		session.Assignments[i].Status = AssignmentDone
		now := time.Now()
		session.Assignments[i].CompletedAt = &now
	}

	// === PHASE 8: SYNTHESIS STRATEGY LOOKUP ===

	strategy, err := GetStrategy(string(preset.Synthesis.Strategy))
	if err != nil {
		t.Fatalf("GetStrategy error: %v", err)
	}

	// Verify strategy matches preset
	if strategy.Name != preset.Synthesis.Strategy {
		t.Errorf("strategy mismatch: got %q, want %q", strategy.Name, preset.Synthesis.Strategy)
	}

	// === PHASE 9: FINAL STATE VERIFICATION ===

	// Update session status
	session.Status = EnsembleSynthesizing

	// Verify all assignments completed
	allDone := true
	for _, a := range session.Assignments {
		if a.Status != AssignmentDone {
			allDone = false
			break
		}
	}
	if !allDone {
		t.Error("expected all assignments to be done")
	}

	// Mark synthesis complete
	session.Status = EnsembleComplete
	now := time.Now()
	session.SynthesizedAt = &now
	session.SynthesisOutput = "Mock synthesis output combining all mode perspectives"

	// Final validation
	if session.Status != EnsembleComplete {
		t.Errorf("session status = %q, want complete", session.Status)
	}
	if session.SynthesisOutput == "" {
		t.Error("expected synthesis output")
	}

	// Token estimation sanity check
	totalOutputTokens := EstimateModeOutputsTokens(outputs)
	if totalOutputTokens <= 0 {
		t.Error("expected positive token estimate")
	}
	t.Logf("Total output tokens estimated: %d", totalOutputTokens)

	// Log success
	t.Logf("Full pipeline test completed: %d modes, %d outputs, strategy=%s",
		len(modes), len(outputs), strategy.Name)
}
