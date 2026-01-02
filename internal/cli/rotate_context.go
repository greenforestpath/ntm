package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	ctxmon "github.com/Dicklesworthstone/ntm/internal/context"
	"github.com/Dicklesworthstone/ntm/internal/output"
	"github.com/Dicklesworthstone/ntm/internal/tui/theme"
)

func newRotateContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Context window rotation management",
		Long: `Commands for viewing and managing context window rotation history.

Context rotation automatically replaces agents when their context window
approaches capacity, preserving work in progress.

Examples:
  ntm rotate context history              # View recent rotations
  ntm rotate context history --last=20    # View last 20 rotations
  ntm rotate context history myproject    # Rotations for a session
  ntm rotate context history --failed     # Only failed rotations
  ntm rotate context stats                # Rotation statistics`,
	}

	cmd.AddCommand(newRotateContextHistoryCmd())
	cmd.AddCommand(newRotateContextStatsCmd())
	cmd.AddCommand(newRotateContextClearCmd())

	return cmd
}

func newRotateContextHistoryCmd() *cobra.Command {
	var (
		limit  int
		failed bool
	)

	cmd := &cobra.Command{
		Use:   "history [session]",
		Short: "View context rotation history",
		Long: `View the history of context window rotations.

Examples:
  ntm rotate context history              # All recent rotations
  ntm rotate context history --last=20    # Last 20 rotations
  ntm rotate context history myproject    # Rotations for a session
  ntm rotate context history --failed     # Only failed rotations`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContextRotationHistory(args, limit, failed)
		},
	}

	cmd.Flags().IntVar(&limit, "last", 20, "Number of recent rotations to show")
	cmd.Flags().BoolVar(&failed, "failed", false, "Show only failed rotations")

	return cmd
}

func newRotateContextStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show context rotation statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContextRotationStats()
		},
	}
}

func newRotateContextClearCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear context rotation history",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContextRotationClear(force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation")

	return cmd
}

// RotationHistoryResult contains the rotation history output
type RotationHistoryResult struct {
	Records    []ctxmon.RotationRecord `json:"records"`
	TotalCount int                     `json:"total_count"`
	Showing    int                     `json:"showing"`
}

func (r *RotationHistoryResult) Text(w io.Writer) error {
	t := theme.Current()

	if len(r.Records) == 0 {
		fmt.Fprintf(w, "%sNo context rotations found%s\n", colorize(t.Warning), colorize(t.Text))
		return nil
	}

	// Print header
	fmt.Fprintf(w, " %sTIME%s          %sSESSION%s       %sAGENT%s      %sCONTEXT%s  %sRESULT%s\n",
		colorize(t.Surface1), colorize(t.Text),
		colorize(t.Surface1), colorize(t.Text),
		colorize(t.Surface1), colorize(t.Text),
		colorize(t.Surface1), colorize(t.Text),
		colorize(t.Surface1), colorize(t.Text))

	// Print entries in reverse order (newest first)
	for i := len(r.Records) - 1; i >= 0; i-- {
		rec := r.Records[i]

		// Format time
		timeStr := rec.Timestamp.Local().Format("Jan 02 15:04")

		// Format session (truncate if needed)
		sessionName := truncateStr(rec.SessionName, 12)

		// Format agent (truncate if needed)
		agentID := truncateStr(rec.AgentID, 10)

		// Context usage
		contextStr := fmt.Sprintf("%.0f%%", rec.ContextBefore)

		// Result indicator
		resultStr := colorize(t.Success) + "✓" + colorize(t.Text)
		if !rec.Success {
			resultStr = colorize(t.Red) + "✗" + colorize(t.Text)
			if rec.FailureReason != "" {
				resultStr += " " + truncateStr(rec.FailureReason, 20)
			}
		}

		fmt.Fprintf(w, " %-14s %-13s %-10s %7s  %s\n",
			timeStr,
			sessionName,
			agentID,
			contextStr,
			resultStr)
	}

	if r.TotalCount > r.Showing {
		fmt.Fprintf(w, "\n%sShowing %d of %d rotations. Use --last for more.%s\n",
			colorize(t.Surface1), r.Showing, r.TotalCount, colorize(t.Text))
	}

	return nil
}

func (r *RotationHistoryResult) JSON() interface{} {
	return r
}

