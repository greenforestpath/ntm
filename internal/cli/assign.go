package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/Dicklesworthstone/ntm/internal/bv"
	"github.com/Dicklesworthstone/ntm/internal/robot"
	"github.com/Dicklesworthstone/ntm/internal/tmux"
	"github.com/Dicklesworthstone/ntm/internal/tui/theme"
)

var (
	assignAuto     bool
	assignStrategy string
	assignBeads    string
)

func newAssignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assign [session]",
		Short: "Intelligently assign work to agents based on BV triage",
		Long: `Analyze ready work from BV and recommend or execute task-to-agent assignments.

This command queries BV for prioritized ready work and matches tasks to idle agents
based on agent type strengths and the selected strategy.

Strategies:
  balanced   - Balance workload across agents (default)
  speed      - Prioritize quick task completion
  quality    - Prioritize agent-task match quality
  dependency - Prioritize unblocking downstream work

Examples:
  ntm assign myproject                         # Show assignment recommendations
  ntm assign myproject --auto                  # Execute assignments without confirmation
  ntm assign myproject --strategy=quality      # Use quality-focused matching
  ntm assign myproject --beads=bd-123,bd-456   # Assign specific beads only
  ntm assign myproject --json                  # Output as JSON`,
		Args: cobra.MaximumNArgs(1),
		RunE: runAssign,
	}

	cmd.Flags().BoolVar(&assignAuto, "auto", false, "Execute assignments without confirmation")
	cmd.Flags().StringVar(&assignStrategy, "strategy", "balanced", "Assignment strategy: balanced, speed, quality, dependency")
	cmd.Flags().StringVar(&assignBeads, "beads", "", "Comma-separated list of specific bead IDs to assign")

	return cmd
}

