package panels

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dicklesworthstone/ntm/internal/cost"
	"github.com/Dicklesworthstone/ntm/internal/tui/components"
	"github.com/Dicklesworthstone/ntm/internal/tui/layout"
	"github.com/Dicklesworthstone/ntm/internal/tui/theme"
)

type CostTrend int

const (
	CostTrendFlat CostTrend = iota
	CostTrendUp
	CostTrendDown
)

func (t CostTrend) Arrow() string {
	switch t {
	case CostTrendUp:
		return "↑"
	case CostTrendDown:
		return "↓"
	default:
		return "→"
	}
}

type CostAgentRow struct {
	PaneTitle    string
	Model        string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	Trend        CostTrend
}

type CostPanelData struct {
	Agents          []CostAgentRow
	SessionTotalUSD float64
	LastHourUSD     float64

	DailyBudgetUSD float64
	BudgetUsedUSD  float64
}

type CostPanel struct {
	PanelBase
	data  CostPanelData
	theme theme.Theme
	err   error
}

func costConfig() PanelConfig {
	return PanelConfig{
		ID:              "cost",
		Title:           "Cost Tracking",
		Priority:        PriorityNormal,
		RefreshInterval: 10 * time.Second,
		MinWidth:        30,
		MinHeight:       8,
		Collapsible:     true,
	}
}

func NewCostPanel() *CostPanel {
	return &CostPanel{
		PanelBase: NewPanelBase(costConfig()),
		theme:     theme.Current(),
	}
}

func (c *CostPanel) Init() tea.Cmd { return nil }

func (c *CostPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return c, nil
}

func (c *CostPanel) SetData(data CostPanelData, err error) {
	// Keep a stable ordering even if callers don't sort.
	if len(data.Agents) > 1 {
		sort.SliceStable(data.Agents, func(i, j int) bool {
			if data.Agents[i].CostUSD == data.Agents[j].CostUSD {
				return strings.ToLower(data.Agents[i].PaneTitle) < strings.ToLower(data.Agents[j].PaneTitle)
			}
			return data.Agents[i].CostUSD > data.Agents[j].CostUSD
		})
	}

	c.data = data
	c.err = err
	if err == nil {
		c.SetLastUpdate(time.Now())
	}
}

func (c *CostPanel) HasError() bool { return c.err != nil }

func (c *CostPanel) HasData() bool {
	if len(c.data.Agents) > 0 {
		return true
	}
	if c.data.SessionTotalUSD > 0 {
		return true
	}
	if c.data.DailyBudgetUSD > 0 {
		return true
	}
	return false
}

