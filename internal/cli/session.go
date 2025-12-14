package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Dicklesworthstone/ntm/internal/agentmail"
	"github.com/Dicklesworthstone/ntm/internal/output"
	"github.com/Dicklesworthstone/ntm/internal/status"
	"github.com/Dicklesworthstone/ntm/internal/tmux"
	"github.com/Dicklesworthstone/ntm/internal/tui/icons"
	"github.com/Dicklesworthstone/ntm/internal/tui/layout"
	"github.com/Dicklesworthstone/ntm/internal/tui/theme"
)

func newAttachCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "attach <session-name>",
		Aliases: []string{"a"},
		Short:   "Attach to a tmux session",
		Long: `Attach to an existing tmux session. If already inside tmux,
switches to the target session instead.

If the session doesn't exist, shows available sessions.

Examples:
  ntm attach myproject`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				// No session specified, list sessions
				return runList(nil)
			}
			return runAttach(args[0])
		},
	}
}

func runAttach(session string) error {
	if err := tmux.EnsureInstalled(); err != nil {
		return err
	}

	if tmux.SessionExists(session) {
		// Update Agent Mail activity (non-blocking)
		updateSessionActivity(session)
		return tmux.AttachOrSwitch(session)
	}

	if IsJSONOutput() {
		return output.PrintJSON(output.NewError(fmt.Sprintf("session '%s' does not exist", session)))
	}

	fmt.Printf("Session '%s' does not exist.\n\n", session)
	fmt.Println("Available sessions:")
	if err := runList(nil); err != nil {
		return err
	}
	fmt.Println()

	if confirm(fmt.Sprintf("Create '%s' with default settings?", session)) {
		return runCreate(session, 0)
	}

	return nil
}

func newListCmd() *cobra.Command {
	var tags []string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List all tmux sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(tags)
		},
	}
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "filter sessions by agent tag (shows session if any agent matches)")
	return cmd
}

func runList(tags []string) error {
	if err := tmux.EnsureInstalled(); err != nil {
		if IsJSONOutput() {
			_ = output.PrintJSON(output.NewError(err.Error()))
			return err
		}
		return err
	}

	sessions, err := tmux.ListSessions()
	if err != nil {
		if IsJSONOutput() {
			_ = output.PrintJSON(output.NewError(err.Error()))
			return err
		}
		return err
	}

	// Filter sessions by tag
	if len(tags) > 0 {
		var filtered []tmux.Session
		for _, s := range sessions {
			panes, err := tmux.GetPanes(s.Name)
			if err == nil {
				// Check if any pane has matching tag
				hasTag := false
				for _, p := range panes {
					if HasAnyTag(p.Tags, tags) {
						hasTag = true
						break
					}
				}
				if hasTag {
					filtered = append(filtered, s)
				}
			}
		}
		sessions = filtered
	}

	// Build response for JSON output
	if IsJSONOutput() {
		items := make([]output.SessionListItem, len(sessions))
		for i, s := range sessions {
			item := output.SessionListItem{
				Name:             s.Name,
				Windows:          s.Windows,
				Attached:         s.Attached,
				WorkingDirectory: s.Directory,
			}

			// Get panes to count agents
			panes, err := tmux.GetPanes(s.Name)
			if err == nil {
				item.PaneCount = len(panes)

				// Count agent types
				var claudeCount, codexCount, geminiCount, userCount int
				for _, p := range panes {
					switch p.Type {
					case tmux.AgentClaude:
						claudeCount++
					case tmux.AgentCodex:
						codexCount++
					case tmux.AgentGemini:
						geminiCount++
					default:
						userCount++
					}
				}
				item.AgentCounts = &output.AgentCountsResponse{
					Claude: claudeCount,
					Codex:  codexCount,
					Gemini: geminiCount,
					User:   userCount,
					Total:  len(panes),
				}
			}
			items[i] = item
		}
		resp := output.ListResponse{
			TimestampedResponse: output.NewTimestamped(),
			Sessions:            items,
			Count:               len(sessions),
		}
		return output.PrintJSON(resp)
	}

	// Text output
	if len(sessions) == 0 {
		fmt.Println("No tmux sessions running")
		return nil
	}

	// Check terminal width for responsive output
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	isWide := width >= 100

	if isWide {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "SESSION\tWINDOWS\tSTATE\tAGENTS")

		for _, s := range sessions {
			attached := "detached"
			if s.Attached {
				attached = "attached"
			}

			// Fetch agents summary
			agents := "-"
			panes, err := tmux.GetPanes(s.Name)
			if err == nil {
				var cc, cod, gmi, user int
				for _, p := range panes {
					switch p.Type {
					case tmux.AgentClaude:
						cc++
					case tmux.AgentCodex:
						cod++
					case tmux.AgentGemini:
						gmi++
					default:
						user++
					}
				}
				var parts []string
				if cc > 0 {
					parts = append(parts, fmt.Sprintf("%d CC", cc))
				}
				if cod > 0 {
					parts = append(parts, fmt.Sprintf("%d COD", cod))
				}
				if gmi > 0 {
					parts = append(parts, fmt.Sprintf("%d GMI", gmi))
				}
				if user > 0 {
					parts = append(parts, fmt.Sprintf("%d Usr", user))
				}
				if len(parts) > 0 {
					agents = strings.Join(parts, ", ")
				}
			}

			fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", s.Name, s.Windows, attached, agents)
		}
		w.Flush()
	} else {
		// Standard output for narrow screens
		for _, s := range sessions {
			attached := ""
			if s.Attached {
				attached = " (attached)"
			}
			fmt.Printf("  %s: %d windows%s\n", s.Name, s.Windows, attached)
		}
	}

	return nil
}
func newStatusCmd() *cobra.Command {
	var tags []string
	cmd := &cobra.Command{
		Use:   "status <session-name>",
		Short: "Show detailed status of a session",
		Long: `Show detailed information about a session including:
- All panes with their titles and current commands
- Agent type counts (Claude, Codex, Gemini)
- Session directory

Examples:
  ntm status myproject
  ntm status myproject --tag=frontend`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd.OutOrStdout(), args[0], tags)
		},
	}
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "filter panes by tag")
	return cmd
}

