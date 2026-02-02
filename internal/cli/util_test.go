package cli

import (
	"sort"
	"strings"
	"testing"

	"github.com/Dicklesworthstone/ntm/internal/ensemble"
	"github.com/Dicklesworthstone/ntm/internal/tmux"
)

func TestResolveExplicitSessionName(t *testing.T) {
	sessions := []tmux.Session{
		{Name: "alpha"},
		{Name: "beta"},
		{Name: "my_project"},
		{Name: "my_test"},
	}

	cases := []struct {
		name        string
		input       string
		allowPrefix bool
		sessions    []tmux.Session
		want        string
		wantReason  string
		wantErr     bool
		errContains []string
	}{
		{
			name:        "exact match",
			input:       "my_project",
			allowPrefix: true,
			sessions:    sessions,
			want:        "my_project",
			wantReason:  "exact match",
		},
		{
			name:        "unique prefix",
			input:       "alp",
			allowPrefix: true,
			sessions:    sessions,
			want:        "alpha",
			wantReason:  "prefix match",
		},
		{
			name:        "ambiguous prefix",
			input:       "my",
			allowPrefix: true,
			sessions:    sessions,
			wantErr:     true,
			errContains: []string{"matches multiple sessions", "my_project", "my_test"},
		},
		{
			name:        "prefix disabled",
			input:       "alp",
			allowPrefix: false,
			sessions:    sessions,
			wantErr:     true,
			errContains: []string{"not found", "available"},
		},
		{
			name:        "no match",
			input:       "zzz",
			allowPrefix: true,
			sessions:    sessions,
			wantErr:     true,
			errContains: []string{"not found", "available", "alpha", "beta"},
		},
		{
			name:        "no sessions",
			input:       "anything",
			allowPrefix: true,
			sessions:    nil,
			wantErr:     true,
			errContains: []string{"no tmux sessions running"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolved, reason, err := resolveExplicitSessionName(tc.input, tc.sessions, tc.allowPrefix)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				for _, substr := range tc.errContains {
					if !strings.Contains(err.Error(), substr) {
						t.Fatalf("expected error to contain %q, got %q", substr, err.Error())
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resolved != tc.want {
				t.Fatalf("resolved %q, want %q", resolved, tc.want)
			}
			if reason != tc.wantReason {
				t.Fatalf("reason %q, want %q", reason, tc.wantReason)
			}
		})
	}
}

func TestTruncateWithEllipsis_Util(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		max   int
		want  string
	}{
		{"empty string", "", 10, ""},
		{"shorter than max", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncated", "hello world", 8, "hello..."},
		{"max zero", "hello", 0, ""},
		{"max negative", "hello", -1, ""},
		{"max 3 or less", "hello", 3, "hel"},
		{"max 1", "hello", 1, "h"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := truncateWithEllipsis(tc.input, tc.max)
			if got != tc.want {
				t.Errorf("truncateWithEllipsis(%q, %d) = %q, want %q", tc.input, tc.max, got, tc.want)
			}
		})
	}
}

func TestUniqueStrings_Util(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"empty", []string{}, []string{}},
		{"nil", nil, []string{}},
		{"no duplicates", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"with duplicates", []string{"b", "a", "b", "c", "a"}, []string{"a", "b", "c"}},
		{"whitespace trimmed", []string{" a ", " b ", " a "}, []string{"a", "b"}},
		{"empty strings filtered", []string{"a", "", "b", "  ", "c"}, []string{"a", "b", "c"}},
		{"sorted output", []string{"c", "a", "b"}, []string{"a", "b", "c"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := uniqueStrings(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("uniqueStrings(%v) len = %d, want %d", tc.input, len(got), len(tc.want))
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("uniqueStrings(%v)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestCountAgentStates_Util(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		assignments []ensemble.ModeAssignment
		wantReady   int
		wantPending int
		wantWorking int
	}{
		{
			name:        "empty",
			assignments: nil,
			wantReady:   0,
			wantPending: 0,
			wantWorking: 0,
		},
		{
			name: "mixed states",
			assignments: []ensemble.ModeAssignment{
				{Status: ensemble.AssignmentDone},
				{Status: ensemble.AssignmentDone},
				{Status: ensemble.AssignmentPending},
				{Status: ensemble.AssignmentInjecting},
				{Status: ensemble.AssignmentActive},
			},
			wantReady:   2,
			wantPending: 2,
			wantWorking: 1,
		},
		{
			name: "all done",
			assignments: []ensemble.ModeAssignment{
				{Status: ensemble.AssignmentDone},
				{Status: ensemble.AssignmentDone},
				{Status: ensemble.AssignmentDone},
			},
			wantReady:   3,
			wantPending: 0,
			wantWorking: 0,
		},
		{
			name: "error status not counted",
			assignments: []ensemble.ModeAssignment{
				{Status: ensemble.AssignmentError},
			},
			wantReady:   0,
			wantPending: 0,
			wantWorking: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			state := &ensemble.EnsembleSession{Assignments: tc.assignments}
			ready, pending, working := countAgentStates(state)
			if ready != tc.wantReady {
				t.Errorf("ready = %d, want %d", ready, tc.wantReady)
			}
			if pending != tc.wantPending {
				t.Errorf("pending = %d, want %d", pending, tc.wantPending)
			}
			if working != tc.wantWorking {
				t.Errorf("working = %d, want %d", working, tc.wantWorking)
			}
		})
	}
}

func TestEscapeShellQuotes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no quotes", "hello world", "hello world"},
		{"single double quote", `say "hello"`, `say \"hello\"`},
		{"multiple quotes", `"a" and "b"`, `\"a\" and \"b\"`},
		{"empty string", "", ""},
		{"only quotes", `""`, `\"\"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := escapeShellQuotes(tc.input)
			if got != tc.want {
				t.Errorf("escapeShellQuotes(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// Verify uniqueStrings output is truly sorted
func TestUniqueStrings_SortOrder(t *testing.T) {
	t.Parallel()

	input := []string{"zebra", "apple", "mango", "banana", "apple", "zebra"}
	got := uniqueStrings(input)

	if !sort.StringsAreSorted(got) {
		t.Errorf("uniqueStrings output not sorted: %v", got)
	}
}
