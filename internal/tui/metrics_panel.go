// Package tui provides terminal user interface components.
// metrics_panel.go implements the observability metrics dashboard for ensemble runs.
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dicklesworthstone/ntm/internal/ensemble"
	"github.com/Dicklesworthstone/ntm/internal/tui/layout"
	"github.com/Dicklesworthstone/ntm/internal/tui/theme"
)

// MetricsPanelMsg updates the metrics panel with new data.
type MetricsPanelMsg struct {
	Metrics *ensemble.ObservabilityMetrics
	Report  *ensemble.MetricsReport
	Err     error
}

// MetricsPanel displays ensemble observability metrics.
type MetricsPanel struct {
	Width  int
	Height int

	metrics *ensemble.ObservabilityMetrics
	report  *ensemble.MetricsReport
	err     error
}

// NewMetricsPanel creates a new ensemble metrics panel.
func NewMetricsPanel(width, height int) *MetricsPanel {
	return &MetricsPanel{
		Width:  width,
		Height: height,
	}
}

// SetMetrics updates the panel with new metrics data.
func (m *MetricsPanel) SetMetrics(metrics *ensemble.ObservabilityMetrics) {
	m.metrics = metrics
	if metrics != nil {
		m.report = metrics.GetReport()
	} else {
		m.report = nil
	}
	m.err = nil
}

// SetError sets an error state for the panel.
func (m *MetricsPanel) SetError(err error) {
	m.err = err
}

// Init implements tea.Model.
func (m *MetricsPanel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m *MetricsPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case MetricsPanelMsg:
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.metrics = msg.Metrics
			m.report = msg.Report
			m.err = nil
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}
	return m, nil
}

// View renders the metrics panel.
func (m *MetricsPanel) View() string {
	t := theme.Current()
	w, h := m.Width, m.Height

	if w <= 0 || h <= 0 {
		return ""
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Width(w - 2).
		Height(h - 2).
		Padding(0, 1)

	var content strings.Builder

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Lavender).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(t.Surface1).
		Width(w - 6).
		Align(lipgloss.Center).
		Render("Observability Metrics")
	content.WriteString(header + "\n\n")

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(t.Red)
		content.WriteString(errStyle.Render("Error: "+m.err.Error()) + "\n")
		return boxStyle.Render(content.String())
	}

	if m.report == nil {
		emptyStyle := lipgloss.NewStyle().Foreground(t.Subtext).Align(lipgloss.Center).Width(w - 6)
		content.WriteString(emptyStyle.Render("No metrics available yet"))
		return boxStyle.Render(content.String())
	}

	// Available height for content sections
	usedHeight := 4 // header + padding
	availHeight := h - usedHeight - 2

	// Distribute height among sections
	sectionHeight := availHeight / 4
	if sectionHeight < 3 {
		sectionHeight = 3
	}

	// Coverage section
	content.WriteString(m.renderCoverage(w-6, sectionHeight))

	// Velocity section
	content.WriteString(m.renderVelocity(w-6, sectionHeight))

	// Redundancy section
	content.WriteString(m.renderRedundancy(w-6, sectionHeight))

	// Conflict density section
	content.WriteString(m.renderConflicts(w-6, sectionHeight))

	// Suggestions footer
	if len(m.report.Suggestions) > 0 {
		content.WriteString(m.renderSuggestions(w - 6))
	}

	return boxStyle.Render(content.String())
}

