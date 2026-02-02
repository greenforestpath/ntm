package robot

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/tmux"
)

// RestartPaneOutput is the structured output for --robot-restart-pane
type RestartPaneOutput struct {
	RobotResponse
	Session      string         `json:"session"`
	RestartedAt  time.Time      `json:"restarted_at"`
	Restarted    []string       `json:"restarted"`
	Failed       []RestartError `json:"failed"`
	DryRun       bool           `json:"dry_run,omitempty"`
	WouldAffect  []string       `json:"would_affect,omitempty"`
	BeadAssigned string         `json:"bead_assigned,omitempty"` // Bead ID if --bead was used
	PromptSent   bool           `json:"prompt_sent,omitempty"`   // True if prompt was sent to pane(s)
	PromptError  string         `json:"prompt_error,omitempty"`  // Non-fatal prompt send error
}

// RestartError represents a failed restart attempt
type RestartError struct {
	Pane   string `json:"pane"`
	Reason string `json:"reason"`
}

// RestartPaneOptions configures the PrintRestartPane operation
type RestartPaneOptions struct {
	Session string   // Target session name
	Panes   []string // Specific pane indices to restart (empty = all agents)
	Type    string   // Filter by agent type (e.g., "claude", "cc")
	All     bool     // Include all panes (including user)
	DryRun  bool     // Preview mode
	Bead    string   // Bead ID to assign after restart (fetches info via br show --json)
	Prompt  string   // Custom prompt to send after restart (overrides --bead template)
}