func runContextRotationHistory(args []string, limit int, failedOnly bool) error {
	store := ctxmon.DefaultRotationHistoryStore

	var records []ctxmon.RotationRecord
	var err error

	// Determine how to fetch records based on filters
	if len(args) > 0 {
		// Filter by session first
		records, err = store.ReadForSession(args[0])
		if err != nil {
			return err
		}
		// Additionally filter by failed if specified
		if failedOnly {
			var filtered []ctxmon.RotationRecord
			for _, r := range records {
				if !r.Success {
					filtered = append(filtered, r)
				}
			}
			records = filtered
		}
	} else if failedOnly {
		records, err = store.ReadFailed()
	} else {
		records, err = store.ReadRecent(limit)
	}

	if err != nil {
		return err
	}

	totalCount := len(records)

	// Apply limit
	showing := len(records)
	if len(records) > limit {
		records = records[len(records)-limit:]
		showing = limit
	}

	result := &RotationHistoryResult{
		Records:    records,
		TotalCount: totalCount,
		Showing:    showing,
	}

	formatter := output.New(output.WithJSON(jsonOutput))
	return formatter.Output(result)
}

// RotationStatsResult contains the rotation statistics output
type RotationStatsResult struct {
	*ctxmon.RotationStats
}

func (r *RotationStatsResult) Text(w io.Writer) error {
	t := theme.Current()
	s := r.RotationStats

	fmt.Fprintf(w, "%sContext Rotation Statistics%s\n", colorize(t.Blue), colorize(t.Text))
	fmt.Fprintf(w, "─────────────────────────────────\n")
	fmt.Fprintf(w, "Total rotations:      %d\n", s.TotalRotations)
	fmt.Fprintf(w, "Successful:           %s%d%s\n", colorize(t.Success), s.SuccessCount, colorize(t.Text))
	fmt.Fprintf(w, "Failed:               %s%d%s\n", colorize(t.Red), s.FailureCount, colorize(t.Text))
	fmt.Fprintf(w, "Unique sessions:      %d\n", s.UniqueSessions)

	if s.TotalRotations > 0 {
		fmt.Fprintf(w, "\n%sContext at rotation:%s  %.1f%% avg\n", colorize(t.Blue), colorize(t.Text), s.AvgContextBefore)
		fmt.Fprintf(w, "%sRotation duration:%s    %dms avg\n", colorize(t.Blue), colorize(t.Text), s.AvgDurationMs)
	}

	if s.ThresholdRotations > 0 || s.ManualRotations > 0 {
		fmt.Fprintf(w, "\n%sBy trigger:%s\n", colorize(t.Blue), colorize(t.Text))
		fmt.Fprintf(w, "  Threshold:          %d\n", s.ThresholdRotations)
		fmt.Fprintf(w, "  Manual:             %d\n", s.ManualRotations)
	}

	if s.CompactionAttempts > 0 {
		fmt.Fprintf(w, "\n%sCompaction:%s\n", colorize(t.Blue), colorize(t.Text))
		fmt.Fprintf(w, "  Attempts:           %d\n", s.CompactionAttempts)
		fmt.Fprintf(w, "  Successes:          %d\n", s.CompactionSuccesses)
	}

	if len(s.RotationsByAgentType) > 0 {
		fmt.Fprintf(w, "\n%sBy agent type:%s\n", colorize(t.Blue), colorize(t.Text))
		for agentType, count := range s.RotationsByAgentType {
			fmt.Fprintf(w, "  %-18s %d\n", agentType+":", count)
		}
	}

	fmt.Fprintf(w, "\nFile size:            %s\n", formatBytes(s.FileSizeBytes))
	fmt.Fprintf(w, "File location:        %s\n", ctxmon.RotationHistoryStoragePath())

	return nil
}

func (r *RotationStatsResult) JSON() interface{} {
	return r.RotationStats
}

func runContextRotationStats() error {
	stats, err := ctxmon.GetRotationStats()
	if err != nil {
		return err
	}

	result := &RotationStatsResult{RotationStats: stats}
	formatter := output.New(output.WithJSON(jsonOutput))
	return formatter.Output(result)
}

func runContextRotationClear(force bool) error {
	t := theme.Current()

	store := ctxmon.DefaultRotationHistoryStore
	count, err := store.Count()
	if err != nil {
		return err
	}

	if count == 0 {
		fmt.Printf("%sNo rotation history to clear%s\n", colorize(t.Warning), colorize(t.Text))
		return nil
	}

	if !force {
		fmt.Printf("This will remove all %d rotation history entries. Continue? [y/N]: ", count)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := store.Clear(); err != nil {
		return err
	}

	fmt.Printf("%s✓%s Cleared %d rotation history entries\n", colorize(t.Success), colorize(t.Text), count)
	return nil
}

// Note: truncateStr is defined in checkpoint.go