// renderCoverage renders the category coverage section.
func (m *MetricsPanel) renderCoverage(width, height int) string {
	t := theme.Current()
	if m.report.Coverage == nil {
		return ""
	}

	var b strings.Builder

	// Section header
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Blue)
	b.WriteString(sectionStyle.Render("Coverage") + "\n")

	// Overall coverage bar
	overallPct := m.report.Coverage.Overall
	barWidth := clampInt(width-20, 10, 30)
	bar := renderMetricsBar(overallPct, barWidth, t.Green, t.Surface1)
	b.WriteString(fmt.Sprintf("  Overall: %s %.0f%%\n", bar, overallPct*100))

	// Blind spots (if any)
	if len(m.report.Coverage.BlindSpots) > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(t.Yellow)
		blindSpots := make([]string, 0, len(m.report.Coverage.BlindSpots))
		for _, bs := range m.report.Coverage.BlindSpots {
			blindSpots = append(blindSpots, bs.CategoryLetter())
		}
		b.WriteString(warnStyle.Render("  Blind spots: "+strings.Join(blindSpots, ", ")) + "\n")
	}

	b.WriteString("\n")
	return b.String()
}

// renderVelocity renders the findings velocity section.
func (m *MetricsPanel) renderVelocity(width, height int) string {
	t := theme.Current()
	if m.report.Velocity == nil {
		return ""
	}

	var b strings.Builder

	// Section header
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Mauve)
	b.WriteString(sectionStyle.Render("Findings Velocity") + "\n")

	// Overall velocity
	overall := m.report.Velocity.Overall
	b.WriteString(fmt.Sprintf("  Overall: %.2f findings/1K tokens\n", overall))

	// Per-mode summary (top performers and underperformers)
	if len(m.report.Velocity.HighPerformers) > 0 {
		goodStyle := lipgloss.NewStyle().Foreground(t.Green)
		hp := layout.TruncateWidthDefault(strings.Join(m.report.Velocity.HighPerformers, ", "), width-15)
		b.WriteString(goodStyle.Render("  High: "+hp) + "\n")
	}
	if len(m.report.Velocity.LowPerformers) > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(t.Yellow)
		lp := layout.TruncateWidthDefault(strings.Join(m.report.Velocity.LowPerformers, ", "), width-15)
		b.WriteString(warnStyle.Render("  Low: "+lp) + "\n")
	}

	b.WriteString("\n")
	return b.String()
}

// renderRedundancy renders the redundancy analysis section.
func (m *MetricsPanel) renderRedundancy(width, height int) string {
	t := theme.Current()
	if m.report.Redundancy == nil {
		return ""
	}

	var b strings.Builder

	// Section header
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Peach)
	b.WriteString(sectionStyle.Render("Redundancy") + "\n")

	// Overall redundancy score with interpretation
	score := m.report.Redundancy.OverallScore
	interpretation := interpretRedundancyScore(score)
	barWidth := clampInt(width-25, 8, 20)

	// Choose color based on score (lower is better for redundancy)
	barColor := t.Green
	if score >= 0.5 {
		barColor = t.Red
	} else if score >= 0.3 {
		barColor = t.Yellow
	}

	bar := renderMetricsBar(score, barWidth, barColor, t.Surface1)
	b.WriteString(fmt.Sprintf("  Score: %s %.0f%% (%s)\n", bar, score*100, interpretation))

	// High redundancy pairs
	highPairs := m.report.Redundancy.GetHighRedundancyPairs(0.5)
	if len(highPairs) > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(t.Yellow)
		pairStrs := make([]string, 0, min(3, len(highPairs)))
		for i, pair := range highPairs {
			if i >= 3 {
				break
			}
			pairStrs = append(pairStrs, fmt.Sprintf("%s↔%s", pair.ModeA, pair.ModeB))
		}
		b.WriteString(warnStyle.Render("  Similar: "+strings.Join(pairStrs, ", ")) + "\n")
	}

	b.WriteString("\n")
	return b.String()
}