func runStatus(w io.Writer, session string, tags []string) error {
	// Helper for JSON error output
	outputError := func(err error) error {
		if IsJSONOutput() {
			_ = output.PrintJSON(output.NewError(err.Error()))
			return err
		}
		return err
	}

	if err := tmux.EnsureInstalled(); err != nil {
		return outputError(err)
	}

	if !tmux.SessionExists(session) {
		if IsJSONOutput() {
			return output.PrintJSON(output.StatusResponse{
				TimestampedResponse: output.NewTimestamped(),
				Session:             session,
				Exists:              false,
			})
		}
		return fmt.Errorf("session '%s' not found", session)
	}

	panes, err := tmux.GetPanes(session)
	if err != nil {
		return outputError(err)
	}

	// Filter panes by tag
	if len(tags) > 0 {
		var filtered []tmux.Pane
		for _, p := range panes {
			if HasAnyTag(p.Tags, tags) {
				filtered = append(filtered, p)
			}
		}
		panes = filtered
	}

	dir := cfg.GetProjectDir(session)

	// Calculate counts
	var ccCount, codCount, gmiCount, otherCount int
	for _, p := range panes {
		switch p.Type {
		case tmux.AgentClaude:
			ccCount++
		case tmux.AgentCodex:
			codCount++
		case tmux.AgentGemini:
			gmiCount++
		default:
			otherCount++
		}
	}

	// JSON output mode - build structured response
	if IsJSONOutput() {
		// Check if session is attached
		attached := false
		sessions, _ := tmux.ListSessions()
		for _, s := range sessions {
			if s.Name == session {
				attached = s.Attached
				break
			}
		}

		resp := output.StatusResponse{
			TimestampedResponse: output.NewTimestamped(),
			Session:             session,
			Exists:              true,
			Attached:            attached,
			WorkingDirectory:    dir,
			AgentCounts: output.AgentCountsResponse{
				Claude: ccCount,
				Codex:  codCount,
				Gemini: gmiCount,
				User:   otherCount,
				Total:  len(panes),
			},
		}

		// Add panes
		for _, p := range panes {
			resp.Panes = append(resp.Panes, output.PaneResponse{
				Index:   p.Index,
				Title:   p.Title,
				Type:    agentTypeToString(p.Type),
				Variant: p.Variant,
				Active:  p.Active,
				Width:   p.Width,
				Height:  p.Height,
				Command: p.Command,
			})
		}

		return output.PrintJSON(resp)
	}

	// Text output
	t := theme.Current()

	// ANSI helpers
	const reset = "\033[0m"
	const bold = "\033[1m"

	// Colors
	primary := colorize(t.Primary)
	surface := colorize(t.Surface0)
	text := colorize(t.Text)
	subtext := colorize(t.Subtext)
	overlay := colorize(t.Overlay)
	success := colorize(t.Success)
	claude := colorize(t.Claude)
	codex := colorize(t.Codex)
	gemini := colorize(t.Gemini)

	ic := icons.Current()

	// Detect terminal width and layout tier
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80 // Default fallback
	}
	tier := layout.TierForWidth(width)

	fmt.Fprintln(w)

	// Header with icon
	fmt.Fprintf(w, "  %s%s%s %s%s%s%s\n", primary, ic.Session, reset, bold, session, reset, text)
	fmt.Fprintf(w, "  %s%s%s\n", surface, "─────────────────────────────────────────────────────────", reset)
	fmt.Fprintln(w)

	// Directory info
	fmt.Fprintf(w, "  %s%s Directory:%s %s%s%s\n", subtext, ic.Folder, reset, text, dir, reset)
	fmt.Fprintf(w, "  %s%s Panes:%s    %s%d%s\n", subtext, ic.Pane, reset, text, len(panes), reset)
	fmt.Fprintln(w)

	// Panes section
	fmt.Fprintf(w, "  %sPanes%s\n", bold, reset)
	fmt.Fprintf(w, "  %s%s%s\n", surface, "─────────────────────────────────────────────────────────", reset)

	// Create status detector for agent state detection
	detector := status.NewDetector()

	// Get error color for status display
	errorColor := colorize(t.Error)

	for i, p := range panes {
		var typeColor, typeIcon string
		switch p.Type {
		case tmux.AgentClaude:
			typeColor = claude
			typeIcon = ic.Claude
		case tmux.AgentCodex:
			typeColor = codex
			typeIcon = ic.Codex
		case tmux.AgentGemini:
			typeColor = gemini
			typeIcon = ic.Gemini
		default:
			typeColor = success
			typeIcon = ic.User
		}

		// Number for quick selection (1-9)
		num := ""
		if i < 9 {
			num = fmt.Sprintf("%s%d%s ", overlay, i+1, reset)
		} else {
			num = "  "
		}

		// Detect agent status
		agentStatus, _ := detector.Detect(p.ID)
		stateIcon := agentStatus.State.Icon()
		stateColor := overlay
		stateText := ""
		switch agentStatus.State {
		case status.StateIdle:
			stateColor = overlay
			stateText = "idle"
		case status.StateWorking:
			stateColor = success
			stateText = "working"
		case status.StateError:
			stateColor = errorColor
			stateText = "error"
			if agentStatus.ErrorType != status.ErrorNone {
				stateText = string(agentStatus.ErrorType)
			}
		default:
			stateColor = overlay
			stateText = "unknown"
		}

		// Calculate columns based on tier
		var variantPart, cmdPart string
		var titleWidth int
		var variantWidth int
		var cmdWidth int

		switch {
		case tier >= layout.TierUltra:
			titleWidth = 35
			variantWidth = 15
			cmdWidth = 40
		case tier >= layout.TierWide:
			titleWidth = 25
			variantWidth = 10
			cmdWidth = 25
		case tier >= layout.TierSplit:
			titleWidth = 20
			variantWidth = 0
			cmdWidth = 15
		default: // Narrow
			titleWidth = 15
			variantWidth = 0
			cmdWidth = 10
		}

		if actual := utf8.RuneCountInString(p.Title); actual > titleWidth {
			titleWidth = actual
		}

		title := layout.TruncateRunes(p.Title, titleWidth, "…")
		titlePart := fmt.Sprintf("%*s", titleWidth, title)

		if variantWidth > 0 {
			variant := ""
			if p.Variant != "" {
				variant = layout.TruncateRunes(p.Variant, variantWidth, "…")
			}
			variantPart = fmt.Sprintf(" %s%-*s%s", subtext, variantWidth, variant, reset)
		}

		if cmdWidth > 0 {
			cmd := ""
			if p.Command != "" {
				cmd = layout.TruncateRunes(p.Command, cmdWidth, "…")
			}
			cmdPart = fmt.Sprintf(" %s%-*s%s", subtext, cmdWidth, cmd, reset)
		}

		// Pane info with status
		fmt.Fprintf(w, "  %s%s %s%s %s%s%s%s %s│%s %s%-8s%s\n",
			num,
			stateIcon,
			typeColor, typeIcon,
			titlePart,
			reset,
			variantPart,
			cmdPart,
			surface, reset,
			stateColor, stateText, reset)
	}

	fmt.Fprintf(w, "  %s%s%s\n", surface, "─────────────────────────────────────────────────────────", reset)
	fmt.Fprintln(w)

	// Agent summary with icons
	fmt.Fprintf(w, "  %sAgents%s\n", bold, reset)

	if ccCount > 0 {
		fmt.Fprintf(w, "    %s%s Claude%s  %s%d instance(s)%s\n", claude, ic.Claude, reset, text, ccCount, reset)
	}
	if codCount > 0 {
		fmt.Fprintf(w, "    %s%s Codex%s   %s%d instance(s)%s\n", codex, ic.Codex, reset, text, codCount, reset)
	}
	if gmiCount > 0 {
		fmt.Fprintf(w, "    %s%s Gemini%s  %s%d instance(s)%s\n", gemini, ic.Gemini, reset, text, gmiCount, reset)
	}
	if otherCount > 0 {
		fmt.Fprintf(w, "    %s%s User%s    %s%d pane(s)%s\n", success, ic.User, reset, text, otherCount, reset)
	}

	totalAgents := ccCount + codCount + gmiCount
	if totalAgents == 0 {
		fmt.Fprintf(w, "    %sNo agents running%s\n", overlay, reset)
	}

	fmt.Fprintln(w)

	// Agent Mail section
	agentMailStatus := fetchAgentMailStatus(dir)
	if agentMailStatus != nil && agentMailStatus.Available {
		mailColor := colorize(t.Lavender)
		lockIcon := "🔒"

		fmt.Fprintf(w, "  %sAgent Mail%s\n", bold, reset)
		fmt.Fprintf(w, "  %s%s%s\n", surface, "─────────────────────────────────────────────────────────", reset)

		if agentMailStatus.Connected {
			fmt.Fprintf(w, "    %s✓ Connected%s to %s%s%s\n", success, reset, subtext, agentMailStatus.ServerURL, reset)
		} else {
			fmt.Fprintf(w, "    %s○ Available%s at %s%s%s\n", overlay, reset, subtext, agentMailStatus.ServerURL, reset)
		}

		if agentMailStatus.ActiveLocks > 0 {
			fmt.Fprintf(w, "    %s%s Active Locks:%s %s%d reservation(s)%s\n",
				mailColor, lockIcon, reset, text, agentMailStatus.ActiveLocks, reset)
			for _, r := range agentMailStatus.Reservations {
				lockType := "shared"
				if r.Exclusive {
					lockType = "exclusive"
				}
				fmt.Fprintf(w, "      %s• %s%s  %s%s%s (%s, %s)\n",
					subtext, text, r.PathPattern, overlay, r.AgentName, reset, lockType, r.ExpiresIn)
			}
		} else {
			fmt.Fprintf(w, "    %s%s No active file locks%s\n", overlay, lockIcon, reset)
		}

		fmt.Fprintln(w)
	}

	// Quick actions hint
	fmt.Fprintf(w, "  %sQuick actions:%s\n", overlay, reset)
	fmt.Fprintf(w, "    %sntm send %s --all \"prompt\"%s  %s# Broadcast to all agents%s\n",
		subtext, session, reset, overlay, reset)
	fmt.Fprintf(w, "    %sntm view %s%s                 %s# Tile all panes%s\n",
		subtext, session, reset, overlay, reset)
	fmt.Fprintf(w, "    %sntm zoom %s <n>%s             %s# Zoom pane n%s\n",
		subtext, session, reset, overlay, reset)
	fmt.Fprintln(w)

	return nil
}

