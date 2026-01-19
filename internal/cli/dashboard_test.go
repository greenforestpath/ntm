package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestIsCI(t *testing.T) {
	// Save original CI value
	originalCI := os.Getenv("CI")
	defer os.Setenv("CI", originalCI)

	tests := []struct {
		name     string
		ciValue  string
		expected bool
	}{
		{"CI=1", "1", true},
		{"CI=true", "true", true},
		{"CI=TRUE", "TRUE", true},
		{"CI=false", "false", false},
		{"CI=0", "0", false},
		{"CI empty", "", false},
		{"CI=yes", "yes", false}, // Only 1, true, TRUE are recognized
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("CI", tc.ciValue)
			result := isCI()
			if result != tc.expected {
				t.Errorf("isCI() with CI=%q: expected %v, got %v", tc.ciValue, tc.expected, result)
			}
		})
	}
}

func TestDashboardStatusJSON(t *testing.T) {
	// Test that DashboardStatus can be properly marshaled to JSON
	status := DashboardStatus{
		Session:    "test-session",
		ProjectDir: "/data/projects/test",
		Fleet: FleetStatus{
			Total:   5,
			Active:  4,
			Claude:  2,
			Codex:   1,
			Gemini:  1,
			User:    0,
			Unknown: 1,
		},
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal DashboardStatus: %v", err)
	}

	// Verify it can be unmarshaled back
	var unmarshaled DashboardStatus
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal DashboardStatus: %v", err)
	}

	if unmarshaled.Session != status.Session {
		t.Errorf("Session mismatch: expected %q, got %q", status.Session, unmarshaled.Session)
	}
	if unmarshaled.Fleet.Total != status.Fleet.Total {
		t.Errorf("Fleet.Total mismatch: expected %d, got %d", status.Fleet.Total, unmarshaled.Fleet.Total)
	}
	if unmarshaled.Fleet.Claude != status.Fleet.Claude {
		t.Errorf("Fleet.Claude mismatch: expected %d, got %d", status.Fleet.Claude, unmarshaled.Fleet.Claude)
	}
}

func TestDashboardStatusJSONFormat(t *testing.T) {
	// Verify JSON output format matches expected schema
	status := DashboardStatus{
		Session:    "my-session",
		ProjectDir: "/home/user/projects/my-session",
		Fleet: FleetStatus{
			Total:   3,
			Active:  2,
			Claude:  1,
			Codex:   1,
			Gemini:  0,
			User:    0,
			Unknown: 1,
		},
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(status); err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Parse and verify structure
	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify required fields exist
	if _, ok := parsed["session"]; !ok {
		t.Error("JSON missing 'session' field")
	}
	if _, ok := parsed["project_dir"]; !ok {
		t.Error("JSON missing 'project_dir' field")
	}
	if _, ok := parsed["fleet"]; !ok {
		t.Error("JSON missing 'fleet' field")
	}

	fleet, ok := parsed["fleet"].(map[string]interface{})
	if !ok {
		t.Fatal("'fleet' is not an object")
	}

	for _, field := range []string{"total", "active", "claude", "codex", "gemini", "user", "unknown"} {
		if _, ok := fleet[field]; !ok {
			t.Errorf("JSON fleet missing '%s' field", field)
		}
	}
}

func TestNewDashboardCmdFlags(t *testing.T) {
	cmd := newDashboardCmd()

	// Verify flags exist
	noTUI := cmd.Flags().Lookup("no-tui")
	if noTUI == nil {
		t.Error("--no-tui flag not found")
	} else if noTUI.DefValue != "false" {
		t.Errorf("--no-tui default should be 'false', got %q", noTUI.DefValue)
	}

	jsonFlag := cmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Error("--json flag not found")
	} else if jsonFlag.DefValue != "false" {
		t.Errorf("--json default should be 'false', got %q", jsonFlag.DefValue)
	}
}
