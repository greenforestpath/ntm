package coordinator

import (
	"strings"
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/bv"
	"github.com/Dicklesworthstone/ntm/internal/persona"
	"github.com/Dicklesworthstone/ntm/internal/robot"
)

func TestWorkAssignmentStruct(t *testing.T) {
	now := time.Now()
	wa := WorkAssignment{
		BeadID:         "ntm-1234",
		BeadTitle:      "Implement feature X",
		AgentPaneID:    "%0",
		AgentMailName:  "BlueFox",
		AgentType:      "cc",
		AssignedAt:     now,
		Priority:       1,
		Score:          0.85,
		FilesToReserve: []string{"internal/feature/*.go"},
	}

	if wa.BeadID != "ntm-1234" {
		t.Errorf("expected BeadID 'ntm-1234', got %q", wa.BeadID)
	}
	if wa.Score != 0.85 {
		t.Errorf("expected Score 0.85, got %f", wa.Score)
	}
	if len(wa.FilesToReserve) != 1 {
		t.Errorf("expected 1 file to reserve, got %d", len(wa.FilesToReserve))
	}
}

func TestAssignmentResultStruct(t *testing.T) {
	ar := AssignmentResult{
		Success:      true,
		MessageSent:  true,
		Reservations: []string{"internal/*.go"},
	}

	if !ar.Success {
		t.Error("expected Success to be true")
	}
	if !ar.MessageSent {
		t.Error("expected MessageSent to be true")
	}
	if ar.Error != "" {
		t.Error("expected empty error on success")
	}
}

func TestRemoveRecommendation(t *testing.T) {
	recs := []bv.TriageRecommendation{
		{ID: "ntm-001", Title: "First"},
		{ID: "ntm-002", Title: "Second"},
		{ID: "ntm-003", Title: "Third"},
	}

	result := removeRecommendation(recs, "ntm-002")

	if len(result) != 2 {
		t.Errorf("expected 2 recommendations after removal, got %d", len(result))
	}
	for _, r := range result {
		if r.ID == "ntm-002" {
			t.Error("expected ntm-002 to be removed")
		}
	}

	// Test removing non-existent ID
	result2 := removeRecommendation(recs, "ntm-999")
	if len(result2) != 3 {
		t.Errorf("expected 3 recommendations when removing non-existent, got %d", len(result2))
	}

	// Test empty slice (should not panic)
	result3 := removeRecommendation(nil, "ntm-001")
	if result3 != nil {
		t.Errorf("expected nil for empty input, got %v", result3)
	}

	result4 := removeRecommendation([]bv.TriageRecommendation{}, "ntm-001")
	if result4 != nil {
		t.Errorf("expected nil for empty slice, got %v", result4)
	}
}

func TestFindBestMatch(t *testing.T) {
	c := New("test-session", "/tmp/test", nil, "TestAgent")

	agent := &AgentState{
		PaneID:        "%0",
		AgentType:     "cc",
		AgentMailName: "BlueFox",
		Status:        robot.StateWaiting,
		Healthy:       true,
	}

	recs := []bv.TriageRecommendation{
		{ID: "ntm-001", Title: "Blocked Task", Status: "blocked", Score: 0.9},
		{ID: "ntm-002", Title: "Ready Task", Status: "open", Priority: 1, Score: 0.8},
		{ID: "ntm-003", Title: "Another Ready", Status: "open", Priority: 2, Score: 0.7},
	}

	assignment, rec := c.findBestMatch(agent, recs)

	if assignment == nil {
		t.Fatal("expected assignment, got nil")
	}
	if rec == nil {
		t.Fatal("expected recommendation, got nil")
	}
	if assignment.BeadID != "ntm-002" {
		t.Errorf("expected BeadID 'ntm-002' (first non-blocked), got %q", assignment.BeadID)
	}
	if assignment.AgentMailName != "BlueFox" {
		t.Errorf("expected AgentMailName 'BlueFox', got %q", assignment.AgentMailName)
	}
}

