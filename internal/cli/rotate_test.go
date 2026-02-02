package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRotateCmdValidation(t *testing.T) {
	tests := []struct {
		name                     string
		args                     []string
		flags                    map[string]string
		wantError                string
		wantErrorAny             []string
		skipIfAutoSelectPossible bool // Skip if exactly one session is running (auto-select applies)
	}{
		{
			name:                     "missing session and not in tmux",
			args:                     []string{},
			wantError:                "session",
			skipIfAutoSelectPossible: true, // Session auto-selected when only one exists
		},
		{
			name: "missing pane index",
			args: []string{"mysession"},
			wantErrorAny: []string{
				"pane index required",
				"session", // session may not exist in shared tmux environment
			},
		},
		{
			name: "dry run requires valid session/pane",
			args: []string{"mysession"},
			flags: map[string]string{
				"pane":    "0",
				"dry-run": "true",
			},
			// Dry run still needs to look up pane info, which fails without tmux
			wantErrorAny: []string{
				"getting panes",
				"session", // session may not exist in shared tmux environment
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Change to a temp dir to prevent CWD-based session inference
			tmpDir := t.TempDir()
			oldWd, _ := os.Getwd()
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("chdir failed: %v", err)
			}
			defer os.Chdir(oldWd)

			// Unset TMUX env var to prevent auto-detection from environment
			oldTmux := os.Getenv("TMUX")
			os.Unsetenv("TMUX")
			defer os.Setenv("TMUX", oldTmux)

			if tt.skipIfAutoSelectPossible && sessionAutoSelectPossible() {
				t.Skip("Skipping: exactly one tmux session running (auto-selection applies)")
			}

			cmd := newRotateCmd()
			// Redirect output to buffer to ensure non-interactive mode
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Set args
			if len(tt.args) > 0 {
				cmd.SetArgs(tt.args)
			} else {
				cmd.SetArgs([]string{})
			}

			// Set flags
			for k, v := range tt.flags {
				_ = cmd.Flags().Set(k, v)
			}

			// Execute
			err := cmd.Execute()

			if tt.wantError != "" || len(tt.wantErrorAny) > 0 {
				if err == nil {
					if tt.wantError != "" {
						t.Errorf("expected error containing %q, got nil", tt.wantError)
					} else {
						t.Errorf("expected error containing one of %q, got nil", tt.wantErrorAny)
					}
				} else if !errorMatchesAny(err.Error(), append(tt.wantErrorAny, tt.wantError)) {
					if tt.wantError != "" {
						t.Errorf("expected error containing %q, got %q", tt.wantError, err.Error())
					} else {
						t.Errorf("expected error containing one of %q, got %q", tt.wantErrorAny, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func errorMatchesAny(err string, matches []string) bool {
	for _, match := range matches {
		if match == "" {
			continue
		}
		if strings.Contains(err, match) {
			return true
		}
	}
	return false
}
