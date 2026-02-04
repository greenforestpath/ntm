package robot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetGIILFetch_MissingBinary(t *testing.T) {
	t.Setenv("PATH", "")

	output, err := GetGIILFetch("https://share.icloud.com/photos/abc123")
	if err != nil {
		t.Fatalf("GetGIILFetch returned error: %v", err)
	}
	if output.Success {
		t.Fatalf("expected failure when giil missing")
	}
	if output.ErrorCode != ErrCodeDependencyMissing {
		t.Fatalf("expected %s, got %s", ErrCodeDependencyMissing, output.ErrorCode)
	}
}

func TestGetGIILFetch_EmptyURL(t *testing.T) {
	tmpDir := t.TempDir()
	stubPath := filepath.Join(tmpDir, "giil")
	if err := os.WriteFile(stubPath, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write giil stub: %v", err)
	}
	t.Setenv("PATH", tmpDir)

	output, err := GetGIILFetch(" ")
	if err != nil {
		t.Fatalf("GetGIILFetch returned error: %v", err)
	}
	if output.Success {
		t.Fatalf("expected failure when url empty")
	}
	if output.ErrorCode != ErrCodeInvalidFlag {
		t.Fatalf("expected %s, got %s", ErrCodeInvalidFlag, output.ErrorCode)
	}
}

func TestGetGIILFetch_Success(t *testing.T) {
	tmpDir := t.TempDir()
	imagePath := filepath.Join(tmpDir, "image.jpg")
	content := []byte("giil-test-bytes")
	if err := os.WriteFile(imagePath, content, 0644); err != nil {
		t.Fatalf("failed to write image file: %v", err)
	}

	script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "--version" ]; then
  echo "giil version 3.1.0"
  exit 0
fi
echo '{"path":"%s","filename":"image.jpg","width":1024,"height":768,"platform":"icloud"}'
`, strings.ReplaceAll(imagePath, `"`, `\"`))

	stubPath := filepath.Join(tmpDir, "giil")
	if err := os.WriteFile(stubPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write giil stub: %v", err)
	}
	t.Setenv("PATH", tmpDir)

	url := "https://share.icloud.com/photos/abc123"
	output, err := GetGIILFetch(url)
	if err != nil {
		t.Fatalf("GetGIILFetch returned error: %v", err)
	}
	if !output.Success {
		t.Fatalf("expected success, got error: %s", output.Error)
	}
	if output.SourceURL != url {
		t.Fatalf("expected source_url %s, got %s", url, output.SourceURL)
	}
	if output.Path != imagePath {
		t.Fatalf("expected path %s, got %s", imagePath, output.Path)
	}
	if output.Filename != "image.jpg" {
		t.Fatalf("expected filename image.jpg, got %s", output.Filename)
	}
	if output.SizeBytes != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), output.SizeBytes)
	}
	if output.Width != 1024 || output.Height != 768 {
		t.Fatalf("expected dimensions 1024x768, got %dx%d", output.Width, output.Height)
	}
	if output.Platform != "icloud" {
		t.Fatalf("expected platform icloud, got %s", output.Platform)
	}
}