func TestFindBestMatchAllBlocked(t *testing.T) {
	c := New("test-session", "/tmp/test", nil, "TestAgent")

	agent := &AgentState{
		PaneID:    "%0",
		AgentType: "cc",
	}

	recs := []bv.TriageRecommendation{
		{ID: "ntm-001", Title: "Blocked 1", Status: "blocked"},
		{ID: "ntm-002", Title: "Blocked 2", Status: "blocked"},
	}

	assignment, rec := c.findBestMatch(agent, recs)

	if assignment != nil {
		t.Error("expected nil assignment when all are blocked")
	}
	if rec != nil {
		t.Error("expected nil recommendation when all are blocked")
	}
}

func TestFindBestMatchEmpty(t *testing.T) {
	c := New("test-session", "/tmp/test", nil, "TestAgent")

	agent := &AgentState{
		PaneID:    "%0",
		AgentType: "cc",
	}

	assignment, rec := c.findBestMatch(agent, nil)

	if assignment != nil || rec != nil {
		t.Error("expected nil for empty recommendations")
	}

	assignment, rec = c.findBestMatch(agent, []bv.TriageRecommendation{})

	if assignment != nil || rec != nil {
		t.Error("expected nil for empty slice")
	}
}

func TestFormatAssignmentMessage(t *testing.T) {
	c := New("test-session", "/tmp/test", nil, "TestAgent")

	assignment := &WorkAssignment{
		BeadID:    "ntm-1234",
		BeadTitle: "Implement feature X",
		Priority:  1,
		Score:     0.85,
	}

	rec := &bv.TriageRecommendation{
		ID:          "ntm-1234",
		Title:       "Implement feature X",
		Reasons:     []string{"High impact", "Unblocks others"},
		UnblocksIDs: []string{"ntm-2000", "ntm-2001"},
	}

	body := c.formatAssignmentMessage(assignment, rec)

	if body == "" {
		t.Error("expected non-empty message body")
	}
	if !strings.Contains(body, "# Work Assignment") {
		t.Error("expected markdown header in message")
	}
	if !strings.Contains(body, "ntm-1234") {
		t.Error("expected bead ID in message")
	}
	if !strings.Contains(body, "High impact") {
		t.Error("expected reasons in message")
	}
	if !strings.Contains(body, "bd show") {
		t.Error("expected bd show instruction in message")
	}
}

func TestDefaultScoreConfig(t *testing.T) {
	config := DefaultScoreConfig()

	if !config.PreferCriticalPath {
		t.Error("expected PreferCriticalPath to be true by default")
	}
	if !config.PenalizeFileOverlap {
		t.Error("expected PenalizeFileOverlap to be true by default")
	}
	if !config.UseAgentProfiles {
		t.Error("expected UseAgentProfiles to be true by default")
	}
	if !config.BudgetAware {
		t.Error("expected BudgetAware to be true by default")
	}
	if config.ContextThreshold != 80 {
		t.Errorf("expected ContextThreshold 80, got %f", config.ContextThreshold)
	}
}

func TestEstimateTaskComplexity(t *testing.T) {
	tests := []struct {
		name     string
		rec      *bv.TriageRecommendation
		expected float64
		minExp   float64
		maxExp   float64
	}{
		{
			name:   "epic is complex",
			rec:    &bv.TriageRecommendation{Type: "epic", Priority: 2},
			minExp: 0.7,
			maxExp: 1.0,
		},
		{
			name:   "chore is simple",
			rec:    &bv.TriageRecommendation{Type: "chore", Priority: 2},
			minExp: 0.0,
			maxExp: 0.4,
		},
		{
			name:   "feature is moderately complex",
			rec:    &bv.TriageRecommendation{Type: "feature", Priority: 2},
			minExp: 0.6,
			maxExp: 0.8,
		},
		{
			name:   "epic with many unblocks is very complex",
			rec:    &bv.TriageRecommendation{Type: "epic", Priority: 2, UnblocksIDs: []string{"a", "b", "c", "d", "e"}},
			minExp: 0.9,
			maxExp: 1.0,
		},
		{
			name:   "critical bug is simpler",
			rec:    &bv.TriageRecommendation{Type: "bug", Priority: 0},
			minExp: 0.3,
			maxExp: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complexity := estimateTaskComplexity(tt.rec)
			if complexity < tt.minExp || complexity > tt.maxExp {
				t.Errorf("expected complexity in [%f, %f], got %f", tt.minExp, tt.maxExp, complexity)
			}
		})
	}
}