func runAssign(cmd *cobra.Command, args []string) error {
	var session string
	if len(args) > 0 {
		session = args[0]
	}

	if err := tmux.EnsureInstalled(); err != nil {
		return err
	}

	// Resolve session
	res, err := ResolveSession(session, cmd.OutOrStdout())
	if err != nil {
		return err
	}
	if res.Session == "" {
		return nil
	}
	res.ExplainIfInferred(cmd.ErrOrStderr())
	session = res.Session

	// Check if bv is available
	if !bv.IsInstalled() {
		return fmt.Errorf("bv is not installed - required for work assignment")
	}

	// Parse beads if specified
	var beadIDs []string
	if assignBeads != "" {
		beadIDs = strings.Split(assignBeads, ",")
		for i := range beadIDs {
			beadIDs[i] = strings.TrimSpace(beadIDs[i])
		}
	}

	// Get assignment recommendations via robot module
	opts := robot.AssignOptions{
		Session:  session,
		Beads:    beadIDs,
		Strategy: assignStrategy,
	}

	// For JSON output, use the robot module directly
	if IsJSONOutput() {
		return robot.PrintAssign(opts)
	}

	// For text output, get the data and format it nicely
	output, err := getAssignOutput(opts)
	if err != nil {
		return err
	}

	// Display the recommendations
	displayAssignOutput(output)

	// If no recommendations, we're done
	if len(output.Recommendations) == 0 {
		return nil
	}

	// If auto mode, execute assignments
	if assignAuto {
		return executeAssignments(session, output.Recommendations)
	}

	// Otherwise, prompt for confirmation
	fmt.Println()
	fmt.Print("Execute all assignments? [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" {
		return executeAssignments(session, output.Recommendations)
	}

	fmt.Println("Assignments cancelled.")
	return nil
}

// getAssignOutput builds the assignment output without printing
func getAssignOutput(opts robot.AssignOptions) (*robot.AssignOutput, error) {
	if !tmux.SessionExists(opts.Session) {
		return nil, fmt.Errorf("session '%s' not found", opts.Session)
	}

	// Get panes from tmux
	panes, err := tmux.GetPanes(opts.Session)
	if err != nil {
		return nil, fmt.Errorf("failed to get panes: %w", err)
	}

	// Build agent info similar to robot.PrintAssign
	var idleAgentPanes []string
	totalAgents := 0

	for _, pane := range panes {
		agentType := detectAgentTypeFromTitle(pane.Title)
		if agentType == "user" || agentType == "unknown" {
			continue
		}
		totalAgents++

		// Capture state
		scrollback, _ := tmux.CapturePaneOutput(pane.ID, 10)
		state := determineAgentState(scrollback, agentType)
		if state == "idle" {
			idleAgentPanes = append(idleAgentPanes, fmt.Sprintf("%d", pane.Index))
		}
	}

	// Get beads from bv
	wd, _ := os.Getwd()
	readyBeads := bv.GetReadyPreview(wd, 50)
	inProgress := bv.GetInProgressList(wd, 50)

	// Filter to specific beads if requested
	if len(opts.Beads) > 0 {
		beadSet := make(map[string]bool)
		for _, b := range opts.Beads {
			beadSet[b] = true
		}
		var filtered []bv.BeadPreview
		for _, b := range readyBeads {
			if beadSet[b.ID] {
				filtered = append(filtered, b)
			}
		}
		readyBeads = filtered
	}

	// Generate recommendations
	recommendations := generateRecommendations(panes, readyBeads, opts.Strategy, idleAgentPanes)

	output := &robot.AssignOutput{
		Session:         opts.Session,
		Strategy:        opts.Strategy,
		Recommendations: recommendations,
		IdleAgents:      idleAgentPanes,
		Summary: robot.AssignSummary{
			TotalAgents:     totalAgents,
			IdleAgents:      len(idleAgentPanes),
			WorkingAgents:   totalAgents - len(idleAgentPanes),
			ReadyBeads:      len(readyBeads),
			Recommendations: len(recommendations),
		},
	}

	// Add warnings
	hints := &robot.AssignAgentHints{}
	if len(recommendations) == 0 && len(readyBeads) == 0 {
		hints.Summary = "No work available to assign"
	} else if len(recommendations) == 0 && len(idleAgentPanes) == 0 {
		hints.Summary = fmt.Sprintf("%d beads ready but no idle agents available", len(readyBeads))
	} else if len(recommendations) > 0 {
		hints.Summary = fmt.Sprintf("%d assignments recommended for %d idle agents", len(recommendations), len(idleAgentPanes))
	}

	if len(readyBeads) > len(idleAgentPanes) && len(idleAgentPanes) > 0 {
		diff := len(readyBeads) - len(idleAgentPanes)
		hints.Warnings = append(hints.Warnings,
			fmt.Sprintf("%d beads won't be assigned - not enough idle agents", diff))
	}

	for _, b := range inProgress {
		if b.UpdatedAt.IsZero() {
			continue
		}
		// Check for stale beads (simplified)
	}

	output.AgentHints = hints

	return output, nil
}

// generateRecommendations creates assignment recommendations
func generateRecommendations(panes []tmux.Pane, beads []bv.BeadPreview, strategy string, idleAgents []string) []robot.AssignRecommend {
	var recommendations []robot.AssignRecommend

	// Create a map of idle agents
	idleSet := make(map[string]bool)
	for _, a := range idleAgents {
		idleSet[a] = true
	}

	// Get idle pane details
	var idlePanes []tmux.Pane
	for _, p := range panes {
		paneKey := fmt.Sprintf("%d", p.Index)
		if idleSet[paneKey] {
			idlePanes = append(idlePanes, p)
		}
	}

	// Match beads to idle agents
	beadIdx := 0
	for _, pane := range idlePanes {
		if beadIdx >= len(beads) {
			break
		}

		bead := beads[beadIdx]
		agentType := detectAgentTypeFromTitle(pane.Title)
		model := detectModelFromTitle(agentType, pane.Title)

		confidence := calculateMatchConfidence(agentType, bead, strategy)
		reasoning := buildReasoning(agentType, bead, strategy)

		recommendations = append(recommendations, robot.AssignRecommend{
			Agent:      fmt.Sprintf("%d", pane.Index),
			AgentType:  agentType,
			Model:      model,
			AssignBead: bead.ID,
			BeadTitle:  bead.Title,
			Priority:   bead.Priority,
			Confidence: confidence,
			Reasoning:  reasoning,
		})

		beadIdx++
	}

	return recommendations
}

// detectAgentTypeFromTitle determines agent type from pane title
func detectAgentTypeFromTitle(title string) string {
	title = strings.ToLower(title)
	if strings.Contains(title, "__cc") || strings.Contains(title, "claude") {
		return "claude"
	}
	if strings.Contains(title, "__cod") || strings.Contains(title, "codex") {
		return "codex"
	}
	if strings.Contains(title, "__gmi") || strings.Contains(title, "gemini") {
		return "gemini"
	}
	if strings.Contains(title, "__user") || strings.Contains(title, "user") {
		return "user"
	}
	return "unknown"
}

// detectModelFromTitle extracts model variant from title
func detectModelFromTitle(agentType, title string) string {
	// Simplified model detection
	title = strings.ToLower(title)
	if strings.Contains(title, "opus") {
		return "opus"
	}
	if strings.Contains(title, "sonnet") {
		return "sonnet"
	}
	if strings.Contains(title, "haiku") {
		return "haiku"
	}
	return ""
}

// determineAgentState checks if agent is idle or working
func determineAgentState(scrollback, agentType string) string {
	lines := strings.Split(scrollback, "\n")
	if len(lines) == 0 {
		return "unknown"
	}

	lastLine := strings.TrimSpace(lines[len(lines)-1])

	// Look for common idle patterns
	idlePatterns := []string{
		"$", ">", ">>> ", "claude>", "codex>", "gemini>",
		"What would you like", "How can I help",
		"Ready for", "Waiting for",
	}

	for _, p := range idlePatterns {
		if strings.HasSuffix(lastLine, p) || strings.Contains(lastLine, p) {
			return "idle"
		}
	}

	return "working"
}

// calculateMatchConfidence calculates how well an agent matches a task
func calculateMatchConfidence(agentType string, bead bv.BeadPreview, strategy string) float64 {
	baseConfidence := 0.7

	// Task type inference
	title := strings.ToLower(bead.Title)
	taskType := "task"

	taskPatterns := map[string][]string{
		"bug":           {"bug", "fix", "broken", "error", "crash"},
		"testing":       {"test", "spec", "coverage"},
		"documentation": {"doc", "readme", "comment"},
		"refactor":      {"refactor", "cleanup", "improve"},
		"analysis":      {"analyze", "investigate", "research"},
		"feature":       {"feature", "implement", "add", "new"},
	}

	for tt, patterns := range taskPatterns {
		for _, p := range patterns {
			if strings.Contains(title, p) {
				taskType = tt
				break
			}
		}
	}

	// Agent strengths
	strengths := map[string]map[string]float64{
		"claude": {"analysis": 0.9, "refactor": 0.9, "documentation": 0.8, "feature": 0.8, "bug": 0.7},
		"codex":  {"feature": 0.9, "bug": 0.8, "task": 0.8, "refactor": 0.6},
		"gemini": {"documentation": 0.9, "analysis": 0.8, "feature": 0.8},
	}

	if agentStrengths, ok := strengths[agentType]; ok {
		if strength, ok := agentStrengths[taskType]; ok {
			baseConfidence = strength
		}
	}

	// Strategy adjustments
	switch strategy {
	case "speed":
		baseConfidence = (baseConfidence + 0.9) / 2
	case "dependency":
		priority := parsePriorityString(bead.Priority)
		if priority <= 1 {
			baseConfidence = min(baseConfidence+0.1, 0.95)
		}
	}

	return baseConfidence
}

// parsePriorityString converts "P0"-"P4" to integer
func parsePriorityString(p string) int {
	if len(p) == 2 && p[0] == 'P' {
		if n := p[1] - '0'; n <= 4 {
			return int(n)
		}
	}
	return 2
}

// buildReasoning creates explanation for assignment
func buildReasoning(agentType string, bead bv.BeadPreview, strategy string) string {
	var reasons []string

	title := strings.ToLower(bead.Title)
	priority := parsePriorityString(bead.Priority)

	// Task-agent match
	if agentType == "claude" && (strings.Contains(title, "refactor") || strings.Contains(title, "analyze")) {
		reasons = append(reasons, "Claude excels at analysis/refactoring")
	} else if agentType == "codex" && (strings.Contains(title, "feature") || strings.Contains(title, "implement")) {
		reasons = append(reasons, "Codex excels at implementations")
	} else if agentType == "gemini" && strings.Contains(title, "doc") {
		reasons = append(reasons, "Gemini excels at documentation")
	}

	// Priority
	if priority == 0 {
		reasons = append(reasons, "critical priority")
	} else if priority == 1 {
		reasons = append(reasons, "high priority")
	}

	// Strategy
	switch strategy {
	case "balanced":
		reasons = append(reasons, "balanced workload")
	case "speed":
		reasons = append(reasons, "optimizing for speed")
	case "quality":
		reasons = append(reasons, "optimizing for quality")
	case "dependency":
		reasons = append(reasons, "prioritizing unblocks")
	}

	if len(reasons) == 0 {
		return "available agent matched to available work"
	}

	return strings.Join(reasons, "; ")
}

// displayAssignOutput renders the assignment output as formatted text
func displayAssignOutput(output *robot.AssignOutput) {
	th := theme.Current()

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(th.Primary)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(th.Subtext)

	fmt.Println()
	fmt.Println(titleStyle.Render(fmt.Sprintf("Task Assignment Recommendations for %s", output.Session)))
	fmt.Println(strings.Repeat("━", 50))

	// Summary
	fmt.Println()
	fmt.Printf("Strategy: %s\n", output.Strategy)
	fmt.Printf("Agents: %d total, %d idle, %d working\n",
		output.Summary.TotalAgents,
		output.Summary.IdleAgents,
		output.Summary.WorkingAgents)
	fmt.Printf("Beads: %d ready\n", output.Summary.ReadyBeads)

	// Hints summary
	if output.AgentHints != nil && output.AgentHints.Summary != "" {
		fmt.Println()
		fmt.Println(subtitleStyle.Render(output.AgentHints.Summary))
	}

	// Recommendations
	if len(output.Recommendations) > 0 {
		fmt.Println()
		fmt.Println(titleStyle.Render("Recommended Assignments:"))
		fmt.Println()

		for _, rec := range output.Recommendations {
			agentStyle := getAgentStyle(rec.AgentType, th)
			priorityStyle := getPriorityStyle(rec.Priority, th)

			// Agent badge
			agentBadge := agentStyle.Render(fmt.Sprintf("[%s pane %s]", rec.AgentType, rec.Agent))
			if rec.Model != "" {
				agentBadge = agentStyle.Render(fmt.Sprintf("[%s/%s pane %s]", rec.AgentType, rec.Model, rec.Agent))
			}

			// Priority badge
			priorityBadge := priorityStyle.Render(fmt.Sprintf("[%s]", rec.Priority))

			// Confidence
			confStr := fmt.Sprintf("(%.0f%% confidence)", rec.Confidence*100)

			fmt.Printf("  %s → %s %s %s\n",
				agentBadge,
				rec.AssignBead,
				priorityBadge,
				confStr)
			fmt.Printf("     %s\n", rec.BeadTitle)
			if rec.Reasoning != "" {
				fmt.Printf("     %s\n", subtitleStyle.Render(rec.Reasoning))
			}
			fmt.Println()
		}
	} else {
		fmt.Println()
		fmt.Println(subtitleStyle.Render("No assignments to recommend."))
	}

	// Warnings
	if output.AgentHints != nil && len(output.AgentHints.Warnings) > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(th.Warning)
		fmt.Println(warnStyle.Render("Warnings:"))
		for _, w := range output.AgentHints.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}
}

