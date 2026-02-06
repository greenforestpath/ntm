package agent

import (
	"regexp"
	"testing"
)

// =============================================================================
// extractFloat / extractInt missing branches
// =============================================================================

// TestExtractFloat_ShortMatch tests the len(match) < 2 branch: pattern matches
// but the match array has fewer than 2 elements (no capture group).
func TestExtractFloat_ShortMatch(t *testing.T) {
	t.Parallel()

	// Pattern with no capture group — match has exactly 1 element (full match)
	noGroup := regexp.MustCompile(`\d+\.?\d*`)
	got := extractFloat(noGroup, "42.5")
	if got != nil {
		t.Errorf("extractFloat with no capture group = %v, want nil", *got)
	}
}

// TestExtractFloat_ParseError tests the parse error branch.
func TestExtractFloat_ParseError(t *testing.T) {
	t.Parallel()

	// Pattern with capture group that captures non-numeric text
	badCapture := regexp.MustCompile(`value=(\w+)`)
	got := extractFloat(badCapture, "value=abc")
	if got != nil {
		t.Errorf("extractFloat with unparseable capture = %v, want nil", *got)
	}
}

// TestExtractFloat_CommaNumber tests comma-separated number parsing.
func TestExtractFloat_CommaNumber(t *testing.T) {
	t.Parallel()

	commaPattern := regexp.MustCompile(`total=([\d,]+\.?\d*)`)
	got := extractFloat(commaPattern, "total=1,234.5")
	if got == nil {
		t.Fatal("extractFloat with comma number = nil, want 1234.5")
	}
	if *got != 1234.5 {
		t.Errorf("extractFloat with comma number = %v, want 1234.5", *got)
	}
}

// TestExtractInt_ShortMatch tests the len(match) < 2 branch.
func TestExtractInt_ShortMatch(t *testing.T) {
	t.Parallel()

	noGroup := regexp.MustCompile(`\d+`)
	got := extractInt(noGroup, "42")
	if got != nil {
		t.Errorf("extractInt with no capture group = %v, want nil", *got)
	}
}

// TestExtractInt_ParseError tests the parse error branch.
func TestExtractInt_ParseError(t *testing.T) {
	t.Parallel()

	badCapture := regexp.MustCompile(`count=(\w+)`)
	got := extractInt(badCapture, "count=xyz")
	if got != nil {
		t.Errorf("extractInt with unparseable capture = %v, want nil", *got)
	}
}

// TestExtractInt_MultipleMatches tests that the last match is used.
func TestExtractInt_MultipleMatches(t *testing.T) {
	t.Parallel()

	// Pattern that matches multiple times — should use the last
	numPat := regexp.MustCompile(`(\d+)`)
	got := extractInt(numPat, "first=10 second=20 third=30")
	if got == nil {
		t.Fatal("extractInt = nil, want 30")
	}
	if *got != 30 {
		t.Errorf("extractInt = %d, want 30 (last match)", *got)
	}
}

// =============================================================================
// calculateConfidence branches
// =============================================================================

// TestCalculateConfidence_Base tests the base confidence with no special signals.
func TestCalculateConfidence_Base(t *testing.T) {
	t.Parallel()

	p := &parserImpl{}
	state := &AgentState{
		Type: AgentTypeClaudeCode,
	}
	conf := p.calculateConfidence(state)
	if conf != 0.5 {
		t.Errorf("base confidence = %f, want 0.5", conf)
	}
}

// TestCalculateConfidence_ContextRemaining tests the +0.25 boost.
func TestCalculateConfidence_ContextRemaining(t *testing.T) {
	t.Parallel()

	p := &parserImpl{}
	ctx := 50.0
	state := &AgentState{
		Type:             AgentTypeClaudeCode,
		ContextRemaining: &ctx,
	}
	conf := p.calculateConfidence(state)
	if conf != 0.75 {
		t.Errorf("confidence with ContextRemaining = %f, want 0.75", conf)
	}
}

// TestCalculateConfidence_TokensUsed tests the +0.05 boost.
func TestCalculateConfidence_TokensUsed(t *testing.T) {
	t.Parallel()

	p := &parserImpl{}
	tokens := int64(1000)
	state := &AgentState{
		Type:       AgentTypeClaudeCode,
		TokensUsed: &tokens,
	}
	conf := p.calculateConfidence(state)
	if conf != 0.55 {
		t.Errorf("confidence with TokensUsed = %f, want 0.55", conf)
	}
}

