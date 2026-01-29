package ensemble

import (
	"math"
	"strings"
	"testing"
)

func TestVelocityTracker_DedupFindings(t *testing.T) {
	tracker := NewVelocityTracker()
	output := ModeOutput{
		ModeID: "formal",
		TopFindings: []Finding{
			{Finding: "Cache results.", Impact: ImpactLow, Confidence: 0.5, EvidencePointer: "file.go:10"},
			{Finding: "cache results", Impact: ImpactLow, Confidence: 0.5, EvidencePointer: "file.go:10"},
			{Finding: "Cache results", Impact: ImpactLow, Confidence: 0.5},
		},
	}

	tracker.RecordOutput("formal", output, 1000)

	if len(tracker.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(tracker.Entries))
	}
	entry := tracker.Entries[0]
	if entry.FindingsCount != 3 {
		t.Errorf("FindingsCount = %d, want 3", entry.FindingsCount)
	}
	if entry.UniqueFindings != 2 {
		t.Errorf("UniqueFindings = %d, want 2", entry.UniqueFindings)
	}
	if entry.Velocity != 2 {
		t.Errorf("Velocity = %.2f, want 2.00", entry.Velocity)
	}
}

func TestVelocityTracker_ZeroTokens(t *testing.T) {
	tracker := NewVelocityTracker()
	output := ModeOutput{
		ModeID: "formal",
		TopFindings: []Finding{
			{Finding: "Needs refactor", Impact: ImpactMedium, Confidence: 0.6},
		},
	}

	tracker.RecordOutput("formal", output, 0)
	report := tracker.CalculateVelocity()

	if len(tracker.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(tracker.Entries))
	}
	if tracker.Entries[0].Velocity != 0 {
		t.Errorf("Velocity = %.2f, want 0", tracker.Entries[0].Velocity)
	}
	if report.Overall != 0 {
		t.Errorf("Overall velocity = %.2f, want 0", report.Overall)
	}
}

func TestVelocityTracker_OverallAndLabels(t *testing.T) {
	tracker := NewVelocityTracker()
	outputA := ModeOutput{
		ModeID: "mode-a",
		TopFindings: []Finding{
			{Finding: "Finding A1", Impact: ImpactLow, Confidence: 0.5},
		},
	}
	outputB := ModeOutput{
		ModeID: "mode-b",
		TopFindings: []Finding{
			{Finding: "Finding B1", Impact: ImpactLow, Confidence: 0.5},
			{Finding: "Finding B2", Impact: ImpactLow, Confidence: 0.5},
		},
	}

	tracker.RecordOutput("mode-a", outputA, 2000)
	tracker.RecordOutput("mode-b", outputB, 1000)

	report := tracker.CalculateVelocity()
	expectedOverall := 1.0
	if math.Abs(report.Overall-expectedOverall) > 0.0001 {
		t.Errorf("Overall velocity = %.2f, want %.2f", report.Overall, expectedOverall)
	}
	if !containsStringValue(report.LowPerformers, "mode-a") {
		t.Errorf("expected mode-a to be low performer, got %v", report.LowPerformers)
	}
	if !containsStringValue(report.HighPerformers, "mode-b") {
		t.Errorf("expected mode-b to be high performer, got %v", report.HighPerformers)
	}
	if len(report.Suggestions) == 0 || !strings.Contains(report.Suggestions[0], "mode-a") {
		t.Errorf("expected suggestion for mode-a, got %v", report.Suggestions)
	}
}

func TestUniqueFindingKeys(t *testing.T) {
	t.Parallel()

	t.Run("distinct findings", func(t *testing.T) {
		t.Parallel()
		findings := []Finding{
			{Finding: "Issue A", EvidencePointer: "file.go:10"},
			{Finding: "Issue B", EvidencePointer: "file.go:20"},
		}
		keys := uniqueFindingKeys(findings)
		if len(keys) != 2 {
			t.Errorf("expected 2 unique keys, got %d", len(keys))
		}
	})

	t.Run("duplicate findings", func(t *testing.T) {
		t.Parallel()
		findings := []Finding{
			{Finding: "Issue A", EvidencePointer: "file.go:10"},
			{Finding: "Issue A", EvidencePointer: "file.go:10"},
		}
		keys := uniqueFindingKeys(findings)
		if len(keys) != 1 {
			t.Errorf("expected 1 unique key, got %d", len(keys))
		}
	})

	t.Run("empty findings", func(t *testing.T) {
		t.Parallel()
		keys := uniqueFindingKeys(nil)
		if len(keys) != 0 {
			t.Errorf("expected 0 keys, got %d", len(keys))
		}
	})

	t.Run("empty finding text skipped", func(t *testing.T) {
		t.Parallel()
		findings := []Finding{
			{Finding: "", EvidencePointer: ""},
			{Finding: "Real finding"},
		}
		keys := uniqueFindingKeys(findings)
		if len(keys) != 1 {
			t.Errorf("expected 1 key (empty skipped), got %d", len(keys))
		}
	})
}

func TestAverageVelocity(t *testing.T) {
	t.Parallel()

	t.Run("normal entries", func(t *testing.T) {
		t.Parallel()
		entries := []VelocityEntry{
			{Velocity: 2.0, TokensSpent: 1000},
			{Velocity: 4.0, TokensSpent: 2000},
		}
		got := averageVelocity(entries)
		if got != 3.0 {
			t.Errorf("averageVelocity = %f, want 3.0", got)
		}
	})

	t.Run("zero tokens skipped", func(t *testing.T) {
		t.Parallel()
		entries := []VelocityEntry{
			{Velocity: 2.0, TokensSpent: 1000},
			{Velocity: 99.0, TokensSpent: 0}, // should be skipped
		}
		got := averageVelocity(entries)
		if got != 2.0 {
			t.Errorf("averageVelocity = %f, want 2.0", got)
		}
	})

	t.Run("empty entries", func(t *testing.T) {
		t.Parallel()
		got := averageVelocity(nil)
		if got != 0 {
			t.Errorf("averageVelocity(nil) = %f, want 0", got)
		}
	})

	t.Run("all zero tokens", func(t *testing.T) {
		t.Parallel()
		entries := []VelocityEntry{
			{Velocity: 1.0, TokensSpent: 0},
		}
		got := averageVelocity(entries)
		if got != 0 {
			t.Errorf("averageVelocity = %f, want 0", got)
		}
	})
}

func TestVelocityLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   float64
		average float64
		want    string
	}{
		{"above average", 5.0, 3.0, "HIGH"},
		{"below threshold", 0.5, 3.0, "LOW"},
		{"normal", 2.0, 3.0, ""},
		{"equal to average", 3.0, 3.0, ""},
		{"at threshold", 1.0, 3.0, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := velocityLabel(tc.value, tc.average)
			if got != tc.want {
				t.Errorf("velocityLabel(%f, %f) = %q, want %q", tc.value, tc.average, got, tc.want)
			}
		})
	}
}

func TestDisplayName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		entry VelocityEntry
		want  string
	}{
		{"has mode name", VelocityEntry{ModeName: "Security Scanner", ModeID: "mode-a"}, "Security Scanner"},
		{"no mode name", VelocityEntry{ModeID: "mode-b"}, "mode-b"},
		{"both empty", VelocityEntry{}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := displayName(tc.entry)
			if got != tc.want {
				t.Errorf("displayName = %q, want %q", got, tc.want)
			}
		})
	}
}

func containsStringValue(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
