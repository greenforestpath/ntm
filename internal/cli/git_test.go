package cli

import (
	"testing"
)

func TestParseConflicts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		output   string
		expected []string
	}{
		{
			name:     "no conflicts",
			output:   "Already up to date.\n",
			expected: nil,
		},
		{
			name:     "empty string",
			output:   "",
			expected: nil,
		},
		{
			name: "single conflict",
			output: `Auto-merging file.go
CONFLICT (content): Merge conflict in file.go
Automatic merge failed; fix conflicts and then commit the result.`,
			expected: []string{"CONFLICT (content): Merge conflict in file.go"},
		},
		{
			name: "multiple conflicts",
			output: `Auto-merging internal/cli/root.go
CONFLICT (content): Merge conflict in internal/cli/root.go
Auto-merging internal/cli/spawn.go
CONFLICT (content): Merge conflict in internal/cli/spawn.go
CONFLICT (modify/delete): internal/cli/old.go deleted in HEAD and modified in feature.
Automatic merge failed; fix conflicts and then commit the result.`,
			expected: []string{
				"CONFLICT (content): Merge conflict in internal/cli/root.go",
				"CONFLICT (content): Merge conflict in internal/cli/spawn.go",
				"CONFLICT (modify/delete): internal/cli/old.go deleted in HEAD and modified in feature.",
			},
		},
		{
			name: "add/add conflict",
			output: `CONFLICT (add/add): Merge conflict in newfile.go
Auto-merging newfile.go`,
			expected: []string{"CONFLICT (add/add): Merge conflict in newfile.go"},
		},
		{
			name:     "no CONFLICT prefix",
			output:   "conflict in the message but not at start\n",
			expected: nil,
		},
		{
			name: "mixed output with conflicts",
			output: `Updating abc1234..def5678
Fast-forward
 file1.go | 10 ++++++++++
 1 file changed, 10 insertions(+)
Already up to date.
Then some more text.
CONFLICT (content): Merge conflict in critical.go
More text after conflict.`,
			expected: []string{"CONFLICT (content): Merge conflict in critical.go"},
		},
		{
			name:     "whitespace only",
			output:   "   \n\t\n   ",
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := parseConflicts(tc.output)

			if len(result) != len(tc.expected) {
				t.Fatalf("parseConflicts() returned %d conflicts; want %d\nGot: %v\nWant: %v",
					len(result), len(tc.expected), result, tc.expected)
			}

			for i, conflict := range result {
				if conflict != tc.expected[i] {
					t.Errorf("parseConflicts()[%d] = %q; want %q", i, conflict, tc.expected[i])
				}
			}
		})
	}
}