// TestCalculateConfidence_WorkIndicators tests the +0.1*min(n,3) boost.
func TestCalculateConfidence_WorkIndicators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		indicators []string
		want       float64
	}{
		{"one indicator", []string{"writing"}, 0.6},
		{"two indicators", []string{"writing", "reading"}, 0.7},
		{"three indicators", []string{"a", "b", "c"}, 0.8},
		{"four capped at 3", []string{"a", "b", "c", "d"}, 0.8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &parserImpl{}
			state := &AgentState{
				Type:           AgentTypeClaudeCode,
				WorkIndicators: tt.indicators,
			}
			conf := p.calculateConfidence(state)
			if conf != tt.want {
				t.Errorf("confidence = %f, want %f", conf, tt.want)
			}
		})
	}
}

// TestCalculateConfidence_LimitIndicators tests the +0.2 boost.
func TestCalculateConfidence_LimitIndicators(t *testing.T) {
	t.Parallel()

	p := &parserImpl{}
	state := &AgentState{
		Type:            AgentTypeClaudeCode,
		LimitIndicators: []string{"rate limit"},
	}
	conf := p.calculateConfidence(state)
	if conf != 0.7 {
		t.Errorf("confidence with LimitIndicators = %f, want 0.7", conf)
	}
}

// TestCalculateConfidence_UnknownType tests the -0.3 penalty.
func TestCalculateConfidence_UnknownType(t *testing.T) {
	t.Parallel()

	p := &parserImpl{}
	state := &AgentState{
		Type: AgentTypeUnknown,
	}
	conf := p.calculateConfidence(state)
	if conf != 0.2 {
		t.Errorf("confidence with Unknown type = %f, want 0.2", conf)
	}
}

// TestCalculateConfidence_ConflictPenalty tests the -0.2 penalty for conflicting signals.
func TestCalculateConfidence_ConflictPenalty(t *testing.T) {
	t.Parallel()

	p := &parserImpl{}
	state := &AgentState{
		Type:      AgentTypeClaudeCode,
		IsWorking: true,
		IsIdle:    true,
	}
	conf := p.calculateConfidence(state)
	if conf != 0.3 { // 0.5 - 0.2
		t.Errorf("confidence with conflict = %f, want 0.3", conf)
	}
}

// TestCalculateConfidence_ClampHigh tests clamping to 1.0.
func TestCalculateConfidence_ClampHigh(t *testing.T) {
	t.Parallel()

	p := &parserImpl{}
	ctx := 50.0
	tokens := int64(1000)
	state := &AgentState{
		Type:             AgentTypeClaudeCode,
		ContextRemaining: &ctx,     // +0.25
		TokensUsed:       &tokens,  // +0.05
		WorkIndicators:   []string{"a", "b", "c"}, // +0.3
		LimitIndicators:  []string{"limit"},        // +0.2
	}
	// 0.5 + 0.25 + 0.05 + 0.3 + 0.2 = 1.3, clamped to 1.0
	conf := p.calculateConfidence(state)
	if conf != 1.0 {
		t.Errorf("confidence should clamp to 1.0, got %f", conf)
	}
}

// TestCalculateConfidence_ClampLow tests clamping to 0.0.
func TestCalculateConfidence_ClampLow(t *testing.T) {
	t.Parallel()

	p := &parserImpl{}
	state := &AgentState{
		Type:      AgentTypeUnknown, // -0.3
		IsWorking: true,
		IsIdle:    true, // -0.2
	}
	// 0.5 - 0.3 - 0.2 = 0.0
	conf := p.calculateConfidence(state)
	if conf != 0.0 {
		t.Errorf("confidence should clamp to 0.0, got %f", conf)
	}
}

// =============================================================================
// ProfileName missing Ollama case
// =============================================================================

// TestAgentType_ProfileName_Ollama tests the Ollama case in ProfileName.
func TestAgentType_ProfileName_Ollama(t *testing.T) {
	t.Parallel()

	if got := AgentTypeOllama.ProfileName(); got != "Ollama" {
		t.Errorf("AgentTypeOllama.ProfileName() = %q, want %q", got, "Ollama")
	}
}
