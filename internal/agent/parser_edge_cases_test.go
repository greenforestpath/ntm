package agent

import (
	"testing"
)

func TestParser_EdgeCases_UserTyping(t *testing.T) {
	p := NewParser()

	// Scenario: User types "I am thinking..." at the prompt.
	// "thinking" is a working pattern for Claude.
	// We want to ensure this isn't flagged as the AGENT working.
	output := `Claude Opus 4.5 ready
> I am thinking about the next step...`

	state, err := p.Parse(output)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if state.Type != AgentTypeClaudeCode {
		t.Errorf("Type = %v, want %v", state.Type, AgentTypeClaudeCode)
	}

	// This is the critical check.
	// If the user is typing, the AGENT is idle (waiting for input), but the parser
	// might see "thinking" and claim IsWorking=true.
	if state.IsWorking {
		t.Error("FAIL: IsWorking is true when user is typing 'thinking' at prompt")
	}
}
