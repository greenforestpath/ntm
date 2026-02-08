package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Dicklesworthstone/ntm/internal/config"
	"github.com/Dicklesworthstone/ntm/internal/output"
	"github.com/Dicklesworthstone/ntm/internal/redaction"
	"github.com/Dicklesworthstone/ntm/internal/util"
)

type RedactPreviewFinding struct {
	Category redaction.Category `json:"category"`
	Redacted string             `json:"redacted"`
	Start    int                `json:"start"`
	End      int                `json:"end"`
	Line     int                `json:"line,omitempty"`
	Column   int                `json:"column,omitempty"`
}

type RedactPreviewResponse struct {
	output.TimestampedResponse

	Source   string                 `json:"source"`         // text|file
	Path     string                 `json:"path,omitempty"` // only when source=file
	InputLen int                    `json:"input_len"`      // bytes
	Findings []RedactPreviewFinding `json:"findings"`       // never includes raw matches
	Output   string                 `json:"output"`         // redacted output (safe)
}

func newRedactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redact",
		Short: "Redaction utilities",
		Long: `Redaction utilities for previewing and debugging secret detection.

These commands NEVER print raw matched secrets. Output is always safe-redacted.`,
	}

	cmd.AddCommand(
		newRedactPreviewCmd(),
	)

	return cmd
}

func newRedactPreviewCmd() *cobra.Command {
	var (
		text string
		file string
	)

	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Preview redaction findings and safe-redacted output",
		Long: `Preview secret detection on input text (or a file) and print:
- A list of findings (category + position + placeholder)
- A safe-redacted output

This command never prints raw matched secrets, even if your configured redaction mode is warn/off.

Examples:
  ntm redact preview --text "password=hunter2hunter2"
  ntm redact preview --file ./notes.txt
  ntm redact preview --text "..." --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Cobra commands are reused across tests within the same process; flags bound via
			// StringVar can retain values between Execute() calls when a flag is omitted.
			// Snapshot and then reset to keep behavior deterministic for both tests and CLI.
			currentText := text
			currentFile := file
			text = ""
			file = ""

			if currentText == "" && currentFile == "" {
				return fmt.Errorf("must provide exactly one of --text or --file")
			}
			if currentText != "" && currentFile != "" {
				return fmt.Errorf("flags --text and --file are mutually exclusive")
			}

			source := "text"
			absPath := ""
			input := currentText
			if currentFile != "" {
				source = "file"
				p := util.ExpandPath(currentFile)
				abs, err := filepath.Abs(p)
				if err != nil {
					return fmt.Errorf("resolve --file %q: %w", currentFile, err)
				}
				b, err := os.ReadFile(abs)
				if err != nil {
					return fmt.Errorf("read %q: %w", abs, err)
				}
				absPath = abs
				input = string(b)
			}

			if cfg == nil {
				cfg = config.Default()
			}

			// Always compute a safe-redacted output for preview. This prevents accidental leaks
			// when the global config/flags are set to warn/off.
			redactCfg := cfg.Redaction.ToRedactionLibConfig()
			redactCfg.Mode = redaction.ModeRedact

			res := redaction.ScanAndRedact(input, redactCfg)
			redaction.AddLineInfo(input, res.Findings)

			findings := make([]RedactPreviewFinding, 0, len(res.Findings))
			for _, f := range res.Findings {
				findings = append(findings, RedactPreviewFinding{
					Category: f.Category,
					Redacted: f.Redacted,
					Start:    f.Start,
					End:      f.End,
					Line:     f.Line,
					Column:   f.Column,
				})
			}

			resp := RedactPreviewResponse{
				TimestampedResponse: output.NewTimestamped(),
				Source:              source,
				Path:                absPath,
				InputLen:            len(input),
				Findings:            findings,
				Output:              res.Output,
			}

			if IsJSONOutput() {
				return output.PrintJSON(resp)
			}

			if resp.Source == "file" {
				fmt.Printf("Source: %s\n", resp.Path)
			} else {
				fmt.Println("Source: text")
			}
			fmt.Printf("Findings: %d\n", len(resp.Findings))
			for _, f := range resp.Findings {
				if f.Line > 0 && f.Column > 0 {
					fmt.Printf("- %d:%d %s %s\n", f.Line, f.Column, f.Category, f.Redacted)
					continue
				}
				fmt.Printf("- %d-%d %s %s\n", f.Start, f.End, f.Category, f.Redacted)
			}
			fmt.Println()
			fmt.Println("Redacted output:")
			fmt.Print(resp.Output)
			if resp.Output != "" && !strings.HasSuffix(resp.Output, "\n") {
				fmt.Println()
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&text, "text", "", "Input text to scan/redact (mutually exclusive with --file)")
	cmd.Flags().StringVar(&file, "file", "", "File to scan/redact (mutually exclusive with --text)")

	return cmd
}