func TestComputeAgentTypeBonus(t *testing.T) {
	tests := []struct {
		name      string
		agentType string
		rec       *bv.TriageRecommendation
		wantSign  string // "positive", "negative", "zero"
	}{
		{
			name:      "claude on epic gets bonus",
			agentType: "cc",
			rec:       &bv.TriageRecommendation{Type: "epic", Priority: 2},
			wantSign:  "positive",
		},
		{
			name:      "claude on chore gets penalty",
			agentType: "claude",
			rec:       &bv.TriageRecommendation{Type: "chore", Priority: 2},
			wantSign:  "negative",
		},
		{
			name:      "codex on chore gets bonus",
			agentType: "cod",
			rec:       &bv.TriageRecommendation{Type: "chore", Priority: 2},
			wantSign:  "positive",
		},
		{
			name:      "codex on epic gets penalty",
			agentType: "codex",
			rec:       &bv.TriageRecommendation{Type: "epic", Priority: 2},
			wantSign:  "negative",
		},
		{
			name:      "gemini on medium task neutral or small bonus",
			agentType: "gmi",
			rec:       &bv.TriageRecommendation{Type: "task", Priority: 2},
			wantSign:  "zero", // task is medium complexity
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bonus := computeAgentTypeBonus(tt.agentType, tt.rec)
			switch tt.wantSign {
			case "positive":
				if bonus <= 0 {
					t.Errorf("expected positive bonus, got %f", bonus)
				}
			case "negative":
				if bonus >= 0 {
					t.Errorf("expected negative bonus, got %f", bonus)
				}
			case "zero":
				if bonus < -0.05 || bonus > 0.1 {
					t.Errorf("expected near-zero bonus, got %f", bonus)
				}
			}
		})
	}
}

func TestComputeContextPenalty(t *testing.T) {
	tests := []struct {
		name         string
		contextUsage float64
		threshold    float64
		wantZero     bool
	}{
		{"below threshold", 50, 80, true},
		{"at threshold", 80, 80, true},
		{"above threshold", 90, 80, false},
		{"way above threshold", 95, 80, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			penalty := computeContextPenalty(tt.contextUsage, tt.threshold)
			if tt.wantZero && penalty != 0 {
				t.Errorf("expected zero penalty, got %f", penalty)
			}
			if !tt.wantZero && penalty <= 0 {
				t.Errorf("expected positive penalty, got %f", penalty)
			}
		})
	}

	// Verify penalty values are reasonable (normalized to 0-1 scale)
	t.Run("penalty values are reasonable", func(t *testing.T) {
		// 10% over threshold (90% usage, 80% threshold)
		penalty10 := computeContextPenalty(90, 80)
		if penalty10 < 0.04 || penalty10 > 0.06 {
			t.Errorf("10%% over threshold should give ~0.05 penalty, got %f", penalty10)
		}

		// 20% over threshold (100% usage, 80% threshold)
		penalty20 := computeContextPenalty(100, 80)
		if penalty20 < 0.09 || penalty20 > 0.11 {
			t.Errorf("20%% over threshold should give ~0.10 penalty, got %f", penalty20)
		}
	})
}

func TestComputeFileOverlapPenalty(t *testing.T) {
	tests := []struct {
		name         string
		agent        *AgentState
		reservations map[string][]string
		wantZero     bool
	}{
		{
			name:         "no reservations",
			agent:        &AgentState{PaneID: "%0"},
			reservations: nil,
			wantZero:     true,
		},
		{
			name:         "agent with reservations",
			agent:        &AgentState{PaneID: "%0", Reservations: []string{"a.go", "b.go", "c.go"}},
			reservations: nil,
			wantZero:     false,
		},
		{
			name:         "reservations in map",
			agent:        &AgentState{PaneID: "%0"},
			reservations: map[string][]string{"%0": {"x.go"}},
			wantZero:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			penalty := computeFileOverlapPenalty(tt.agent, tt.reservations)
			if tt.wantZero && penalty != 0 {
				t.Errorf("expected zero penalty, got %f", penalty)
			}
			if !tt.wantZero && penalty <= 0 {
				t.Errorf("expected positive penalty, got %f", penalty)
			}
		})
	}
}