// GetRestartPane restarts panes (respawn-pane -k) and returns the result.
// This function returns the data struct directly, enabling CLI/REST parity.
func GetRestartPane(opts RestartPaneOptions) (*RestartPaneOutput, error) {
	output := &RestartPaneOutput{
		RobotResponse: NewRobotResponse(true),
		Session:       opts.Session,
		RestartedAt:   time.Now().UTC(),
		Restarted:     []string{},
		Failed:        []RestartError{},
	}

	// If --bead is provided, validate it before restarting anything
	var beadPrompt string
	if opts.Bead != "" {
		prompt, err := buildBeadPrompt(opts.Bead)
		if err != nil {
			output.RobotResponse = NewErrorResponse(
				err,
				ErrCodeInvalidFlag,
				fmt.Sprintf("Bead %s not found or not readable. Use: br show %s", opts.Bead, opts.Bead),
			)
			return output, nil
		}
		beadPrompt = prompt
	}

	// Determine which prompt to send (explicit --prompt overrides --bead template)
	promptToSend := opts.Prompt
	if promptToSend == "" && beadPrompt != "" {
		promptToSend = beadPrompt
	}

	if !tmux.SessionExists(opts.Session) {
		output.Failed = append(output.Failed, RestartError{
			Pane:   "session",
			Reason: fmt.Sprintf("session '%s' not found", opts.Session),
		})
		output.RobotResponse = NewErrorResponse(
			fmt.Errorf("session '%s' not found", opts.Session),
			ErrCodeSessionNotFound,
			"Use --robot-status to list available sessions",
		)
		return output, nil
	}

	panes, err := tmux.GetPanes(opts.Session)
	if err != nil {
		output.Failed = append(output.Failed, RestartError{
			Pane:   "panes",
			Reason: fmt.Sprintf("failed to get panes: %v", err),
		})
		output.RobotResponse = NewErrorResponse(
			err,
			ErrCodeInternalError,
			"Check tmux session state",
		)
		return output, nil
	}

	// Build pane filter map
	paneFilterMap := make(map[string]bool)
	for _, p := range opts.Panes {
		paneFilterMap[p] = true
	}
	hasPaneFilter := len(paneFilterMap) > 0

	// Determine which panes to restart
	var targetPanes []tmux.Pane
	for _, pane := range panes {
		paneKey := fmt.Sprintf("%d", pane.Index)

		// Check specific pane filter
		if hasPaneFilter && !paneFilterMap[paneKey] && !paneFilterMap[pane.ID] {
			continue
		}

		// Filter by type if specified
		if opts.Type != "" {
			agentType := detectAgentType(pane.Title)
			// Normalize type for comparison (handle aliases like cc vs claude)
			targetType := translateAgentTypeForStatus(opts.Type)
			currentType := translateAgentTypeForStatus(agentType)
			if targetType != currentType {
				continue
			}
		}

		// Skip user panes by default unless --all or specific pane filter
		if !opts.All && !hasPaneFilter && opts.Type == "" {
			agentType := detectAgentType(pane.Title)
			if pane.Index == 0 && agentType == "unknown" {
				continue
			}
			if agentType == "user" {
				continue
			}
		}

		targetPanes = append(targetPanes, pane)
	}

	if len(targetPanes) == 0 {
		return output, nil
	}

	// Dry-run mode
	if opts.DryRun {
		output.DryRun = true
		for _, pane := range targetPanes {
			paneKey := fmt.Sprintf("%d", pane.Index)
			output.WouldAffect = append(output.WouldAffect, paneKey)
		}
		if opts.Bead != "" {
			output.BeadAssigned = opts.Bead
		}
		return output, nil
	}

	// Restart targets
	for _, pane := range targetPanes {
		paneKey := fmt.Sprintf("%d", pane.Index)

		// Always use kill=true for restart to ensure process is cycled
		err := tmux.RespawnPane(pane.ID, true)
		if err != nil {
			output.Failed = append(output.Failed, RestartError{
				Pane:   paneKey,
				Reason: fmt.Sprintf("failed to respawn: %v", err),
			})
		} else {
			output.Restarted = append(output.Restarted, paneKey)
		}
	}

	// Send prompt to successfully restarted panes
	if promptToSend != "" && len(output.Restarted) > 0 {
		if opts.Bead != "" {
			output.BeadAssigned = opts.Bead
		}

		// Wait for panes to initialize after respawn
		time.Sleep(500 * time.Millisecond)

		var promptErrors []string
		for _, paneKey := range output.Restarted {
			target := fmt.Sprintf("%s:%s", opts.Session, paneKey)
			if err := tmux.SendKeys(target, promptToSend+"\n", false); err != nil {
				promptErrors = append(promptErrors, fmt.Sprintf("pane %s: %v", paneKey, err))
			}
		}

		if len(promptErrors) > 0 {
			output.PromptSent = false
			output.PromptError = strings.Join(promptErrors, "; ")
		} else {
			output.PromptSent = true
		}
	}

	return output, nil
}

// restartPaneBeadPromptTemplate is the default prompt template for --bead assignment.
const restartPaneBeadPromptTemplate = "Read AGENTS.md, register with Agent Mail. Work on: {bead_id} - {bead_title}.\nUse br show {bead_id} for details. Mark in_progress when starting. Use ultrathink."

// buildBeadPrompt fetches bead info via br show --json and builds the assignment prompt.
func buildBeadPrompt(beadID string) (string, error) {
	cmd := exec.Command("br", "show", beadID, "--json")
	cmd.Dir, _ = os.Getwd()
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("br show %s failed: %w", beadID, err)
	}

	var issues []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(out, &issues); err != nil {
		return "", fmt.Errorf("parse br show output: %w", err)
	}
	if len(issues) == 0 || issues[0].Title == "" {
		return "", fmt.Errorf("bead %s not found", beadID)
	}

	prompt := strings.NewReplacer(
		"{bead_id}", beadID,
		"{bead_title}", issues[0].Title,
	).Replace(restartPaneBeadPromptTemplate)

	return prompt, nil
}

// PrintRestartPane handles the --robot-restart-pane command.
// This is a thin wrapper around GetRestartPane() for CLI output.
func PrintRestartPane(opts RestartPaneOptions) error {
	output, err := GetRestartPane(opts)
	if err != nil {
		return err
	}
	return encodeJSON(output)
}
