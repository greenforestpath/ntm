package summary

import "testing"

func TestAggregateOutputsBasic(t *testing.T) {
	outputs := []AgentOutput{
		{
			AgentID: "agent1",
			Output: `
Some chat text.
{
	"accomplishments": ["Fixed bug A"],
	"changes": [{"path": "main.go", "action": "modified"}]
}
Some more text.
- Next: Fix bug B
`,
		},
		{
			AgentID: "agent2",
			Output: `
## Accomplishments
- Added feature C

## Files
- created utils.go
`,
		},
	}

	data := aggregateOutputs(outputs)

	expectedAccomplishments := []string{"Fixed bug A", "Added feature C"}
	if !elementsMatch(data.accomplishments, expectedAccomplishments) {
		t.Errorf("Accomplishments: got %v, want %v", data.accomplishments, expectedAccomplishments)
	}

	expectedPending := []string{"Fix bug B"}
	if !elementsMatch(data.pending, expectedPending) {
		t.Errorf("Pending: got %v, want %v", data.pending, expectedPending)
	}

	// Files are harder to match exactly due to struct nature, checking paths/actions
	fileMap := make(map[string]string)
	for _, f := range data.files {
		fileMap[f.Path] = f.Action
	}

	if fileMap["main.go"] != "modified" {
		t.Errorf("main.go: got %s, want modified", fileMap["main.go"])
	}
	if fileMap["utils.go"] != "created" {
		t.Errorf("utils.go: got %s, want created", fileMap["utils.go"])
	}
}

func elementsMatch(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int)
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
		if seen[s] < 0 {
			return false
		}
	}
	return true
}

func TestExtractCompleteJSON(t *testing.T) {
	input := []string{
		"prefix",
		"{",
		`  "key": "value",`,
		`  "nested": {`,
		`    "a": 1`,
		`  }`,
		"}",
		"suffix",
	}

	jsonStr, nextIdx := extractCompleteJSON(input, 1)
	if nextIdx != 7 {
		t.Errorf("Next index: got %d, want 7", nextIdx)
	}
	if !isValidJSON(jsonStr) {
		t.Errorf("Extracted invalid JSON: %s", jsonStr)
	}
}
