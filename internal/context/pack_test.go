package context

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Dicklesworthstone/ntm/internal/state"
)

func TestDefaultBudgetAllocation(t *testing.T) {
	t.Parallel()

	alloc := DefaultBudgetAllocation()
	if alloc.Triage != 10 {
		t.Errorf("Triage = %d, want 10", alloc.Triage)
	}
	if alloc.CM != 5 {
		t.Errorf("CM = %d, want 5", alloc.CM)
	}
	if alloc.CASS != 15 {
		t.Errorf("CASS = %d, want 15", alloc.CASS)
	}
	if alloc.S2P != 70 {
		t.Errorf("S2P = %d, want 70", alloc.S2P)
	}

	total := alloc.Triage + alloc.CM + alloc.CASS + alloc.S2P
	if total != 100 {
		t.Errorf("total allocation = %d, want 100", total)
	}
}

func TestCacheKey(t *testing.T) {
	t.Parallel()

	opts := BuildOptions{
		RepoRev:   "abc123",
		BeadID:    "bd-test",
		AgentType: "cc",
	}
	key := cacheKey(opts)

	if len(key) != 16 {
		t.Errorf("cacheKey length = %d, want 16", len(key))
	}

	// Same options should produce the same key
	key2 := cacheKey(opts)
	if key != key2 {
		t.Errorf("same options produced different keys: %q vs %q", key, key2)
	}

	// Different options should produce different keys
	opts2 := BuildOptions{
		RepoRev:   "def456",
		BeadID:    "bd-test",
		AgentType: "cc",
	}
	key3 := cacheKey(opts2)
	if key == key3 {
		t.Errorf("different options produced same key: %q", key)
	}
}

func TestPackEstimateTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty string", "", 0},
		{"4 chars = 1 token", "abcd", 1},
		{"8 chars = 2 tokens", "abcdefgh", 2},
		{"3 chars = 0 tokens (floor)", "abc", 0},
		{"100 chars", strings.Repeat("a", 100), 25},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := estimateTokens(tc.input)
			if result != tc.expected {
				t.Errorf("estimateTokens(%q) = %d, want %d", tc.input, result, tc.expected)
			}
		})
	}
}

func TestComponentTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected string
	}{
		{"triage", "BV Triage (Priority & Planning)"},
		{"cm", "CM Rules (Learned Guidelines)"},
		{"cass", "CASS History (Prior Solutions)"},
		{"s2p", "File Context"},
		{"custom", "Custom"},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := componentTitle(tc.name)
			if result != tc.expected {
				t.Errorf("componentTitle(%q) = %q, want %q", tc.name, result, tc.expected)
			}
		})
	}
}

func TestTruncateJSON_Small(t *testing.T) {
	t.Parallel()

	data := json.RawMessage(`{"key":"value"}`)
	result := truncateJSON(data, 1000) // budget is large enough
	if string(result) != string(data) {
		t.Errorf("small JSON was truncated: got %q, want %q", result, data)
	}
}

func TestTruncateJSON_Array(t *testing.T) {
	t.Parallel()

	// Build a JSON array with 10 elements
	arr := make([]string, 10)
	for i := range arr {
		arr[i] = strings.Repeat("x", 50)
	}
	data, _ := json.Marshal(arr)

	// Use a tight budget that should truncate
	result := truncateJSON(data, 50) // 50 tokens * 4 = 200 chars budget
	if len(result) > 200 {
		t.Errorf("truncated array too large: %d chars > 200", len(result))
	}

	// Result should be valid JSON
	var parsed interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Errorf("truncated array is not valid JSON: %v", err)
	}
}

