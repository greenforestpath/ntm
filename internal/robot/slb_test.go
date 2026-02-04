package robot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func fakeToolsPath(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for dir := wd; dir != "/"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			fakePath := filepath.Join(dir, "testdata", "faketools")
			if _, err := os.Stat(fakePath); err == nil {
				return fakePath
			}
			break
		}
	}

	return ""
}

func withFakeTools(t *testing.T) func() {
	t.Helper()

	fakePath := fakeToolsPath(t)
	if fakePath == "" {
		t.Skip("testdata/faketools not found")
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", fakePath+":"+oldPath)

	return func() {
		os.Setenv("PATH", oldPath)
	}
}

func TestGetSLBPending(t *testing.T) {
	cleanup := withFakeTools(t)
	defer cleanup()

	output, err := GetSLBPending()
	if err != nil {
		t.Fatalf("GetSLBPending error: %v", err)
	}

	if !output.Success {
		t.Fatalf("expected success, got error: %s", output.Error)
	}

	var pending []map[string]interface{}
	if err := json.Unmarshal(output.Pending, &pending); err != nil {
		t.Fatalf("pending payload invalid JSON: %v", err)
	}
	if len(pending) != output.Count {
		t.Fatalf("count mismatch: count=%d pending=%d", output.Count, len(pending))
	}
}

func TestGetSLBApprove(t *testing.T) {
	cleanup := withFakeTools(t)
	defer cleanup()

	output, err := GetSLBApprove("req-123")
	if err != nil {
		t.Fatalf("GetSLBApprove error: %v", err)
	}
	if !output.Success {
		t.Fatalf("expected success, got error: %s", output.Error)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(output.Result, &payload); err != nil {
		t.Fatalf("approve payload invalid JSON: %v", err)
	}
	if got, _ := payload["id"].(string); got != "req-123" {
		t.Fatalf("approve id=%q, want req-123", got)
	}
}

func TestGetSLBDeny(t *testing.T) {
	cleanup := withFakeTools(t)
	defer cleanup()

	output, err := GetSLBDeny("req-456", "too risky")
	if err != nil {
		t.Fatalf("GetSLBDeny error: %v", err)
	}
	if !output.Success {
		t.Fatalf("expected success, got error: %s", output.Error)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(output.Result, &payload); err != nil {
		t.Fatalf("deny payload invalid JSON: %v", err)
	}
	if got, _ := payload["id"].(string); got != "req-456" {
		t.Fatalf("deny id=%q, want req-456", got)
	}
}

func TestGetSLBApproveMissingID(t *testing.T) {
	output, err := GetSLBApprove("")
	if err != nil {
		t.Fatalf("GetSLBApprove error: %v", err)
	}
	if output.Success {
		t.Fatalf("expected failure for missing ID")
	}
	if output.ErrorCode != ErrCodeInvalidFlag {
		t.Fatalf("error_code=%q, want %q", output.ErrorCode, ErrCodeInvalidFlag)
	}
}
