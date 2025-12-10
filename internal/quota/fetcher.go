package quota

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/tmux"
)

// PTYFetcher implements Fetcher by sending commands to tmux panes
type PTYFetcher struct {
	// CommandTimeout is how long to wait for command execution
	CommandTimeout time.Duration
	// CaptureLines is how many lines to capture from pane output
	CaptureLines int
}

// providerCommands maps providers to their quota commands
var providerCommands = map[Provider]struct {
	UsageCmd  string
	StatusCmd string
}{
	ProviderClaude: {
		UsageCmd:  "/usage",
		StatusCmd: "/status",
	},
	ProviderCodex: {
		UsageCmd:  "/usage", // May need adjustment after research
		StatusCmd: "/status",
	},
	ProviderGemini: {
		UsageCmd:  "/auth status", // Gemini uses different commands
		StatusCmd: "/auth status",
	},
}

// FetchQuota sends quota commands to a pane and parses the output
func (f *PTYFetcher) FetchQuota(ctx context.Context, paneID string, provider Provider) (*QuotaInfo, error) {
	timeout := f.CommandTimeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	captureLines := f.CaptureLines
	if captureLines == 0 {
		captureLines = 100
	}

	cmds, ok := providerCommands[provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	info := &QuotaInfo{
		Provider:  provider,
		FetchedAt: time.Now(),
	}

	// Capture initial state for comparison
	initialOutput, err := tmux.CapturePaneOutput(paneID, captureLines)
	if err != nil {
		info.Error = fmt.Sprintf("failed to capture initial output: %v", err)
		return info, nil
	}

	// Send /usage command
	if err := tmux.SendKeys(paneID, cmds.UsageCmd, true); err != nil {
		info.Error = fmt.Sprintf("failed to send usage command: %v", err)
		return info, nil
	}

	// Wait for output with context timeout
	usageOutput, err := f.waitForNewOutput(ctx, paneID, initialOutput, captureLines, timeout)
	if err != nil {
		info.Error = fmt.Sprintf("failed to capture usage output: %v", err)
		return info, nil
	}

	// Parse usage output based on provider
	if err := parseUsageOutput(info, usageOutput, provider); err != nil {
		info.Error = fmt.Sprintf("failed to parse usage: %v", err)
	}
	info.RawOutput = usageOutput

	// Optionally fetch status for additional info
	statusOutput, err := f.fetchStatus(ctx, paneID, cmds.StatusCmd, captureLines, timeout)
	if err == nil && statusOutput != "" {
		parseStatusOutput(info, statusOutput, provider)
		info.RawOutput += "\n---\n" + statusOutput
	}

	return info, nil
}

// waitForNewOutput polls until new output appears after the initial capture
func (f *PTYFetcher) waitForNewOutput(ctx context.Context, paneID, initialOutput string, lines int, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return "", fmt.Errorf("timeout waiting for output")
			}

			output, err := tmux.CapturePaneOutput(paneID, lines)
			if err != nil {
				continue
			}

			// Check if output has changed
			if output != initialOutput && len(output) > len(initialOutput) {
				// Return the new portion
				newPart := strings.TrimPrefix(output, initialOutput)
				if newPart != "" {
					return strings.TrimSpace(newPart), nil
				}
			}
		}
	}
}

// fetchStatus sends a status command and captures output
func (f *PTYFetcher) fetchStatus(ctx context.Context, paneID, statusCmd string, lines int, timeout time.Duration) (string, error) {
	// Don't send if status command is same as usage command
	cmds := providerCommands[ProviderClaude]
	if statusCmd == cmds.UsageCmd {
		return "", nil
	}

	initialOutput, err := tmux.CapturePaneOutput(paneID, lines)
	if err != nil {
		return "", err
	}

	if err := tmux.SendKeys(paneID, statusCmd, true); err != nil {
		return "", err
	}

	return f.waitForNewOutput(ctx, paneID, initialOutput, lines, timeout)
}

// parseUsageOutput routes to provider-specific parsers
func parseUsageOutput(info *QuotaInfo, output string, provider Provider) error {
	switch provider {
	case ProviderClaude:
		return parseClaudeUsage(info, output)
	case ProviderCodex:
		return parseCodexUsage(info, output)
	case ProviderGemini:
		return parseGeminiUsage(info, output)
	default:
		return fmt.Errorf("no parser for provider: %s", provider)
	}
}

// parseStatusOutput routes to provider-specific status parsers
func parseStatusOutput(info *QuotaInfo, output string, provider Provider) {
	switch provider {
	case ProviderClaude:
		parseClaudeStatus(info, output)
	case ProviderCodex:
		parseCodexStatus(info, output)
	case ProviderGemini:
		parseGeminiStatus(info, output)
	}
}