func TestTruncateJSON_Object(t *testing.T) {
	t.Parallel()

	// Build a large JSON object
	obj := make(map[string]string)
	for i := 0; i < 20; i++ {
		obj[strings.Repeat("k", 10)+string(rune('a'+i))] = strings.Repeat("v", 100)
	}
	data, _ := json.Marshal(obj)

	// Use budget too small for full object but large enough for some fields
	result := truncateJSON(data, 100) // 100 tokens * 4 = 400 chars
	if len(result) > 400 {
		t.Errorf("truncated object too large: %d chars > 400", len(result))
	}

	// Result should be valid JSON with truncation marker
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Errorf("truncated object is not valid JSON: %v", err)
	}
	if _, ok := parsed["_truncated"]; !ok {
		t.Error("truncated object should have _truncated key")
	}
}

func TestTruncateJSON_InvalidJSON(t *testing.T) {
	t.Parallel()

	data := json.RawMessage(`not valid json at all`)
	result := truncateJSON(data, 5) // Very small budget

	// Should return valid JSON fallback
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Errorf("fallback is not valid JSON: %v", err)
	}
	if _, ok := parsed["_truncated"]; !ok {
		t.Error("fallback should have _truncated key")
	}
}

func TestTruncateText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		text        string
		tokenBudget int
		wantFull    bool
		wantSuffix  string
	}{
		{"short text fits", "hello", 100, true, ""},
		{"exact fit", strings.Repeat("a", 400), 100, true, ""},
		{"needs truncation", strings.Repeat("a", 500), 100, false, "...[truncated]"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := truncateText(tc.text, tc.tokenBudget)
			if tc.wantFull {
				if result != tc.text {
					t.Errorf("expected full text, got truncated (%d chars)", len(result))
				}
			} else {
				if !strings.HasSuffix(result, tc.wantSuffix) {
					t.Errorf("truncated text should end with %q, got %q", tc.wantSuffix, result[len(result)-20:])
				}
			}
		})
	}
}

func TestCalculateFilePriority(t *testing.T) {
	t.Parallel()

	b := &ContextPackBuilder{}

	tests := []struct {
		file     string
		minScore int
	}{
		// High priority: main entry points
		{"cmd/main.go", 100},
		{"src/index.ts", 100},
		{"app.py", 100},
		// Medium-high: core logic
		{"internal/core/engine.go", 50},
		{"services/auth.go", 50},
		{"controller/user.go", 50},
		{"handler/api.go", 50},
		// Medium: config/routing
		{"config/database.yaml", 30},
		{"router/routes.go", 30},
		{"middleware/auth.go", 30},
		// Lower priority: tests
		{"pkg/service_test.go", -20},
		{"spec/auth_spec.rb", -20},
	}

	for _, tc := range tests {
		t.Run(tc.file, func(t *testing.T) {
			priority := b.calculateFilePriority(tc.file)
			if priority < tc.minScore {
				t.Errorf("calculateFilePriority(%q) = %d, want >= %d", tc.file, priority, tc.minScore)
			}
		})
	}

	// Short paths should get bonus
	shallow := b.calculateFilePriority("src/main.go") // 2 slashes -> bonus
	deep := b.calculateFilePriority("a/b/c/d/main.go")
	if shallow <= deep {
		t.Errorf("shallow path (%d) should have higher priority than deep path (%d)", shallow, deep)
	}
}

func TestSelectS2PFormat(t *testing.T) {
	t.Parallel()

	b := &ContextPackBuilder{}

	if f := b.selectS2PFormat(20000); f != "compact" {
		t.Errorf("small budget should use compact, got %q", f)
	}
	if f := b.selectS2PFormat(29999); f != "compact" {
		t.Errorf("just under 30k should use compact, got %q", f)
	}
	if f := b.selectS2PFormat(30000); f != "" {
		t.Errorf("30k budget should use default, got %q", f)
	}
	if f := b.selectS2PFormat(100000); f != "" {
		t.Errorf("large budget should use default, got %q", f)
	}
}

func TestIntelligentTruncate_Short(t *testing.T) {
	t.Parallel()

	b := &ContextPackBuilder{}
	text := "short content"
	result := b.intelligentTruncate(text, 1000)
	if result != text {
		t.Errorf("short content should not be truncated")
	}
}

