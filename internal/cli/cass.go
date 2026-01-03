package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Dicklesworthstone/ntm/internal/cass"
	"github.com/Dicklesworthstone/ntm/internal/output"
	"github.com/Dicklesworthstone/ntm/internal/tui/theme"
)

func newCassCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cass",
		Short: "Interact with CASS (Coding Agent Session Search)",
		Long:  `Search, analyze, and explore past agent sessions indexed by CASS.`,
	}

	cmd.AddCommand(newCassStatusCmd())
	cmd.AddCommand(newCassSearchCmd())
	cmd.AddCommand(newCassInsightsCmd())
	cmd.AddCommand(newCassTimelineCmd())

	return cmd
}

func handleCassError(err error) error {
	if err == cass.ErrNotInstalled {
		if IsJSONOutput() {
			return output.PrintJSON(map[string]interface{}{
				"error": "cass_not_installed",
				"hint":  "Install CASS to enable this feature",
			})
		}
		fmt.Println("CASS is not installed.")
		fmt.Println("To enable cross-agent session search:")
		fmt.Println("  brew install nightowlai/tap/cass    # macOS")
		fmt.Println("  cargo install cass                  # From source")
		return nil
	}
	return err
}

func newCassStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show CASS index health and statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCassStatus()
		},
	}
}

func newCassClient() *cass.Client {
	var opts []cass.ClientOption
	if cfg != nil && cfg.CASS.BinaryPath != "" {
		opts = append(opts, cass.WithBinaryPath(cfg.CASS.BinaryPath))
	}
	return cass.NewClient(opts...)
}

func runCassStatus() error {
	client := newCassClient()
	status, err := client.Status(context.Background())
	if err != nil {
		return handleCassError(err)
	}
	// ... rest of function unchanged

	if IsJSONOutput() {
		return output.PrintJSON(status)
	}

	t := theme.Current()
	fmt.Printf("%sCASS Index Status%s\n", "\033[1m", "\033[0m")
	fmt.Printf("%s%s%s\n\n", "\033[2m", strings.Repeat("─", 40), "\033[0m")

	healthyMark := fmt.Sprintf("%s✓%s", colorize(t.Success), "\033[0m")
	if !status.Healthy {
		healthyMark = fmt.Sprintf("%s✗%s", colorize(t.Error), "\033[0m")
	}

	fmt.Printf("  Healthy:        %s\n", healthyMark)
	fmt.Printf("  Conversations:  %d\n", status.Conversations)
	fmt.Printf("  Messages:       %d\n", status.Messages)
	fmt.Printf("  Last Indexed:   %s\n", formatAge(status.LastIndexedAt.Time))
	fmt.Printf("  Index Size:     %.1f MB\n", status.Index.SizeMB())
	if status.Pending.HasPending() {
		fmt.Printf("  Pending:        %d sessions, %d files\n", status.Pending.Sessions, status.Pending.Files)
	}

	return nil
}

func newCassSearchCmd() *cobra.Command {
	var (
		agent     string
		workspace string
		since     string
		limit     int
		offset    int
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search past agent sessions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCassSearch(args[0], agent, workspace, since, limit, offset)
		},
	}

	cmd.Flags().StringVar(&agent, "agent", "", "Filter by agent type")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Filter by workspace/project")
	cmd.Flags().StringVar(&since, "since", "", "Filter by time (e.g. 7d)")
	cmd.Flags().IntVar(&limit, "limit", 10, "Max results")
	cmd.Flags().IntVar(&offset, "offset", 0, "Result offset")

	return cmd
}

