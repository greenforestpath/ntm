package ensemble

import (
	"fmt"
	"log/slog"
	"strings"
)

const defaultLowVelocityThreshold = 1.0

// VelocityTracker tracks findings velocity per mode.
type VelocityTracker struct {
	Entries        []VelocityEntry
	uniqueFindings map[string]struct{}
}

// VelocityEntry captures per-mode velocity stats.
type VelocityEntry struct {
	ModeID         string
	ModeName       string
	TokensSpent    int
	FindingsCount  int
	UniqueFindings int
	Velocity       float64 // findings per 1k tokens
}

// VelocityReport summarizes velocity across modes.
type VelocityReport struct {
	Overall        float64
	PerMode        []VelocityEntry
	HighPerformers []string
	LowPerformers  []string
	Suggestions    []string
}

// NewVelocityTracker initializes a velocity tracker.
func NewVelocityTracker() *VelocityTracker {
	return &VelocityTracker{
		Entries:        []VelocityEntry{},
		uniqueFindings: make(map[string]struct{}),
	}
}

// RecordOutput records a mode output and tokens spent for velocity tracking.
func (v *VelocityTracker) RecordOutput(modeID string, output ModeOutput, tokens int) {
	if v == nil {
		return
	}
	if modeID == "" {
		modeID = output.ModeID
	}
	modeName := modeID
	if tokens < 0 {
		tokens = 0
	}

	uniqueKeys := uniqueFindingKeys(output.TopFindings)
	for key := range uniqueKeys {
		v.uniqueFindings[key] = struct{}{}
	}

	uniqueCount := len(uniqueKeys)
	findingsCount := len(output.TopFindings)
	velocity := 0.0
	if tokens > 0 {
		velocity = float64(uniqueCount) / float64(tokens) * 1000
	} else {
		slog.Warn("findings velocity tokens missing",
			"mode_id", modeID,
			"tokens_spent", tokens,
		)
	}

	v.Entries = append(v.Entries, VelocityEntry{
		ModeID:         modeID,
		ModeName:       modeName,
		TokensSpent:    tokens,
		FindingsCount:  findingsCount,
		UniqueFindings: uniqueCount,
		Velocity:       velocity,
	})

	slog.Debug("findings velocity recorded",
		"mode_id", modeID,
		"tokens_spent", tokens,
		"findings", findingsCount,
		"unique_findings", uniqueCount,
		"velocity", fmt.Sprintf("%.2f", velocity),
	)
}

// CalculateVelocity computes velocity metrics and labels performers.
func (v *VelocityTracker) CalculateVelocity() *VelocityReport {
	report := &VelocityReport{
		PerMode: []VelocityEntry{},
	}
	if v == nil {
		return report
	}

	report.PerMode = append(report.PerMode, v.Entries...)

	totalTokens := 0
	for _, entry := range v.Entries {
		totalTokens += entry.TokensSpent
	}
	if totalTokens > 0 {
		report.Overall = float64(len(v.uniqueFindings)) / float64(totalTokens) * 1000
	}

	average := averageVelocity(v.Entries)
	highPerformers := make([]string, 0)
	lowPerformers := make([]string, 0)
	suggestions := make([]string, 0)

	for _, entry := range v.Entries {
		if entry.Velocity > average {
			highPerformers = append(highPerformers, entry.ModeID)
		}
		if entry.Velocity < defaultLowVelocityThreshold {
			lowPerformers = append(lowPerformers, entry.ModeID)
			suggestions = append(suggestions, fmt.Sprintf("%s underperforming, consider early stop", displayName(entry)))
			slog.Info("findings velocity below threshold",
				"mode_id", entry.ModeID,
				"velocity", fmt.Sprintf("%.2f", entry.Velocity),
				"threshold", fmt.Sprintf("%.2f", defaultLowVelocityThreshold),
			)
		}
	}

	report.HighPerformers = highPerformers
	report.LowPerformers = lowPerformers
	report.Suggestions = suggestions

	return report
}

// Render produces a human-readable summary of findings velocity.
func (v *VelocityTracker) Render() string {
	if v == nil {
		return "No velocity data available"
	}
	report := v.CalculateVelocity()
	if report == nil {
		return "No velocity data available"
	}

	var b strings.Builder
	b.WriteString("Findings Velocity:\n")
	fmt.Fprintf(&b, "Overall: %.2f findings / 1k tokens\n\n", report.Overall)

	if len(report.PerMode) > 0 {
		b.WriteString("Per Mode:\n")
		average := averageVelocity(report.PerMode)
		for _, entry := range report.PerMode {
			label := velocityLabel(entry.Velocity, average)
			if label != "" {
				fmt.Fprintf(&b, "%-22s %.2f findings/1k (%s)\n", displayName(entry), entry.Velocity, label)
			} else {
				fmt.Fprintf(&b, "%-22s %.2f findings/1k\n", displayName(entry), entry.Velocity)
			}
		}
	}

	if len(report.Suggestions) > 0 {
		b.WriteString("\n")
		for _, suggestion := range report.Suggestions {
			fmt.Fprintf(&b, "Suggestion: %s\n", suggestion)
		}
	}

	return b.String()
}

func uniqueFindingKeys(findings []Finding) map[string]struct{} {
	keys := make(map[string]struct{}, len(findings))
	for _, finding := range findings {
		key := normalizeFinding(finding)
		if key == "" {
			continue
		}
		keys[key] = struct{}{}
	}
	return keys
}

func averageVelocity(entries []VelocityEntry) float64 {
	sum := 0.0
	count := 0
	for _, entry := range entries {
		if entry.TokensSpent == 0 {
			continue
		}
		sum += entry.Velocity
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func velocityLabel(value, average float64) string {
	if value > average {
		return "HIGH"
	}
	if value < defaultLowVelocityThreshold {
		return "LOW"
	}
	return ""
}

func displayName(entry VelocityEntry) string {
	if entry.ModeName != "" {
		return entry.ModeName
	}
	return entry.ModeID
}