// renderConflicts renders the conflict density section.
func (m *MetricsPanel) renderConflicts(width, height int) string {
	t := theme.Current()
	if m.report.ConflictDensity == nil {
		return ""
	}

	var b strings.Builder

	// Section header
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Pink)
	b.WriteString(sectionStyle.Render("Conflicts") + "\n")

	density := m.report.ConflictDensity
	b.WriteString(fmt.Sprintf("  Total: %d | Resolved: %d | Unresolved: %d\n",
		density.TotalConflicts, density.ResolvedConflicts, density.UnresolvedConflicts))

	if len(density.HighConflictPairs) > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(t.Yellow)
		pairs := layout.TruncateWidthDefault(strings.Join(density.HighConflictPairs, ", "), width-15)
		b.WriteString(warnStyle.Render("  Hot: "+pairs) + "\n")
	}

	// Show source
	srcStyle := lipgloss.NewStyle().Foreground(t.Overlay)
	b.WriteString(srcStyle.Render(fmt.Sprintf("  (via %s)", density.Source)) + "\n")

	b.WriteString("\n")
	return b.String()
}

// renderSuggestions renders action items.
func (m *MetricsPanel) renderSuggestions(width int) string {
	t := theme.Current()

	var b strings.Builder

	// Section header
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Teal)
	b.WriteString(sectionStyle.Render("Suggestions") + "\n")

	suggestionStyle := lipgloss.NewStyle().Foreground(t.Text)
	for i, suggestion := range m.report.Suggestions {
		if i >= 3 { // Limit displayed suggestions
			more := fmt.Sprintf("  ... and %d more", len(m.report.Suggestions)-3)
			b.WriteString(lipgloss.NewStyle().Foreground(t.Overlay).Render(more) + "\n")
			break
		}
		truncated := layout.TruncateWidthDefault(suggestion, width-4)
		b.WriteString(suggestionStyle.Render("  • "+truncated) + "\n")
	}

	return b.String()
}

// renderMetricsBar creates a simple progress bar for metrics display.
func renderMetricsBar(percent float64, width int, fillColor, emptyColor lipgloss.Color) string {
	if width <= 0 {
		return ""
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}

	filled := int(float64(width) * percent)
	if filled > width {
		filled = width
	}

	fillStyle := lipgloss.NewStyle().Foreground(fillColor)
	emptyStyle := lipgloss.NewStyle().Foreground(emptyColor)

	return fillStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", width-filled))
}

// interpretRedundancyScore returns a human-readable interpretation.
func interpretRedundancyScore(score float64) string {
	switch {
	case score >= 0.7:
		return "high"
	case score >= 0.5:
		return "moderate"
	case score >= 0.3:
		return "acceptable"
	default:
		return "good"
	}
}

// RenderCompactMetrics produces a compact one-line metrics summary.
func RenderCompactMetrics(report *ensemble.MetricsReport) string {
	if report == nil {
		return "No metrics"
	}

	t := theme.Current()
	var parts []string

	// Coverage
	if report.Coverage != nil {
		covStyle := lipgloss.NewStyle().Foreground(t.Blue)
		blindCount := len(report.Coverage.BlindSpots)
		if blindCount > 0 {
			parts = append(parts, covStyle.Render(fmt.Sprintf("Cov:%.0f%% (%d blind)", report.Coverage.Overall*100, blindCount)))
		} else {
			parts = append(parts, covStyle.Render(fmt.Sprintf("Cov:%.0f%%", report.Coverage.Overall*100)))
		}
	}

	// Velocity
	if report.Velocity != nil {
		velStyle := lipgloss.NewStyle().Foreground(t.Mauve)
		parts = append(parts, velStyle.Render(fmt.Sprintf("Vel:%.1f/1K", report.Velocity.Overall)))
	}

	// Redundancy
	if report.Redundancy != nil {
		redColor := t.Green
		if report.Redundancy.OverallScore >= 0.5 {
			redColor = t.Red
		} else if report.Redundancy.OverallScore >= 0.3 {
			redColor = t.Yellow
		}
		redStyle := lipgloss.NewStyle().Foreground(redColor)
		parts = append(parts, redStyle.Render(fmt.Sprintf("Red:%.0f%%", report.Redundancy.OverallScore*100)))
	}

	// Conflicts
	if report.ConflictDensity != nil && report.ConflictDensity.TotalConflicts > 0 {
		conflictStyle := lipgloss.NewStyle().Foreground(t.Pink)
		parts = append(parts, conflictStyle.Render(fmt.Sprintf("Conf:%d", report.ConflictDensity.TotalConflicts)))
	}

	if len(parts) == 0 {
		return "No metrics"
	}

	return strings.Join(parts, " │ ")
}