func TestScoreAndSelectAssignmentsWithAgentReservations(t *testing.T) {
	// Test that agent.Reservations are used when existingReservations map is nil
	agents := []*AgentState{
		{PaneID: "%1", AgentType: "cc", ContextUsage: 30, Status: robot.StateWaiting},
		{PaneID: "%2", AgentType: "cc", ContextUsage: 30, Status: robot.StateWaiting, Reservations: []string{"a.go", "b.go", "c.go"}},
	}

	triage := &bv.TriageResponse{
		Triage: bv.TriageData{
			Recommendations: []bv.TriageRecommendation{
				{ID: "ntm-001", Title: "Task", Type: "task", Status: "open", Priority: 2, Score: 0.5},
			},
		},
	}

	config := DefaultScoreConfig()
	results := ScoreAndSelectAssignments(agents, triage, config, nil) // nil reservations map

	if len(results) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(results))
	}

	// Agent %1 should get the task because %2 has reservations (penalty)
	if results[0].Agent.PaneID != "%1" {
		t.Errorf("expected agent %%1 (no reservations) to get task, got %s", results[0].Agent.PaneID)
	}

	// Verify the penalty was applied to agent %2
	// Score for %1: 0.5 base (no penalty)
	// Score for %2: 0.5 - 0.05 = 0.45 (with reservation penalty for 3 files)
	if results[0].ScoreBreakdown.FileOverlapPenalty != 0 {
		t.Errorf("expected no file overlap penalty for agent %%1, got %f", results[0].ScoreBreakdown.FileOverlapPenalty)
	}
}

func TestScoreAndSelectAssignments(t *testing.T) {
	agents := []*AgentState{
		{PaneID: "%1", AgentType: "cc", ContextUsage: 30, Status: robot.StateWaiting},
		{PaneID: "%2", AgentType: "cod", ContextUsage: 50, Status: robot.StateWaiting},
	}

	triage := &bv.TriageResponse{
		Triage: bv.TriageData{
			Recommendations: []bv.TriageRecommendation{
				{ID: "ntm-001", Title: "Epic task", Type: "epic", Status: "open", Priority: 2, Score: 0.8},
				{ID: "ntm-002", Title: "Quick fix", Type: "chore", Status: "open", Priority: 2, Score: 0.6},
				{ID: "ntm-003", Title: "Blocked", Type: "task", Status: "blocked", Priority: 2, Score: 0.9},
			},
		},
	}

	config := DefaultScoreConfig()
	results := ScoreAndSelectAssignments(agents, triage, config, nil)

	if len(results) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(results))
	}

	// Verify each agent got exactly one task
	agentTasks := make(map[string]string)
	for _, r := range results {
		if existing, ok := agentTasks[r.Agent.PaneID]; ok {
			t.Errorf("agent %s assigned twice: %s and %s", r.Agent.PaneID, existing, r.Assignment.BeadID)
		}
		agentTasks[r.Agent.PaneID] = r.Assignment.BeadID
	}

	// Verify blocked task not assigned
	for _, r := range results {
		if r.Assignment.BeadID == "ntm-003" {
			t.Error("blocked task should not be assigned")
		}
	}
}

func TestScoreAndSelectAssignmentsEmpty(t *testing.T) {
	// Empty agents
	result := ScoreAndSelectAssignments(nil, &bv.TriageResponse{}, DefaultScoreConfig(), nil)
	if result != nil {
		t.Error("expected nil for empty agents")
	}

	// Empty triage
	agents := []*AgentState{{PaneID: "%0", AgentType: "cc"}}
	result = ScoreAndSelectAssignments(agents, nil, DefaultScoreConfig(), nil)
	if result != nil {
		t.Error("expected nil for nil triage")
	}

	// Empty recommendations
	result = ScoreAndSelectAssignments(agents, &bv.TriageResponse{}, DefaultScoreConfig(), nil)
	if result != nil {
		t.Error("expected nil for empty recommendations")
	}
}