// updateSessionActivity updates the Agent Mail activity for a session.
// This is non-blocking and silently ignores errors.
func updateSessionActivity(sessionName string) {
	client := agentmail.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	projectKey := ""
	if cfg != nil {
		projectKey = cfg.GetProjectDir(sessionName)
	}

	_ = client.UpdateSessionActivity(ctx, sessionName, projectKey)
}

// fetchAgentMailStatus retrieves Agent Mail status for display in ntm status.
// Returns nil if Agent Mail is unavailable (graceful degradation).
func fetchAgentMailStatus(projectKey string) *output.AgentMailStatus {
	client := agentmail.NewClient(agentmail.WithProjectKey(projectKey))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Build status response
	status := &output.AgentMailStatus{
		Available: false,
		Connected: false,
		ServerURL: client.BaseURL(),
	}

	// Check if server is available
	if !client.IsAvailable() {
		return status
	}
	status.Available = true

	// Ensure project exists
	_, err := client.EnsureProject(ctx, projectKey)
	if err != nil {
		return status
	}
	status.Connected = true

	// Fetch file reservations (locks)
	reservations, err := client.ListReservations(ctx, projectKey, "", true)
	if err == nil {
		status.ActiveLocks = len(reservations)
		for _, r := range reservations {
			expiresIn := ""
			if !r.ExpiresTS.IsZero() {
				remaining := time.Until(r.ExpiresTS)
				if remaining > 0 {
					expiresIn = formatDuration(remaining)
				} else {
					expiresIn = "expired"
				}
			}
			status.Reservations = append(status.Reservations, output.FileReservationInfo{
				PathPattern: r.PathPattern,
				AgentName:   r.AgentName,
				Exclusive:   r.Exclusive,
				Reason:      r.Reason,
				ExpiresIn:   expiresIn,
			})
		}
	}

	// Note: Fetching inbox requires knowing agent names, which we don't have
	// in the general status view. This would need to iterate over all project
	// agents - deferred to ntm-161 (inbox command).

	return status
}

// formatDuration formats a duration in human-readable form
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
