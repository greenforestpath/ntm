package panels

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dicklesworthstone/ntm/internal/scoring"
	"github.com/Dicklesworthstone/ntm/internal/tui/components"
	"github.com/Dicklesworthstone/ntm/internal/tui/theme"
)

const (
	// Thresholds for score coloring
	scoreHighThreshold    = 0.8
	scoreMediumThreshold  = 0.6
	scoreLowThreshold     = 0.4
	minSamplesForDisplay  = 1
	trendChangeThreshold  = 5.0 // Percent change to consider significant
)

// EffectivenessData holds the data for the effectiveness panel.
type EffectivenessData struct {
	// Summaries contains per-agent effectiveness summaries.
	Summaries []*scoring.AgentSummary

	// SessionName is the current session being monitored.
	SessionName string

	// WindowDays is the trend analysis window in days.
	WindowDays int
}

// EffectivenessPanel displays agent effectiveness metrics.
type EffectivenessPanel struct {
	PanelBase
	data     EffectivenessData
	err      error
	expanded map[string]bool
	theme    theme.Theme
}

// effectivenessConfig returns the configuration for the effectiveness panel.
func effectivenessConfig() PanelConfig {
	return PanelConfig{
		ID:              "effectiveness",
		Title:           "Agent Effectiveness",
		Priority:        PriorityNormal,
		RefreshInterval: 15 * time.Second,
		MinWidth:        35,
		MinHeight:       10,
		Collapsible:     true,
	}
}

// NewEffectivenessPanel creates a new effectiveness panel.
func NewEffectivenessPanel() *EffectivenessPanel {
	return &EffectivenessPanel{
		PanelBase: NewPanelBase(effectivenessConfig()),
		theme:     theme.Current(),
		expanded:  make(map[string]bool),
	}
}

// Init implements tea.Model.
func (e *EffectivenessPanel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (e *EffectivenessPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !e.IsFocused() {
			return e, nil
		}
		switch strings.ToLower(msg.String()) {
		case "d":
			e.toggleExpanded("details")
		}
	}
	return e, nil
}

// SetData updates the panel data.
func (e *EffectivenessPanel) SetData(data EffectivenessData, err error) {
	// Sort by overall score (highest first), then by agent type for stable ordering
	if len(data.Summaries) > 1 {
		sort.SliceStable(data.Summaries, func(i, j int) bool {
			if data.Summaries[i].AvgOverall == data.Summaries[j].AvgOverall {
				return data.Summaries[i].AgentType < data.Summaries[j].AgentType
			}
			return data.Summaries[i].AvgOverall > data.Summaries[j].AvgOverall
		})
	}

	e.data = data
	e.err = err
	if err == nil {
		e.SetLastUpdate(time.Now())
	}
}

// HasError returns true if there's an active error.
func (e *EffectivenessPanel) HasError() bool {
	return e.err != nil
}

// HasData returns true if the panel has displayable data.
func (e *EffectivenessPanel) HasData() bool {
	return len(e.data.Summaries) > 0
}

// Keybindings returns effectiveness panel specific shortcuts.
func (e *EffectivenessPanel) Keybindings() []Keybinding {
	return []Keybinding{
		{
			Key:         key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "toggle details")),
			Description: "Toggle detailed breakdown",
			Action:      "toggle_details",
		},
	}
}

// View renders the panel.
func (e *EffectivenessPanel) View() string {
	t := e.theme
	w, h := e.Width(), e.Height()

	borderColor := t.Surface1
	bgColor := t.Base
	if e.IsFocused() {
		borderColor = t.Primary
		bgColor = t.Surface0
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Background(bgColor).
		Width(w-2).
		Height(h-2).
		Padding(0, 1)

	var content strings.Builder

	title := e.Config().Title
	if e.err != nil {
		errorBadge := lipgloss.NewStyle().
			Background(t.Red).
			Foreground(t.Base).
			Bold(true).
			Padding(0, 1).
			Render("Error")
		title = title + " " + errorBadge
	} else if staleBadge := components.RenderStaleBadge(e.LastUpdate(), e.Config().RefreshInterval); staleBadge != "" {
		title = title + " " + staleBadge
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Lavender).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(t.Surface1).
		Width(w - 4).
		Align(lipgloss.Center)

	content.WriteString(headerStyle.Render(title) + "\n")

	if e.err != nil {
		content.WriteString(components.ErrorState(e.err.Error(), "Press r to retry", w-4) + "\n")
	}

	// Render content
	if len(e.data.Summaries) == 0 {
		content.WriteString(e.renderEmptyState())
	} else {
		content.WriteString(e.renderSummaries())
	}

	if footer := components.RenderFreshnessFooter(components.FreshnessOptions{
		LastUpdate:      e.LastUpdate(),
		RefreshInterval: e.Config().RefreshInterval,
		Width:           w - 4,
	}); footer != "" {
		content.WriteString(footer + "\n")
	}

	return boxStyle.Render(FitToHeight(content.String(), h-4))
}

func (e *EffectivenessPanel) renderEmptyState() string {
	return "No effectiveness data yet.\nData appears as agents complete tasks."
}

func (e *EffectivenessPanel) renderSummaries() string {
	t := e.theme
	var lines []string

	for _, summary := range e.data.Summaries {
		// Score bar and value
		scoreBar := e.renderScoreBar(summary.AvgOverall)
		scoreColor := e.scoreColor(summary.AvgOverall)
		scoreStyle := lipgloss.NewStyle().Foreground(scoreColor)

		// Trend arrow
		trendArrow := e.trendArrow(summary.Trend)

		// Agent row: type | score bar | score value | trend
		agentLine := fmt.Sprintf("%-10s %s %s %s",
			truncate(summary.AgentType, 10),
			scoreBar,
			scoreStyle.Render(fmt.Sprintf("%.0f%%", summary.AvgOverall*100)),
			trendArrow,
		)
		lines = append(lines, agentLine)

		// Show samples count (helps understand data confidence)
		samplesLine := fmt.Sprintf("           (%d samples)", summary.TotalScores)
		dimStyle := lipgloss.NewStyle().Foreground(t.Surface2)
		lines = append(lines, dimStyle.Render(samplesLine))

		// Expanded details
		if e.expanded["details"] {
			lines = append(lines, e.renderAgentDetails(summary)...)
		}
	}

	// Recommendations section
	if recs := e.renderRecommendations(); recs != "" {
		lines = append(lines, "", recs)
	}

	return strings.Join(lines, "\n")
}

func (e *EffectivenessPanel) renderScoreBar(score float64) string {
	width := 10
	filled := int(score * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}

	t := e.theme
	filledStyle := lipgloss.NewStyle().Foreground(e.scoreColor(score))
	emptyStyle := lipgloss.NewStyle().Foreground(t.Surface1)

	return "[" + filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", width-filled)) + "]"
}