// RenderMetricsBadges produces individual badges for each metric.
func RenderMetricsBadges(report *ensemble.MetricsReport) []string {
	if report == nil {
		return nil
	}

	t := theme.Current()
	var badges []string

	// Coverage badge
	if report.Coverage != nil {
		label := fmt.Sprintf("COV %.0f%%", report.Coverage.Overall*100)
		color := t.Green
		if len(report.Coverage.BlindSpots) > 3 {
			color = t.Yellow
		}
		badges = append(badges, lipgloss.NewStyle().
			Background(color).
			Foreground(t.Base).
			Bold(true).
			Padding(0, 1).
			Render(label))
	}

	// Velocity badge
	if report.Velocity != nil {
		label := fmt.Sprintf("VEL %.1f", report.Velocity.Overall)
		color := t.Mauve
		if report.Velocity.Overall < 1.0 {
			color = t.Yellow
		}
		badges = append(badges, lipgloss.NewStyle().
			Background(color).
			Foreground(t.Base).
			Bold(true).
			Padding(0, 1).
			Render(label))
	}

	// Redundancy badge
	if report.Redundancy != nil {
		label := fmt.Sprintf("RED %.0f%%", report.Redundancy.OverallScore*100)
		color := t.Green
		if report.Redundancy.OverallScore >= 0.5 {
			color = t.Red
		} else if report.Redundancy.OverallScore >= 0.3 {
			color = t.Yellow
		}
		badges = append(badges, lipgloss.NewStyle().
			Background(color).
			Foreground(t.Base).
			Bold(true).
			Padding(0, 1).
			Render(label))
	}

	// Conflict badge
	if report.ConflictDensity != nil && report.ConflictDensity.UnresolvedConflicts > 0 {
		label := fmt.Sprintf("⚠ %d", report.ConflictDensity.UnresolvedConflicts)
		badges = append(badges, lipgloss.NewStyle().
			Background(t.Yellow).
			Foreground(t.Base).
			Bold(true).
			Padding(0, 1).
			Render(label))
	}

	return badges
}

// MetricsFromSession computes observability metrics from an ensemble session.
// This is a convenience function for dashboard integration.
func MetricsFromSession(
	session *ensemble.EnsembleSession,
	outputs []ensemble.ModeOutput,
	catalog *ensemble.ModeCatalog,
	budget *ensemble.BudgetState,
	auditReport *ensemble.AuditReport,
) *ensemble.ObservabilityMetrics {
	if session == nil {
		return nil
	}

	metrics := ensemble.NewObservabilityMetrics(catalog)
	_ = metrics.ComputeFromOutputs(outputs, budget, auditReport)
	return metrics
}

// MetricsUpdateCmd creates a tea.Cmd that computes and returns metrics.
func MetricsUpdateCmd(
	outputs []ensemble.ModeOutput,
	catalog *ensemble.ModeCatalog,
	budget *ensemble.BudgetState,
	auditReport *ensemble.AuditReport,
) tea.Cmd {
	return func() tea.Msg {
		metrics := ensemble.NewObservabilityMetrics(catalog)
		if err := metrics.ComputeFromOutputs(outputs, budget, auditReport); err != nil {
			return MetricsPanelMsg{Err: err}
		}
		return MetricsPanelMsg{
			Metrics: metrics,
			Report:  metrics.GetReport(),
		}
	}
}

// FormatMetricsTimestamp formats the computed-at timestamp for display.
func FormatMetricsTimestamp(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Format("15:04:05")
}
