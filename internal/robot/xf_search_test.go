package robot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetXFSearch_MissingBinary(t *testing.T) {
	t.Setenv("PATH", "")

	output, err := GetXFSearch(XFSearchOptions{Query: "test query"})
	if err != nil {
		t.Fatalf("GetXFSearch returned error: %v", err)
	}
	if output.Success {
		t.Fatalf("expected failure when xf missing")
	}
	if output.ErrorCode != ErrCodeDependencyMissing {
		t.Fatalf("expected %s, got %s", ErrCodeDependencyMissing, output.ErrorCode)
	}
}

func TestGetXFSearch_EmptyQuery(t *testing.T) {
	output, err := GetXFSearch(XFSearchOptions{Query: ""})
	if err != nil {
		t.Fatalf("GetXFSearch returned error: %v", err)
	}
	if output.Success {
		t.Fatalf("expected failure when query empty")
	}
	if output.ErrorCode != ErrCodeInvalidFlag {
		t.Fatalf("expected %s, got %s", ErrCodeInvalidFlag, output.ErrorCode)
	}
}

func TestGetXFSearch_WhitespaceQuery(t *testing.T) {
	output, err := GetXFSearch(XFSearchOptions{Query: "   "})
	if err != nil {
		t.Fatalf("GetXFSearch returned error: %v", err)
	}
	if output.Success {
		t.Fatalf("expected failure when query is whitespace")
	}
	if output.ErrorCode != ErrCodeInvalidFlag {
		t.Fatalf("expected %s, got %s", ErrCodeInvalidFlag, output.ErrorCode)
	}
}

func TestGetXFSearch_Success(t *testing.T) {
	tmpDir := t.TempDir()

	results := `[{"id":"tweet-123","content":"error handling in go","created_at":"2024-01-15","type":"tweet","score":0.95}]`
	script := fmt.Sprintf(`#!/bin/sh
if echo "$@" | grep -q -- "--version"; then
  echo "xf 0.2.1"
  exit 0
fi
echo '%s'
`, strings.ReplaceAll(results, "'", "'\\''"))

	stubPath := filepath.Join(tmpDir, "xf")
	if err := os.WriteFile(stubPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write xf stub: %v", err)
	}
	t.Setenv("PATH", tmpDir)

	output, err := GetXFSearch(XFSearchOptions{
		Query: "error handling",
		Limit: 10,
		Mode:  "semantic",
		Sort:  "relevance",
	})
	if err != nil {
		t.Fatalf("GetXFSearch returned error: %v", err)
	}
	if !output.Success {
		t.Fatalf("expected success, got error: %s (code: %s)", output.Error, output.ErrorCode)
	}
	if output.Query != "error handling" {
		t.Fatalf("expected query 'error handling', got %q", output.Query)
	}
	if output.Count != 1 {
		t.Fatalf("expected count 1, got %d", output.Count)
	}
	if output.Mode != "semantic" {
		t.Fatalf("expected mode 'semantic', got %q", output.Mode)
	}
	if output.Sort != "relevance" {
		t.Fatalf("expected sort 'relevance', got %q", output.Sort)
	}
	if len(output.Hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(output.Hits))
	}
	if output.Hits[0].ID != "tweet-123" {
		t.Fatalf("expected hit ID 'tweet-123', got %q", output.Hits[0].ID)
	}
}

func TestGetXFSearch_DefaultLimit(t *testing.T) {
	tmpDir := t.TempDir()

	script := `#!/bin/sh
if echo "$@" | grep -q -- "--version"; then
  echo "xf 0.2.1"
  exit 0
fi
echo '[]'
`
	stubPath := filepath.Join(tmpDir, "xf")
	if err := os.WriteFile(stubPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write xf stub: %v", err)
	}
	t.Setenv("PATH", tmpDir)

	output, err := GetXFSearch(XFSearchOptions{Query: "test"})
	if err != nil {
		t.Fatalf("GetXFSearch returned error: %v", err)
	}
	if !output.Success {
		t.Fatalf("expected success, got error: %s", output.Error)
	}
	if output.Count != 0 {
		t.Fatalf("expected 0 results, got %d", output.Count)
	}
}

func TestGetXFSearch_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()

	script := `#!/bin/sh
if echo "$@" | grep -q -- "--version"; then
  echo "xf 0.2.1"
  exit 0
fi
echo '[]'
`
	stubPath := filepath.Join(tmpDir, "xf")
	if err := os.WriteFile(stubPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write xf stub: %v", err)
	}
	t.Setenv("PATH", tmpDir)

	output, err := GetXFSearch(XFSearchOptions{Query: "test"})
	if err != nil {
		t.Fatalf("GetXFSearch returned error: %v", err)
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal output: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	for _, key := range []string{"success", "timestamp", "query", "count", "hits"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("missing expected key %q in JSON output", key)
		}
	}
}

func TestGetXFStatus_MissingBinary(t *testing.T) {
	t.Setenv("PATH", "")

	output, err := GetXFStatus()
	if err != nil {
		t.Fatalf("GetXFStatus returned error: %v", err)
	}
	if output.Success {
		t.Fatalf("expected failure when xf missing")
	}
	if output.ErrorCode != ErrCodeDependencyMissing {
		t.Fatalf("expected %s, got %s", ErrCodeDependencyMissing, output.ErrorCode)
	}
	if output.XFAvailable {
		t.Fatalf("expected xf_available=false")
	}
}
