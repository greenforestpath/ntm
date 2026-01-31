package panels

import (
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/ensemble"
)

func TestMinInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"a smaller", 1, 5, 1},
		{"b smaller", 10, 3, 3},
		{"equal", 7, 7, 7},
		{"negative a", -5, 3, -5},
		{"negative b", 3, -5, -5},
		{"both negative", -3, -7, -7},
		{"zero and positive", 0, 5, 0},
		{"zero and negative", 0, -5, -5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := minInt(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("minInt(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestMaxInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"a larger", 10, 5, 10},
		{"b larger", 3, 8, 8},
		{"equal", 7, 7, 7},
		{"negative a", -5, 3, 3},
		{"negative b", 3, -5, 3},
		{"both negative", -3, -7, -3},
		{"zero and positive", 0, 5, 5},
		{"zero and negative", 0, -5, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := maxInt(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("maxInt(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestShortenPaneName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"with double underscore", "session__cc_1_opus", "cc_1_opus"},
		{"multiple double underscores", "long__path__final_part", "final_part"},
		{"no double underscore", "simple_name", "simple_name"},
		{"empty string", "", ""},
		{"double underscore at end", "name__", "name__"},
		{"just double underscore", "__", "__"},
		{"double underscore with one char", "__x", "x"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shortenPaneName(tc.input)
			if got != tc.want {
				t.Errorf("shortenPaneName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestAssignmentStatusIcon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status ensemble.AssignmentStatus
		want   string
	}{
		{"active", ensemble.AssignmentActive, "●"},
		{"injecting", ensemble.AssignmentInjecting, "◐"},
		{"pending", ensemble.AssignmentPending, "○"},
		{"done", ensemble.AssignmentDone, "✓"},
		{"error", ensemble.AssignmentError, "✗"},
		{"unknown", ensemble.AssignmentStatus("unknown"), "•"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assignmentStatusIcon(tc.status)
			if got != tc.want {
				t.Errorf("assignmentStatusIcon(%q) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

func TestAssignmentProgress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status ensemble.AssignmentStatus
		want   float64
	}{
		{"pending", ensemble.AssignmentPending, 0.05},
		{"injecting", ensemble.AssignmentInjecting, 0.25},
		{"active", ensemble.AssignmentActive, 0.6},
		{"done", ensemble.AssignmentDone, 1.0},
		{"error", ensemble.AssignmentError, 1.0},
		{"unknown", ensemble.AssignmentStatus("unknown"), 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := assignmentProgress(tc.status)
			if got != tc.want {
				t.Errorf("assignmentProgress(%q) = %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}

func TestSummarizeAssignmentStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		assignments             []ensemble.ModeAssignment
		wantActive, wantDone    int
		wantPending             int
	}{
		{
			name:        "empty",
			assignments: []ensemble.ModeAssignment{},
			wantActive:  0, wantDone: 0, wantPending: 0,
		},
		{
			name: "all active",
			assignments: []ensemble.ModeAssignment{
				{Status: ensemble.AssignmentActive},
				{Status: ensemble.AssignmentActive},
			},
			wantActive: 2, wantDone: 0, wantPending: 0,
		},
		{
			name: "mixed statuses",
			assignments: []ensemble.ModeAssignment{
				{Status: ensemble.AssignmentActive},
				{Status: ensemble.AssignmentInjecting},
				{Status: ensemble.AssignmentDone},
				{Status: ensemble.AssignmentPending},
				{Status: ensemble.AssignmentError},
			},
			wantActive: 2, wantDone: 1, wantPending: 1,
		},
		{
			name: "all done",
			assignments: []ensemble.ModeAssignment{
				{Status: ensemble.AssignmentDone},
				{Status: ensemble.AssignmentDone},
				{Status: ensemble.AssignmentDone},
			},
			wantActive: 0, wantDone: 3, wantPending: 0,
		},
		{
			name: "all pending",
			assignments: []ensemble.ModeAssignment{
				{Status: ensemble.AssignmentPending},
				{Status: ensemble.AssignmentPending},
			},
			wantActive: 0, wantDone: 0, wantPending: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			active, done, pending := summarizeAssignmentStatus(tc.assignments)
			if active != tc.wantActive {
				t.Errorf("active = %d, want %d", active, tc.wantActive)
			}
			if done != tc.wantDone {
				t.Errorf("done = %d, want %d", done, tc.wantDone)
			}
			if pending != tc.wantPending {
				t.Errorf("pending = %d, want %d", pending, tc.wantPending)
			}
		})
	}
}

func TestEnsemblePanel_FormatTimeAgo(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{"just now", fixedNow.Add(-30 * time.Second), "now"},
		{"5 minutes ago", fixedNow.Add(-5 * time.Minute), "5m"},
		{"59 minutes ago", fixedNow.Add(-59 * time.Minute), "59m"},
		{"1 hour ago", fixedNow.Add(-1 * time.Hour), "1h"},
		{"3 hours ago", fixedNow.Add(-3 * time.Hour), "3h"},
		{"23 hours ago", fixedNow.Add(-23 * time.Hour), "23h"},
		{"1 day ago", fixedNow.Add(-25 * time.Hour), "1d"},
		{"5 days ago", fixedNow.Add(-5 * 24 * time.Hour), "5d"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &EnsemblePanel{
				now: func() time.Time { return fixedNow },
			}
			got := p.formatTimeAgo(tc.time)
			if got != tc.want {
				t.Errorf("formatTimeAgo(%v) = %q, want %q", tc.time, got, tc.want)
			}
		})
	}
}

func TestEnsemblePanel_FormatTimeAgo_NilNowFunc(t *testing.T) {
	t.Parallel()

	p := &EnsemblePanel{
		now: nil,
	}

	// With nil now func, it should use time.Now
	// Just verify it doesn't panic and returns something
	result := p.formatTimeAgo(time.Now().Add(-5 * time.Minute))
	if result == "" {
		t.Error("formatTimeAgo with nil now func returned empty string")
	}
}

func TestEnsembleConfig(t *testing.T) {
	t.Parallel()

	cfg := ensembleConfig()

	if cfg.ID != "ensemble" {
		t.Errorf("ID = %q, want %q", cfg.ID, "ensemble")
	}
	if cfg.Title != "Reasoning Ensemble" {
		t.Errorf("Title = %q, want %q", cfg.Title, "Reasoning Ensemble")
	}
	if cfg.Priority != PriorityHigh {
		t.Errorf("Priority = %v, want %v", cfg.Priority, PriorityHigh)
	}
	if cfg.RefreshInterval != 5*time.Second {
		t.Errorf("RefreshInterval = %v, want %v", cfg.RefreshInterval, 5*time.Second)
	}
	if cfg.MinWidth != 30 {
		t.Errorf("MinWidth = %d, want %d", cfg.MinWidth, 30)
	}
	if cfg.MinHeight != 8 {
		t.Errorf("MinHeight = %d, want %d", cfg.MinHeight, 8)
	}
	if !cfg.Collapsible {
		t.Error("Collapsible should be true")
	}
}

func TestNewEnsemblePanel(t *testing.T) {
	t.Parallel()

	p := NewEnsemblePanel()

	if p == nil {
		t.Fatal("NewEnsemblePanel returned nil")
	}
	if p.session != nil {
		t.Error("new panel should have nil session")
	}
	if p.catalog != nil {
		t.Error("new panel should have nil catalog")
	}
	if p.err != nil {
		t.Error("new panel should have nil error")
	}
	if p.now == nil {
		t.Error("new panel should have non-nil now function")
	}

	// Verify config was set
	cfg := p.Config()
	if cfg.ID != "ensemble" {
		t.Errorf("Config ID = %q, want %q", cfg.ID, "ensemble")
	}
}