func TestIntelligentTruncate_Long(t *testing.T) {
	t.Parallel()

	b := &ContextPackBuilder{}

	// Build content with headers and body
	var sb strings.Builder
	sb.WriteString("=== File: important.go ===\n")
	sb.WriteString("# Header\n")
	for i := 0; i < 100; i++ {
		sb.WriteString(strings.Repeat("content line ", 10) + "\n")
	}
	text := sb.String()

	// Truncate to small budget
	result := b.intelligentTruncate(text, 50) // 200 chars
	if len(result) > 250 { // some slack for truncation message
		t.Errorf("truncated text too long: %d chars", len(result))
	}
	if !strings.Contains(result, "truncated") {
		t.Error("truncated text should contain truncation marker")
	}
	// Headers should be preserved
	if !strings.Contains(result, "=== File: important.go ===") {
		t.Error("file header should be preserved in truncated output")
	}
}

func TestOptimizeFilesForBudget(t *testing.T) {
	t.Parallel()

	b := &ContextPackBuilder{}

	files := []string{
		"cmd/main.go",
		"internal/core/engine.go",
		"internal/util/helper.go",
		"internal/api/handler.go",
		"tests/integration_test.go",
		"docs/readme.md",
		"config/settings.yaml",
		"scripts/deploy.sh",
	}

	// Small budget should limit files
	result := b.optimizeFilesForBudget(files, 5000) // 5000/2000 = 2.5, min 3
	if len(result) > 3 {
		t.Errorf("small budget: got %d files, want <= 3", len(result))
	}

	// Large budget should keep all files
	result = b.optimizeFilesForBudget(files, 100000)
	if len(result) != len(files) {
		t.Errorf("large budget: got %d files, want %d", len(result), len(files))
	}
}

func TestOptimizeFilesForBudget_PriorityOrder(t *testing.T) {
	t.Parallel()

	b := &ContextPackBuilder{}

	files := []string{
		"tests/unit_test.go",   // low priority
		"cmd/main.go",          // high priority
		"internal/handler.go",  // medium-high priority
		"examples/demo.go",     // low priority
		"internal/service.go",  // medium-high priority
		"internal/core/core.go", // medium-high priority
	}

	// Budget allows 3 files
	result := b.optimizeFilesForBudget(files, 6000)
	if len(result) != 3 {
		t.Fatalf("expected 3 files, got %d", len(result))
	}

	// main.go should be included (highest priority)
	found := false
	for _, f := range result {
		if f == "cmd/main.go" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("main.go should be in top-3 prioritized files, got %v", result)
	}
}

func TestTokenBudgets(t *testing.T) {
	t.Parallel()

	if TokenBudgets["cc"] != 180000 {
		t.Errorf("cc budget = %d, want 180000", TokenBudgets["cc"])
	}
	if TokenBudgets["cod"] != 120000 {
		t.Errorf("cod budget = %d, want 120000", TokenBudgets["cod"])
	}
	if TokenBudgets["gmi"] != 100000 {
		t.Errorf("gmi budget = %d, want 100000", TokenBudgets["gmi"])
	}
	if TokenBudgets["default"] != 100000 {
		t.Errorf("default budget = %d, want 100000", TokenBudgets["default"])
	}
}

func TestRenderXML(t *testing.T) {
	t.Parallel()

	b := &ContextPackBuilder{allocation: DefaultBudgetAllocation()}

	pack := &ContextPackFull{
		ContextPack: state.ContextPack{
			ID:        "pack-test",
			BeadID:    "bd-123",
			AgentType: state.AgentTypeClaude,
			RepoRev:   "abc123",
		},
		Components: map[string]*PackComponent{
			"triage": {Type: "triage", Data: json.RawMessage(`{"picks":[]}`), TokenCount: 10},
			"cm":     {Type: "cm", Error: "cm not installed"},
		},
	}

	result := b.renderXML(pack)

	if !strings.Contains(result, "<context_pack>") {
		t.Error("XML should contain <context_pack> tag")
	}
	if !strings.Contains(result, "<id>pack-test</id>") {
		t.Error("XML should contain pack ID")
	}
	if !strings.Contains(result, "<triage>") {
		t.Error("XML should contain triage component")
	}
	if !strings.Contains(result, `cm unavailable="true"`) {
		t.Error("XML should show cm as unavailable")
	}
}