func TestComputeCriticalPathBonus(t *testing.T) {
	tests := []struct {
		name      string
		breakdown *bv.ScoreBreakdown
		wantZero  bool
	}{
		{
			name:      "low pagerank",
			breakdown: &bv.ScoreBreakdown{Pagerank: 0.01, BlockerRatio: 0.01},
			wantZero:  true,
		},
		{
			name:      "high pagerank",
			breakdown: &bv.ScoreBreakdown{Pagerank: 0.1, BlockerRatio: 0.01},
			wantZero:  false,
		},
		{
			name:      "high blocker ratio",
			breakdown: &bv.ScoreBreakdown{Pagerank: 0.01, BlockerRatio: 0.1},
			wantZero:  false,
		},
		{
			name:      "high time to impact",
			breakdown: &bv.ScoreBreakdown{Pagerank: 0.01, BlockerRatio: 0.01, TimeToImpact: 0.06},
			wantZero:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bonus := computeCriticalPathBonus(tt.breakdown)
			if tt.wantZero && bonus != 0 {
				t.Errorf("expected zero bonus, got %f", bonus)
			}
			if !tt.wantZero && bonus <= 0 {
				t.Errorf("expected positive bonus, got %f", bonus)
			}
		})
	}
}

func TestExtractTaskTags(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
		wantTags    []string
	}{
		{
			name:     "test-related title",
			title:    "Add unit tests for the parser",
			wantTags: []string{"testing", "implementation"}, // "add" triggers implementation
		},
		{
			name:     "architecture task",
			title:    "Refactor the API design patterns",
			wantTags: []string{"architecture"},
		},
		{
			name:     "documentation task",
			title:    "Update README with new features",
			wantTags: []string{"documentation", "implementation"}, // "feature" triggers implementation
		},
		{
			name:     "implementation task",
			title:    "Implement new feature for user auth",
			wantTags: []string{"implementation"},
		},
		{
			name:     "review task",
			title:    "Code review for PR #123",
			wantTags: []string{"review"},
		},
		{
			name:     "bug fix",
			title:    "Fix crash when loading config",
			wantTags: []string{"bugs"},
		},
		{
			name:        "multiple tags from description",
			title:       "Add tests",
			description: "Refactor the code and add documentation",
			wantTags:    []string{"testing", "architecture", "documentation", "implementation"}, // "add" triggers implementation
		},
		{
			name:     "no matching tags",
			title:    "Random task with no keywords",
			wantTags: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := ExtractTaskTags(tt.title, tt.description)
			if len(tags) != len(tt.wantTags) {
				t.Errorf("expected %d tags, got %d: %v", len(tt.wantTags), len(tags), tags)
				return
			}
			tagSet := make(map[string]bool)
			for _, tag := range tags {
				tagSet[tag] = true
			}
			for _, want := range tt.wantTags {
				if !tagSet[want] {
					t.Errorf("expected tag %q not found in %v", want, tags)
				}
			}
		})
	}
}

func TestExtractMentionedFiles(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
		wantFiles   []string
	}{
		{
			name:      "go file in title",
			title:     "Fix bug in internal/config/config.go",
			wantFiles: []string{"internal/config/config.go"},
		},
		{
			name:      "multiple files",
			title:     "Update cmd/main.go and internal/cli/run.go",
			wantFiles: []string{"cmd/main.go", "internal/cli/run.go"},
		},
		{
			name:      "glob pattern",
			title:     "Refactor internal/**/*.go files",
			wantFiles: []string{"internal/**/*.go"},
		},
		{
			name:        "files in description",
			title:       "Fix tests",
			description: "The tests in tests/e2e/main_test.go are failing",
			wantFiles:   []string{"tests/e2e/main_test.go"},
		},
		{
			name:      "no files mentioned",
			title:     "Improve performance of the system",
			wantFiles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := ExtractMentionedFiles(tt.title, tt.description)
			if len(files) != len(tt.wantFiles) {
				t.Errorf("expected %d files, got %d: %v", len(tt.wantFiles), len(files), files)
				return
			}
			fileSet := make(map[string]bool)
			for _, f := range files {
				fileSet[f] = true
			}
			for _, want := range tt.wantFiles {
				if !fileSet[want] {
					t.Errorf("expected file %q not found in %v", want, files)
				}
			}
		})
	}
}

