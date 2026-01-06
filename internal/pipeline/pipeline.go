package pipeline

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/status"
	"github.com/Dicklesworthstone/ntm/internal/tmux"
)

// Stage represents a step in the pipeline
type Stage struct {
	AgentType string
	Prompt    string
	Model     string // Optional
}

// Pipeline represents a sequence of stages
type Pipeline struct {
	Session string
	Stages  []Stage
}

// Execute runs the pipeline stages sequentially
func Execute(ctx context.Context, p Pipeline) error {
	var previousOutput string
	var lastPaneID string

	detector := status.NewDetector()

	for i, stage := range p.Stages {
		log.Printf("Stage %d/%d [%s]: %s", i+1, len(p.Stages), stage.AgentType, truncate(stage.Prompt, 50))

		// 1. Find a suitable pane
		paneID, err := findPaneForStage(p.Session, stage.AgentType, stage.Model)
		if err != nil {
			return fmt.Errorf("stage %d failed: %w", i+1, err)
		}

		// Capture state BEFORE sending prompt to isolate the new response later
		beforeOutput, err := tmux.CapturePaneOutput(paneID, 2000)
		if err != nil {
			// Non-fatal, just means we might capture too much context
			beforeOutput = ""
		}

		// 2. Prepare prompt
		prompt := stage.Prompt
		if previousOutput != "" {
			if paneID == lastPaneID {
				// Same agent, context is already in the pane history.
				// Add a reference to the previous output instead of duplicating it.
				prompt = fmt.Sprintf("%s\n\n(See previous output above)", prompt)
			} else {
				// Different agent, inject the output from the previous stage
				prompt = fmt.Sprintf("%s\n\nResult from previous stage:\n%s", prompt, previousOutput)
			}
		}

		// 3. Send prompt
		if err := tmux.PasteKeys(paneID, prompt, true); err != nil {
			return fmt.Errorf("stage %d sending prompt: %w", i+1, err)
		}

		// 4. Wait for working state (debounce)
		time.Sleep(2 * time.Second)

		// 5. Wait for idle state
		log.Printf("  Waiting for agent...")
		if err := waitForIdle(ctx, detector, paneID); err != nil {
			return fmt.Errorf("stage %d waiting for completion: %w", i+1, err)
		}
		log.Printf("  Done.")

		// 6. Capture output
		// We capture a larger buffer to ensure we get the full response.
		// 2000 lines should cover most responses without being excessive.
		afterOutput, err := tmux.CapturePaneOutput(paneID, 2000)
		if err != nil {
			return fmt.Errorf("stage %d capturing output: %w", i+1, err)
		}

		// Extract only the new content
		output := extractNewOutput(beforeOutput, afterOutput)
		previousOutput = output
		lastPaneID = paneID
	}

	return nil
}

// extractNewOutput isolates the text added to the pane after the before state.
func extractNewOutput(before, after string) string {
	if before == "" {
		return after
	}
	if after == "" {
		return ""
	}

	// Fast path: if 'after' starts with 'before', it's a simple append
	if len(after) >= len(before) && after[:len(before)] == before {
		return after[len(before):]
	}

	// Handle scrolled output (before is not a prefix of after)
	// We look for the longest suffix of 'before' that matches a prefix of 'after'.

	// Try to match using a chunk from the start of 'after'
	const chunkSize = 40
	var searchChunk string
	if len(after) >= chunkSize {
		searchChunk = after[:chunkSize]
	} else {
		// After is short, use all of it
		searchChunk = after
	}

	// We only need to search the end of 'before' that could possibly contain 'after'
	// The overlap cannot be longer than 'after'.
	scanStart := len(before) - len(after)
	if scanStart < 0 {
		scanStart = 0
	}

	searchRegion := before[scanStart:]

	// Find all occurrences of chunk in searchRegion
	// We want the *first* valid match in searchRegion because that gives the *longest* suffix of before.
	// (Earliest start index = longest suffix)

	remaining := searchRegion
	offset := 0

	for {
		idx := strings.Index(remaining, searchChunk)
		if idx == -1 {
			break
		}

		// Absolute index in 'before'
		absIdx := scanStart + offset + idx

		// Check if this starts a valid suffix match
		// Suffix of before: before[absIdx:]
		// Prefix of after: after[:len(suffix)]
		// We know before[absIdx:] starts with searchChunk.
		// We need to check if the rest matches.

		suffixLen := len(before) - absIdx
		if len(after) >= suffixLen && after[:suffixLen] == before[absIdx:] {
			return after[suffixLen:]
		}

		// Move past this match
		step := idx + 1
		remaining = remaining[step:]
		offset += step
	}

	// Fallback for overlaps smaller than chunkSize (only needed if we capped chunk size)
	if len(after) > chunkSize {
		for k := chunkSize - 1; k > 0; k-- {
			if before[len(before)-k:] == after[:k] {
				return after[k:]
			}
		}
	}

	// No overlap found - return everything
	return after
}

func findPaneForStage(session, agentType, model string) (string, error) {
	targetType := normalizeAgentType(agentType)
	panes, err := tmux.GetPanes(session)
	if err != nil {
		return "", err
	}

	// First pass: look for exact match (type + model)
	for _, p := range panes {
		if string(p.Type) == targetType {
			// Check model if specified
			if model != "" && p.Variant != model {
				continue
			}
			return p.ID, nil
		}
	}

	// Second pass: relaxed match (type only, ignore model mismatch if not found)
	// Only if model was specified but not found
	if model != "" {
		for _, p := range panes {
			if string(p.Type) == targetType {
				return p.ID, nil
			}
		}
	}

	return "", fmt.Errorf("no agent found for type %s (model %s)", agentType, model)
}

func normalizeAgentType(t string) string {
	switch strings.ToLower(t) {
	case "claude", "cc", "claude-code":
		return "cc"
	case "codex", "cod", "openai":
		return "cod"
	case "gemini", "gmi", "google":
		return "gmi"
	default:
		return t
	}
}

func waitForIdle(ctx context.Context, detector status.Detector, paneID string) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute) // Max 30 min per stage default

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for agent")
		case <-ticker.C:
			s, err := detector.Detect(paneID)
			if err != nil {
				continue
			}
			if s.State == status.StateIdle {
				return nil
			}
			// Optional: print progress indicator
		}
	}
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	// When n too small for content + ellipsis, just return first n chars
	if n <= 3 {
		// Find last rune boundary at or before n bytes
		lastValid := 0
		for i := range s {
			if i > n {
				break
			}
			lastValid = i
		}
		if lastValid == 0 && len(s) > 0 {
			return ""
		}
		return s[:lastValid]
	}
	// Find the last rune boundary that allows for "..." suffix within n bytes.
	targetLen := n - 3
	prevI := 0
	for i := range s {
		if i > targetLen {
			return s[:prevI] + "..."
		}
		prevI = i
	}
	// All rune starts are <= targetLen, but string is > n bytes.
	return s[:prevI] + "..."
}
