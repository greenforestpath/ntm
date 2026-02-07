package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultContextFiles(t *testing.T) {
	files := defaultContextFiles()
	if len(files) != 3 {
		t.Fatalf("expected 3 default files, got %d", len(files))
	}
	if files[0] != "AGENTS.md" {
		t.Errorf("expected AGENTS.md first, got %s", files[0])
	}
	if files[1] != "README.md" {
		t.Errorf("expected README.md second, got %s", files[1])
	}
	if files[2] != ".claude/project_context.md" {
		t.Errorf("expected .claude/project_context.md third, got %s", files[2])
	}
}

func TestFormatContextInjectContent_BasicFiles(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Agent Rules\nRule 1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Project README\nDescription"), 0644); err != nil {
		t.Fatal(err)
	}

	content, injected, truncated, err := formatContextInjectContent(dir, defaultContextFiles(), 0)
	if err != nil {
		t.Fatal(err)
	}

	if truncated {
		t.Error("expected no truncation")
	}

	if len(injected) != 2 {
		t.Fatalf("expected 2 injected files, got %d: %v", len(injected), injected)
	}

	if injected[0] != "AGENTS.md" {
		t.Errorf("expected AGENTS.md, got %s", injected[0])
	}
	if injected[1] != "README.md" {
		t.Errorf("expected README.md, got %s", injected[1])
	}

	if !strings.Contains(content, "### AGENTS.md") {
		t.Error("content should contain AGENTS.md header")
	}
	if !strings.Contains(content, "# Agent Rules") {
		t.Error("content should contain AGENTS.md body")
	}
	if !strings.Contains(content, "### README.md") {
		t.Error("content should contain README.md header")
	}
	if !strings.Contains(content, "# Project README") {
		t.Error("content should contain README.md body")
	}
}

func TestFormatContextInjectContent_MissingFiles(t *testing.T) {
	dir := t.TempDir()
	// No files created - all should be skipped

	content, injected, truncated, err := formatContextInjectContent(dir, defaultContextFiles(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if truncated {
		t.Error("expected no truncation")
	}
	if len(injected) != 0 {
		t.Errorf("expected 0 injected files, got %d", len(injected))
	}
	if content != "" {
		t.Errorf("expected empty content, got %q", content)
	}
}

func TestFormatContextInjectContent_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("  \n  "), 0644); err != nil {
		t.Fatal(err)
	}

	_, injected, _, err := formatContextInjectContent(dir, defaultContextFiles(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(injected) != 0 {
		t.Errorf("expected 0 injected files for empty content, got %d", len(injected))
	}
}

func TestFormatContextInjectContent_MaxBytesTruncation(t *testing.T) {
	dir := t.TempDir()

	// Create a large file
	largeContent := strings.Repeat("A", 1000)
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(largeContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("Small content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Limit to 200 bytes total
	content, injected, truncated, err := formatContextInjectContent(dir, defaultContextFiles(), 200)
	if err != nil {
		t.Fatal(err)
	}

	if !truncated {
		t.Error("expected truncation")
	}

	// Should have at least AGENTS.md (truncated)
	if len(injected) == 0 {
		t.Fatal("expected at least one injected file")
	}

	if !strings.Contains(content, "...(truncated)") {
		t.Error("truncated content should contain truncation marker")
	}

	if len(content) > 250 { // Some slack for headers
		t.Errorf("content too large: %d bytes", len(content))
	}
}

func TestFormatContextInjectContent_CustomFiles(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "custom.md"), []byte("Custom content"), 0644); err != nil {
		t.Fatal(err)
	}

	content, injected, _, err := formatContextInjectContent(dir, []string{"custom.md"}, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(injected) != 1 {
		t.Fatalf("expected 1 injected file, got %d", len(injected))
	}
	if injected[0] != "custom.md" {
		t.Errorf("expected custom.md, got %s", injected[0])
	}
	if !strings.Contains(content, "### custom.md") {
		t.Error("content should contain custom.md header")
	}
	if !strings.Contains(content, "Custom content") {
		t.Error("content should contain custom.md body")
	}
}

func TestFormatContextInjectContent_NestedFiles(t *testing.T) {
	dir := t.TempDir()

	// Create nested directory structure
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "project_context.md"), []byte("Claude context"), 0644); err != nil {
		t.Fatal(err)
	}

	content, injected, _, err := formatContextInjectContent(dir, defaultContextFiles(), 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(injected) != 1 {
		t.Fatalf("expected 1 injected file, got %d", len(injected))
	}
	if injected[0] != ".claude/project_context.md" {
		t.Errorf("expected .claude/project_context.md, got %s", injected[0])
	}
	if !strings.Contains(content, "Claude context") {
		t.Error("content should contain Claude context body")
	}
}

func TestFormatContextInjectContent_Separator(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("Agents"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("Readme"), 0644); err != nil {
		t.Fatal(err)
	}

	content, _, _, err := formatContextInjectContent(dir, defaultContextFiles(), 0)
	if err != nil {
		t.Fatal(err)
	}

	// Files should be separated by ---
	if !strings.Contains(content, "---") {
		t.Error("multi-file content should contain separator")
	}
}

func TestFormatContextInjectContent_MaxBytesZeroSkip(t *testing.T) {
	dir := t.TempDir()

	// Create files where the first already fills the budget
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(strings.Repeat("X", 500)), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("Readme content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Budget so tight that second file won't fit after first
	_, injected, truncated, err := formatContextInjectContent(dir, defaultContextFiles(), 50)
	if err != nil {
		t.Fatal(err)
	}

	// First file should be truncated, second should be skipped
	if !truncated {
		t.Error("expected truncation")
	}

	if len(injected) > 1 {
		t.Errorf("expected at most 1 file, got %d", len(injected))
	}
}

func TestContextInjectResult_Fields(t *testing.T) {
	r := ContextInjectResult{
		Success:       true,
		Session:       "test",
		InjectedFiles: []string{"AGENTS.md"},
		TotalBytes:    42,
		Truncated:     false,
		PanesInjected: []int{1, 2},
	}

	if !r.Success {
		t.Error("expected success")
	}
	if r.Session != "test" {
		t.Error("wrong session")
	}
	if len(r.InjectedFiles) != 1 {
		t.Error("wrong file count")
	}
	if r.TotalBytes != 42 {
		t.Error("wrong byte count")
	}
	if r.Truncated {
		t.Error("should not be truncated")
	}
	if len(r.PanesInjected) != 2 {
		t.Error("wrong pane count")
	}
}