func (c *CostPanel) View() string {
	t := c.theme
	w, h := c.Width(), c.Height()

	borderColor := t.Surface1
	bgColor := t.Base
	if c.IsFocused() {
		borderColor = t.Primary
		bgColor = t.Surface0
	}

	if c.data.DailyBudgetUSD > 0 && c.data.BudgetUsedUSD > 0 {
		usedPct := (c.data.BudgetUsedUSD / c.data.DailyBudgetUSD) * 100.0
		if usedPct >= 95.0 || c.data.BudgetUsedUSD >= c.data.DailyBudgetUSD {
			borderColor = t.Red
		} else if usedPct >= 80.0 {
			borderColor = t.Yellow
		}
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Background(bgColor).
		Width(w-2).
		Height(h-2).
		Padding(0, 1)

	var content strings.Builder

	title := c.Config().Title
	if c.err != nil {
		errorBadge := lipgloss.NewStyle().
			Background(t.Red).
			Foreground(t.Base).
			Bold(true).
			Padding(0, 1).
			Render("!")
		title = title + " " + errorBadge
	} else if staleBadge := components.RenderStaleBadge(c.LastUpdate(), c.Config().RefreshInterval); staleBadge != "" {
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

	if c.err != nil {
		content.WriteString(components.ErrorState(c.err.Error(), "Waiting for refresh", w-4) + "\n")
		return boxStyle.Render(FitToHeight(content.String(), h-4))
	}

	if len(c.data.Agents) == 0 {
		content.WriteString("\n" + components.RenderEmptyState(components.EmptyStateOptions{
			Icon:        components.IconWaiting,
			Title:       "No cost data",
			Description: "Send prompts to record usage",
			Width:       w - 4,
			Centered:    true,
		}))
		return boxStyle.Render(FitToHeight(content.String(), h-4))
	}

	content.WriteString("\n")

	innerWidth := w - 4
	tableWidth := innerWidth
	if tableWidth < 0 {
		tableWidth = 0
	}

	// Decide how many columns can fit.
	showTokens := tableWidth >= 44
	showOut := tableWidth >= 36

	// Column widths (right-aligned for numbers).
	inW := 7  // "45.2K"
	outW := 7 // "23.1K"
	costW := 8
	trendW := 1
	sep := 1

	colsFixed := costW + trendW + sep + sep
	if showTokens {
		colsFixed += inW + sep
	}
	if showOut {
		colsFixed += outW + sep
	}

	nameW := tableWidth - colsFixed
	if nameW < 8 {
		nameW = 8
	}

	headerParts := []string{lipgloss.NewStyle().Foreground(t.Subtext).Render(padRight("Agent", nameW))}
	if showTokens {
		headerParts = append(headerParts, lipgloss.NewStyle().Foreground(t.Subtext).Render(padLeft("In", inW)))
	}
	if showOut {
		headerParts = append(headerParts, lipgloss.NewStyle().Foreground(t.Subtext).Render(padLeft("Out", outW)))
	}
	headerParts = append(headerParts, lipgloss.NewStyle().Foreground(t.Subtext).Render(padLeft("Cost", costW)))
	headerParts = append(headerParts, lipgloss.NewStyle().Foreground(t.Subtext).Render(padLeft("", trendW)))
	content.WriteString(strings.Join(headerParts, strings.Repeat(" ", sep)) + "\n")

	// Height budget: header + totals + footer; keep the table compact.
	availRows := h - 7
	if availRows < 1 {
		availRows = 1
	}

	for i, row := range c.data.Agents {
		if i >= availRows {
			remaining := len(c.data.Agents) - availRows
			if remaining > 0 {
				content.WriteString(lipgloss.NewStyle().Foreground(t.Overlay).Render(fmt.Sprintf("+%d more", remaining)) + "\n")
			}
			break
		}

		name := layout.TruncatePaneTitle(row.PaneTitle, nameW)
		name = padRight(name, nameW)

		parts := []string{lipgloss.NewStyle().Foreground(t.Text).Render(name)}
		if showTokens {
			parts = append(parts, lipgloss.NewStyle().Foreground(t.Overlay).Render(padLeft(formatTokenShort(row.InputTokens), inW)))
		}
		if showOut {
			parts = append(parts, lipgloss.NewStyle().Foreground(t.Overlay).Render(padLeft(formatTokenShort(row.OutputTokens), outW)))
		}

		parts = append(parts, lipgloss.NewStyle().Foreground(t.Text).Render(padLeft(cost.FormatCost(row.CostUSD), costW)))

		trendColor := t.Subtext
		if row.Trend == CostTrendUp {
			trendColor = t.Green
		} else if row.Trend == CostTrendDown {
			trendColor = t.Red
		}
		parts = append(parts, lipgloss.NewStyle().Foreground(trendColor).Render(padLeft(row.Trend.Arrow(), trendW)))

		content.WriteString(strings.Join(parts, strings.Repeat(" ", sep)) + "\n")
	}

	content.WriteString("\n")

	// Totals
	totalLine := fmt.Sprintf("Session Total: %s", cost.FormatCost(c.data.SessionTotalUSD))
	if c.data.LastHourUSD > 0 {
		totalLine += fmt.Sprintf("  (1h: %s)", cost.FormatCost(c.data.LastHourUSD))
	}
	content.WriteString(lipgloss.NewStyle().Foreground(t.Text).Bold(true).Render(totalLine) + "\n")

	if c.data.DailyBudgetUSD > 0 {
		remaining := c.data.DailyBudgetUSD - c.data.BudgetUsedUSD
		remainingStr := cost.FormatCost(remaining)

		budgetColor := t.Green
		if remaining <= 0 {
			budgetColor = t.Red
		} else {
			usedPct := (c.data.BudgetUsedUSD / c.data.DailyBudgetUSD) * 100.0
			if usedPct >= 95.0 {
				budgetColor = t.Red
			} else if usedPct >= 80.0 {
				budgetColor = t.Yellow
			}
		}

		budgetLine := fmt.Sprintf("Budget Left: %s  (limit: %s)", remainingStr, cost.FormatCost(c.data.DailyBudgetUSD))
		content.WriteString(lipgloss.NewStyle().Foreground(budgetColor).Bold(true).Render(budgetLine) + "\n")
	}

	if footer := components.RenderFreshnessFooter(components.FreshnessOptions{
		LastUpdate:      c.LastUpdate(),
		RefreshInterval: c.Config().RefreshInterval,
		Width:           w - 4,
	}); footer != "" {
		content.WriteString(footer + "\n")
	}

	return boxStyle.Render(FitToHeight(content.String(), h-4))
}

func formatTokenShort(tokens int) string {
	if tokens <= 0 {
		return "0"
	}
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	if tokens < 1000000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1000.0)
	}
	return fmt.Sprintf("%.1fM", float64(tokens)/1000000.0)
}

func padRight(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func padLeft(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}
