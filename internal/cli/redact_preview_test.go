package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactPreview_CommandJSONOutputIsRedacted(t *testing.T) {
	resetFlags()

	// Avoid reading any user config by isolating XDG config home.
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Ensure deterministic behavior regardless of prior tests.
	oldCfg := cfg
	cfg = nil
	t.Cleanup(func() { cfg = oldCfg })

	input := "prefix password=hunter2hunter2 suffix\n"

	out, err := captureStdout(t, func() error {
		rootCmd.SetArgs([]string{"redact", "preview", "--text", input, "--json"})
		return rootCmd.Execute()
	})
	if err != nil {
		t.Fatalf("redact preview failed: %v\noutput:\n%s", err, out)
	}

	if strings.Contains(out, "hunter2hunter2") {
		t.Fatalf("expected JSON output to be redacted, got:\n%s", out)
	}
	if strings.Contains(out, "\"match\"") {
		t.Fatalf("expected JSON output to never include raw matches, got:\n%s", out)
	}

	var resp RedactPreviewResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput:\n%s", err, out)
	}

	if resp.Source != "text" {
		t.Fatalf("expected source=text, got %q", resp.Source)
	}
	if resp.Path != "" {
		t.Fatalf("expected empty path for source=text, got %q", resp.Path)
	}
	if resp.InputLen != len(input) {
		t.Fatalf("expected input_len=%d, got %d", len(input), resp.InputLen)
	}
	if len(resp.Findings) == 0 {
		t.Fatalf("expected at least 1 finding")
	}
	if strings.Contains(resp.Output, "hunter2hunter2") {
		t.Fatalf("expected output field to be redacted, got %q", resp.Output)
	}
	if !strings.Contains(resp.Output, "[REDACTED:PASSWORD:") {
		t.Fatalf("expected password placeholder, got %q", resp.Output)
	}
}

func TestRedactPreview_CommandTextOutputIsRedacted(t *testing.T) {
	resetFlags()

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	oldCfg := cfg
	cfg = nil
	t.Cleanup(func() { cfg = oldCfg })

	out, err := captureStdout(t, func() error {
		rootCmd.SetArgs([]string{"redact", "preview", "--text", "password=hunter2hunter2"})
		return rootCmd.Execute()
	})
	if err != nil {
		t.Fatalf("redact preview failed: %v\noutput:\n%s", err, out)
	}
	if strings.Contains(out, "hunter2hunter2") {
		t.Fatalf("expected text output to be redacted, got:\n%s", out)
	}
	if !strings.Contains(out, "[REDACTED:PASSWORD:") {
		t.Fatalf("expected placeholder in output, got:\n%s", out)
	}
}

func TestRedactPreview_CommandFileInputIsRedacted(t *testing.T) {
	resetFlags()

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	artifactPath := filepath.Join(tmpDir, "artifact.txt")
	if err := os.WriteFile(artifactPath, []byte("password=hunter2hunter2\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	oldCfg := cfg
	cfg = nil
	t.Cleanup(func() { cfg = oldCfg })

	out, err := captureStdout(t, func() error {
		rootCmd.SetArgs([]string{"redact", "preview", "--file", artifactPath, "--json"})
		return rootCmd.Execute()
	})
	if err != nil {
		t.Fatalf("redact preview failed: %v\noutput:\n%s", err, out)
	}

	if strings.Contains(out, "hunter2hunter2") {
		t.Fatalf("expected JSON output to be redacted, got:\n%s", out)
	}

	var resp RedactPreviewResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput:\n%s", err, out)
	}

	if resp.Source != "file" {
		t.Fatalf("expected source=file, got %q", resp.Source)
	}
	abs, err := filepath.Abs(artifactPath)
	if err != nil {
		t.Fatalf("Abs failed: %v", err)
	}
	if resp.Path != abs {
		t.Fatalf("expected path %q, got %q", abs, resp.Path)
	}
	if strings.Contains(resp.Output, "hunter2hunter2") {
		t.Fatalf("expected output field to be redacted, got %q", resp.Output)
	}
}

func TestRedactPreview_RequiresTextOrFile(t *testing.T) {
	resetFlags()

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	oldCfg := cfg
	cfg = nil
	t.Cleanup(func() { cfg = oldCfg })

	_, err := captureStdout(t, func() error {
		rootCmd.SetArgs([]string{"redact", "preview", "--json"})
		return rootCmd.Execute()
	})
	if err == nil {
		t.Fatalf("expected error when neither --text nor --file is provided")
	}
}

func TestRedactPreview_TextAndFileMutuallyExclusive(t *testing.T) {
	resetFlags()

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	artifactPath := filepath.Join(tmpDir, "artifact.txt")
	if err := os.WriteFile(artifactPath, []byte("password=hunter2hunter2\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	oldCfg := cfg
	cfg = nil
	t.Cleanup(func() { cfg = oldCfg })

	_, err := captureStdout(t, func() error {
		rootCmd.SetArgs([]string{"redact", "preview", "--text", "hi", "--file", artifactPath, "--json"})
		return rootCmd.Execute()
	})
	if err == nil {
		t.Fatalf("expected error when both --text and --file are provided")
	}
}