func (e *EffectivenessPanel) scoreColor(score float64) lipgloss.Color {
	t := e.theme
	switch {
	case score >= scoreHighThreshold:
		return t.Green
	case score >= scoreMediumThreshold:
		return t.Yellow
	case score >= scoreLowThreshold:
		return t.Peach
	default:
		return t.Red
	}
}

func (e *EffectivenessPanel) trendArrow(trend *scoring.TrendAnalysis) string {
	t := e.theme
	if trend == nil || trend.SampleCount < scoring.MinSamplesForTrend {
		dimStyle := lipgloss.NewStyle().Foreground(t.Surface2)
		return dimStyle.Render("~") // Insufficient data
	}

	switch trend.Trend {
	case scoring.TrendImproving:
		return lipgloss.NewStyle().Foreground(t.Green).Render("↑")
	case scoring.TrendDeclining:
		return lipgloss.NewStyle().Foreground(t.Red).Render("↓")
	case scoring.TrendStable:
		return lipgloss.NewStyle().Foreground(t.Surface2).Render("→")
	default:
		return lipgloss.NewStyle().Foreground(t.Surface2).Render("~")
	}
}

func (e *EffectivenessPanel) renderAgentDetails(summary *scoring.AgentSummary) []string {
	t := e.theme
	dimStyle := lipgloss.NewStyle().Foreground(t.Subtext)

	details := []string{
		dimStyle.Render(fmt.Sprintf("           Completion: %.0f%% | Quality: %.0f%% | Efficiency: %.0f%%",
			summary.AvgCompletion*100,
			summary.AvgQuality*100,
			summary.AvgEfficiency*100)),
	}

	if summary.Trend != nil && summary.Trend.SampleCount >= scoring.MinSamplesForTrend {
		changeStr := fmt.Sprintf("%+.1f%%", summary.Trend.ChangePercent)
		details = append(details, dimStyle.Render(
			fmt.Sprintf("           Trend: %s (%.0f%% → %.0f%%)",
				changeStr,
				summary.Trend.EarlierAvg*100,
				summary.Trend.RecentAvg*100)))
	}

	return details
}

func (e *EffectivenessPanel) renderRecommendations() string {
	if len(e.data.Summaries) == 0 {
		return ""
	}

	t := e.theme
	var recs []string

	// Find underperforming agents
	for _, summary := range e.data.Summaries {
		if summary.AvgOverall < scoreLowThreshold && summary.TotalScores >= minSamplesForDisplay {
			recs = append(recs, fmt.Sprintf("  • %s performing below threshold", summary.AgentType))
		}
		if summary.Trend != nil && summary.Trend.Trend == scoring.TrendDeclining {
			recs = append(recs, fmt.Sprintf("  • %s trending downward", summary.AgentType))
		}
	}

	if len(recs) == 0 {
		return ""
	}

	warnStyle := lipgloss.NewStyle().Foreground(t.Yellow).Bold(true)
	recStyle := lipgloss.NewStyle().Foreground(t.Subtext)

	return warnStyle.Render("Recommendations:") + "\n" + recStyle.Render(strings.Join(recs, "\n"))
}

func (e *EffectivenessPanel) toggleExpanded(section string) {
	if e.expanded == nil {
		e.expanded = make(map[string]bool)
	}
	e.expanded[section] = !e.expanded[section]
}

// truncate shortens a string to maxLen, adding ellipsis if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