func runCassSearch(query, agent, workspace, since string, limit, offset int) error {
	client := newCassClient()
	resp, err := client.Search(context.Background(), cass.SearchOptions{
		Query:     query,
		Agent:     agent,
		Workspace: workspace,
		Since:     since,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return handleCassError(err)
	}

	if IsJSONOutput() {
		return output.PrintJSON(resp)
	}

	t := theme.Current()
	fmt.Printf("%sSearch Results (%d of %d)%s\n", "\033[1m", resp.Count, resp.TotalMatches, "\033[0m")
	fmt.Printf("%s%s%s\n\n", "\033[2m", strings.Repeat("─", 60), "\033[0m")

	for _, hit := range resp.Hits {
		score := fmt.Sprintf("%.2f", hit.Score)
		fmt.Printf("  %s%s%s (%s)\n", colorize(t.Primary), hit.Title, "\033[0m", hit.Agent)
		fmt.Printf("    %s%s%s • score: %s • %s\n",
			colorize(t.Subtext), hit.Workspace, "\033[0m",
			score, formatAge(hit.CreatedAtTime()))
		if hit.Snippet != "" {
			fmt.Printf("    %s%s%s\n", "\033[2m", strings.TrimSpace(hit.Snippet), "\033[0m")
		}
		fmt.Println()
	}

	return nil
}

func newCassInsightsCmd() *cobra.Command {
	var since string
	cmd := &cobra.Command{
		Use:   "insights",
		Short: "Show insights on agent activity",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCassInsights(since)
		},
	}
	cmd.Flags().StringVar(&since, "since", "7d", "Analysis period")
	return cmd
}

func runCassInsights(since string) error {
	client := newCassClient()
	resp, err := client.Search(context.Background(), cass.SearchOptions{
		Query: "*",
		Since: since,
		Limit: 0,
	})
	if err != nil {
		return handleCassError(err)
	}

	if IsJSONOutput() {
		return output.PrintJSON(resp.Aggregations)
	}

	t := theme.Current()
	fmt.Printf("%sAgent Insights (Since %s)%s\n", "\033[1m", since, "\033[0m")

	if resp.Aggregations != nil {
		printAggregations("Top Agents", resp.Aggregations.Agents, t)
		printAggregations("Top Workspaces", resp.Aggregations.Workspaces, t)
		printAggregations("Common Tags", resp.Aggregations.Tags, t)
	}

	return nil
}

type kv struct {
	Key   string
	Value int
}

func printAggregations(title string, counts map[string]int, t theme.Theme) {
	if len(counts) == 0 {
		return
	}
	fmt.Printf("\n  %s%s%s\n", colorize(t.Info), title, "\033[0m")

	var sorted []kv
	for k, v := range counts {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	for i, item := range sorted {
		if i >= 5 {
			break
		}
		fmt.Printf("    %-20s %d\n", item.Key, item.Value)
	}
}

func newCassTimelineCmd() *cobra.Command {
	var since, groupBy string
	cmd := &cobra.Command{
		Use:   "timeline",
		Short: "Show agent activity over time",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCassTimeline(since, groupBy)
		},
	}
	cmd.Flags().StringVar(&since, "since", "24h", "Timeline period")
	cmd.Flags().StringVar(&groupBy, "group-by", "hour", "Grouping (hour, day)")
	return cmd
}

func runCassTimeline(since, groupBy string) error {
	client := newCassClient()
	resp, err := client.Timeline(context.Background(), since, groupBy)
	if err != nil {
		return handleCassError(err)
	}

	if IsJSONOutput() {
		return output.PrintJSON(resp)
	}

	t := theme.Current()
	fmt.Printf("%sActivity Timeline (%s)%s\n", colorize(t.Primary), since, "\033[0m")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Time\tType\tCount")
	fmt.Fprintln(w, "────\t────\t─────")

	for _, entry := range resp.Entries {
		ts := entry.TimestampTime().Format("15:04")
		if groupBy == "day" {
			ts = entry.TimestampTime().Format("Jan 02")
		}

		count := 1
		if c, ok := entry.Data.(float64); ok {
			count = int(c)
		} else if m, ok := entry.Data.(map[string]interface{}); ok {
			if c, ok := m["count"].(float64); ok {
				count = int(c)
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%d\n", ts, entry.Type, count)
	}
	w.Flush()

	return nil
}
