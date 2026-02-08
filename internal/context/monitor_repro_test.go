package context

import (
	"testing"
)

func TestParseRobotModeContext_Repro(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *ContextEstimate
	}{
		{
			name: "multiline json with text prefix",
			input: `Here is the current status:
{
  "context_used": 145000,
  "context_limit": 200000
}
Hope that helps!`,
			expected: &ContextEstimate{
				TokensUsed:   145000,
				ContextLimit: 200000,
			},
		},
		{
			name: "json embedded in markdown block",
			input: "Here is the data:\n```json\n{\"context_used\": 50000, \"context_limit\": 100000}\n```",
			expected: &ContextEstimate{
				TokensUsed:   50000,
				ContextLimit: 100000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseRobotModeContext(tt.input)
			if got == nil {
				t.Fatalf("ParseRobotModeContext() returned nil, expected valid estimate")
			}
			if got.TokensUsed != tt.expected.TokensUsed {
				t.Errorf("TokensUsed = %d, want %d", got.TokensUsed, tt.expected.TokensUsed)
			}
		})
	}
}
