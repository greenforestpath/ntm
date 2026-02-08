package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPromptsConfig_ResolveForType_Unknown(t *testing.T) {
	t.Parallel()

	got, err := (PromptsConfig{}).ResolveForType("unknown")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestPromptsConfig_ResolveForType_InlineString(t *testing.T) {
	t.Parallel()

	p := PromptsConfig{
		CCDefault: "\nhello\n",
	}
	got, err := p.ResolveForType("cc")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if want := "hello"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPromptsConfig_ResolveForType_FileFallback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "cc_default.txt")
	if err := os.WriteFile(path, []byte("\nfrom file\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	p := PromptsConfig{
		CCDefaultFile: path,
	}
	got, err := p.ResolveForType("cc")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if want := "from file"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPromptsConfig_ResolveForType_InlineOverridesFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "cc_default.txt")
	if err := os.WriteFile(path, []byte("from file"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	p := PromptsConfig{
		CCDefault:     "from inline",
		CCDefaultFile: path,
	}
	got, err := p.ResolveForType("cc")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if want := "from inline"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPromptsConfig_ResolveForType_MissingFile(t *testing.T) {
	t.Parallel()

	p := PromptsConfig{
		CCDefaultFile: filepath.Join(t.TempDir(), "does_not_exist.txt"),
	}
	if _, err := p.ResolveForType("cc"); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