func TestIsFilePath(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"internal/config/config.go", true},
		{"cmd/main.go", true},
		{"pkg/**/*.ts", true},
		{"README.md", true},
		{".gitignore", true},
		{"hello", false},
		{"the", false},
		{"Fix", false},
		{"", false},
		{"a", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isFilePath(tt.input)
			if got != tt.want {
				t.Errorf("isFilePath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatchFocusPattern(t *testing.T) {
	tests := []struct {
		pattern string
		file    string
		want    bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "main.ts", false},
		{"internal/**", "internal/cli/run.go", true},
		{"internal/**", "cmd/main.go", false},
		{"**/*.go", "internal/config/config.go", true},
		{"**/*.go", "config.ts", false},
		{"internal/*.go", "internal/foo.go", true},
		{"internal/*.go", "internal/sub/foo.go", false},
		{"docs/**", "docs/README.md", true},
		{"tests/**", "internal/test.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.file, func(t *testing.T) {
			got := matchFocusPattern(tt.pattern, tt.file)
			if got != tt.want {
				t.Errorf("matchFocusPattern(%q, %q) = %v, want %v", tt.pattern, tt.file, got, tt.want)
			}
		})
	}
}

func TestComputeProfileTagBonus(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		taskTags []string
		weight   float64
		wantMin  float64
		wantMax  float64
	}{
		{
			name:     "full match",
			tags:     []string{"testing", "qa"},
			taskTags: []string{"testing"},
			weight:   0.15,
			wantMin:  0.075, // 50% match * 0.15
			wantMax:  0.16,
		},
		{
			name:     "no match",
			tags:     []string{"documentation"},
			taskTags: []string{"testing"},
			weight:   0.15,
			wantMin:  0,
			wantMax:  0,
		},
		{
			name:     "multiple overlapping tags",
			tags:     []string{"testing", "qa", "quality"},
			taskTags: []string{"testing", "quality"},
			weight:   0.15,
			wantMin:  0.09, // 2/3 match * 0.15 = 0.10
			wantMax:  0.16,
		},
		{
			name:     "nil profile tags",
			tags:     nil,
			taskTags: []string{"testing"},
			weight:   0.15,
			wantMin:  0,
			wantMax:  0,
		},
		{
			name:     "empty task tags",
			tags:     []string{"testing"},
			taskTags: []string{},
			weight:   0.15,
			wantMin:  0,
			wantMax:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &persona.Persona{Tags: tt.tags}
			bonus := computeProfileTagBonus(profile, tt.taskTags, tt.weight)
			if bonus < tt.wantMin || bonus > tt.wantMax {
				t.Errorf("expected bonus in [%f, %f], got %f", tt.wantMin, tt.wantMax, bonus)
			}
		})
	}
}

func TestComputeFocusPatternBonus(t *testing.T) {
	tests := []struct {
		name           string
		focusPatterns  []string
		mentionedFiles []string
		weight         float64
		wantMin        float64
		wantMax        float64
	}{
		{
			name:           "match internal files",
			focusPatterns:  []string{"internal/**"},
			mentionedFiles: []string{"internal/config/config.go"},
			weight:         0.10,
			wantMin:        0.09,
			wantMax:        0.11,
		},
		{
			name:           "no match",
			focusPatterns:  []string{"docs/**"},
			mentionedFiles: []string{"internal/config/config.go"},
			weight:         0.10,
			wantMin:        0,
			wantMax:        0,
		},
		{
			name:           "partial match",
			focusPatterns:  []string{"internal/**", "docs/**"},
			mentionedFiles: []string{"internal/cli.go", "cmd/main.go"},
			weight:         0.10,
			wantMin:        0.04, // 1/2 match * 0.10
			wantMax:        0.06,
		},
		{
			name:           "nil focus patterns",
			focusPatterns:  nil,
			mentionedFiles: []string{"internal/cli.go"},
			weight:         0.10,
			wantMin:        0,
			wantMax:        0,
		},
		{
			name:           "empty mentioned files",
			focusPatterns:  []string{"internal/**"},
			mentionedFiles: []string{},
			weight:         0.10,
			wantMin:        0,
			wantMax:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &persona.Persona{FocusPatterns: tt.focusPatterns}
			bonus := computeFocusPatternBonus(profile, tt.mentionedFiles, tt.weight)
			if bonus < tt.wantMin || bonus > tt.wantMax {
				t.Errorf("expected bonus in [%f, %f], got %f", tt.wantMin, tt.wantMax, bonus)
			}
		})
	}
}