// getAgentStyle returns a style for an agent type
func getAgentStyle(agentType string, th theme.Theme) lipgloss.Style {
	var color lipgloss.Color
	switch agentType {
	case "claude":
		color = th.Claude
	case "codex":
		color = th.Codex
	case "gemini":
		color = th.Gemini
	default:
		color = th.Text
	}
	return lipgloss.NewStyle().Foreground(color).Bold(true)
}

// getPriorityStyle returns a style for a priority level
func getPriorityStyle(priority string, th theme.Theme) lipgloss.Style {
	var color lipgloss.Color
	switch priority {
	case "P0":
		color = th.Error
	case "P1":
		color = th.Warning
	case "P2":
		color = th.Info
	default:
		color = th.Subtext
	}
	return lipgloss.NewStyle().Foreground(color)
}

// executeAssignments sends the assignments to agents
func executeAssignments(session string, recommendations []robot.AssignRecommend) error {
	fmt.Println()
	fmt.Println("Executing assignments...")

	for _, rec := range recommendations {
		// Build the prompt to send to the agent
		prompt := fmt.Sprintf("Please work on bead %s: %s", rec.AssignBead, rec.BeadTitle)

		// Send to the pane
		paneID := fmt.Sprintf("%s:%s", session, rec.Agent)
		if err := tmux.SendKeys(paneID, prompt, true); err != nil {
			fmt.Printf("  Failed to assign to pane %s: %v\n", rec.Agent, err)
			continue
		}

		fmt.Printf("  Assigned %s to pane %s (%s)\n", rec.AssignBead, rec.Agent, rec.AgentType)
	}

	fmt.Println()
	fmt.Println("Assignments sent. Use 'ntm status' to monitor progress.")
	return nil
}

// marshalAssignOutput converts output to JSON bytes (for testing)
func marshalAssignOutput(output *robot.AssignOutput) ([]byte, error) {
	return json.MarshalIndent(output, "", "  ")
}