func TestRenderMarkdown(t *testing.T) {
	t.Parallel()

	b := &ContextPackBuilder{allocation: DefaultBudgetAllocation()}

	pack := &ContextPackFull{
		ContextPack: state.ContextPack{
			ID:        "pack-test",
			BeadID:    "bd-123",
			AgentType: "cod",
			RepoRev:   "abc123",
		},
		Components: map[string]*PackComponent{
			"triage": {Type: "triage", Data: json.RawMessage(`{"picks":[]}`), TokenCount: 10},
			"cass":   {Type: "cass", Error: "cass not installed"},
		},
	}

	result := b.renderMarkdown(pack)

	if !strings.Contains(result, "# Context Pack") {
		t.Error("markdown should contain header")
	}
	if !strings.Contains(result, "**ID**: pack-test") {
		t.Error("markdown should contain pack ID")
	}
	if !strings.Contains(result, "```json") {
		t.Error("markdown should contain JSON code block for triage")
	}
	if !strings.Contains(result, "*Unavailable: cass not installed*") {
		t.Error("markdown should show cass as unavailable")
	}
}

func TestRender_RoutesToCorrectFormat(t *testing.T) {
	t.Parallel()

	b := &ContextPackBuilder{allocation: DefaultBudgetAllocation()}

	claudePack := &ContextPackFull{
		ContextPack: state.ContextPack{AgentType: state.AgentTypeClaude},
		Components:  map[string]*PackComponent{},
	}
	codexPack := &ContextPackFull{
		ContextPack: state.ContextPack{AgentType: state.AgentTypeCodex},
		Components:  map[string]*PackComponent{},
	}

	xmlResult := b.render(claudePack)
	if !strings.Contains(xmlResult, "<context_pack>") {
		t.Error("Claude agent should get XML format")
	}

	mdResult := b.render(codexPack)
	if !strings.Contains(mdResult, "# Context Pack") {
		t.Error("Codex agent should get Markdown format")
	}
}

func TestGeneratePackID(t *testing.T) {
	t.Parallel()

	id1 := generatePackID()
	id2 := generatePackID()

	if !strings.HasPrefix(id1, "pack-") {
		t.Errorf("pack ID should start with 'pack-', got %q", id1)
	}
	// IDs should be unique (different nanosecond timestamps)
	if id1 == id2 {
		t.Logf("warning: consecutive pack IDs are equal (rare but possible): %q", id1)
	}
}

func TestCacheStatsAndClear(t *testing.T) {
	// Not parallel since it mutates global cache
	b := &ContextPackBuilder{}

	// Clear first to isolate
	b.ClearCache()

	size, keys := b.CacheStats()
	if size != 0 {
		t.Errorf("after clear, cache size = %d, want 0", size)
	}
	if len(keys) != 0 {
		t.Errorf("after clear, keys = %v, want empty", keys)
	}

	// Populate cache manually to test stats
	globalCacheMu.Lock()
	globalCache["test-key-1"] = &ContextPackFull{}
	globalCache["test-key-2"] = &ContextPackFull{}
	globalCacheMu.Unlock()

	size, keys = b.CacheStats()
	if size != 2 {
		t.Errorf("cache size = %d, want 2", size)
	}
	if len(keys) != 2 {
		t.Errorf("keys count = %d, want 2", len(keys))
	}

	// Clear and verify
	b.ClearCache()
	size, _ = b.CacheStats()
	if size != 0 {
		t.Errorf("after second clear, cache size = %d, want 0", size)
	}
}