func TestScoreAssignmentWithProfile(t *testing.T) {
	// Test that profile bonuses are applied correctly in scoreAssignment
	testerProfile := &persona.Persona{
		Name:          "tester",
		Tags:          []string{"testing", "qa", "quality"},
		FocusPatterns: []string{"**/*_test.go", "tests/**"},
	}

	agent := &AgentState{
		PaneID:    "%1",
		AgentType: "cc",
		Profile:   testerProfile,
	}

	// Task that matches tester profile
	testingTask := &bv.TriageRecommendation{
		ID:    "ntm-001",
		Title: "Add unit tests for internal/config/config_test.go",
		Type:  "task",
		Score: 0.5,
	}

	// Task that doesn't match tester profile
	docTask := &bv.TriageRecommendation{
		ID:    "ntm-002",
		Title: "Update documentation in docs/README.md",
		Type:  "task",
		Score: 0.5,
	}

	config := ScoreConfig{
		UseAgentProfiles:        true,
		ProfileTagBoostWeight:   0.15,
		FocusPatternBoostWeight: 0.10,
	}

	testingResult := scoreAssignment(agent, testingTask, config, nil)
	docResult := scoreAssignment(agent, docTask, config, nil)

	// Testing task should have higher profile bonuses
	if testingResult.ScoreBreakdown.ProfileTagBonus <= docResult.ScoreBreakdown.ProfileTagBonus {
		t.Errorf("testing task should have higher ProfileTagBonus: testing=%f, doc=%f",
			testingResult.ScoreBreakdown.ProfileTagBonus, docResult.ScoreBreakdown.ProfileTagBonus)
	}

	if testingResult.ScoreBreakdown.FocusPatternBonus <= docResult.ScoreBreakdown.FocusPatternBonus {
		t.Errorf("testing task should have higher FocusPatternBonus: testing=%f, doc=%f",
			testingResult.ScoreBreakdown.FocusPatternBonus, docResult.ScoreBreakdown.FocusPatternBonus)
	}

	// Total score for testing task should be higher
	if testingResult.TotalScore <= docResult.TotalScore {
		t.Errorf("testing task should have higher total score: testing=%f, doc=%f",
			testingResult.TotalScore, docResult.TotalScore)
	}
}

func TestScoreAssignmentWithNilProfile(t *testing.T) {
	// Test that nil profile doesn't cause panic and results in zero bonuses
	agent := &AgentState{
		PaneID:    "%1",
		AgentType: "cc",
		Profile:   nil,
	}

	task := &bv.TriageRecommendation{
		ID:    "ntm-001",
		Title: "Add unit tests",
		Type:  "task",
		Score: 0.5,
	}

	config := ScoreConfig{
		UseAgentProfiles:        true,
		ProfileTagBoostWeight:   0.15,
		FocusPatternBoostWeight: 0.10,
	}

	result := scoreAssignment(agent, task, config, nil)

	if result.ScoreBreakdown.ProfileTagBonus != 0 {
		t.Errorf("expected zero ProfileTagBonus with nil profile, got %f", result.ScoreBreakdown.ProfileTagBonus)
	}
	if result.ScoreBreakdown.FocusPatternBonus != 0 {
		t.Errorf("expected zero FocusPatternBonus with nil profile, got %f", result.ScoreBreakdown.FocusPatternBonus)
	}
}
