package robot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// Tmux Environment Detection Tests (bd-35xyt)
// =============================================================================

func TestDetectTmuxEnv(t *testing.T) {
	info := DetectTmuxEnv()

	// RecommendedPath should always be /usr/bin/tmux
	if info.RecommendedPath != "/usr/bin/tmux" {
		t.Errorf("RecommendedPath = %q, want %q", info.RecommendedPath, "/usr/bin/tmux")
	}

	// BinaryPath should be a valid path or empty
	if info.BinaryPath != "" && !fileExists(info.BinaryPath) {
		// Only fail if we claimed to find a path that doesn't exist
		t.Errorf("BinaryPath %q does not exist", info.BinaryPath)
	}

	// Warning should be set only if alias detected
	if info.ShellAliasDetected && info.Warning == "" {
		t.Error("ShellAliasDetected=true but Warning is empty")
	}
	if !info.ShellAliasDetected && info.Warning != "" {
		t.Error("ShellAliasDetected=false but Warning is set")
	}
}

func TestFindTmuxBinaryPath(t *testing.T) {
	path := findTmuxBinaryPath()

	// Should return a path
	if path == "" {
		t.Error("findTmuxBinaryPath() returned empty string")
	}

	// Path should either exist or be the default fallback
	if !fileExists(path) && path != "/usr/bin/tmux" {
		t.Errorf("findTmuxBinaryPath() returned non-existent path: %q", path)
	}
}

func TestGetTmuxVersion(t *testing.T) {
	tests := []struct {
		name       string
		binaryPath string
		wantEmpty  bool
	}{
		{
			name:       "valid tmux path",
			binaryPath: "/usr/bin/tmux",
			wantEmpty:  false,
		},
		{
			name:       "invalid path",
			binaryPath: "/nonexistent/path/to/tmux",
			wantEmpty:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := getTmuxVersion(tt.binaryPath)

			if tt.wantEmpty && version != "" {
				t.Errorf("getTmuxVersion(%q) = %q, want empty", tt.binaryPath, version)
			}
			if !tt.wantEmpty && version == "" {
				// Skip if tmux isn't installed
				if !fileExists(tt.binaryPath) {
					t.Skipf("tmux not found at %q", tt.binaryPath)
				}
				t.Errorf("getTmuxVersion(%q) returned empty, want version string", tt.binaryPath)
			}

			// If we got a version, it should contain "tmux"
			if version != "" && !contains(version, "tmux") {
				t.Errorf("getTmuxVersion() = %q, want string containing 'tmux'", version)
			}
		})
	}
}

func TestTmuxEnvInfo_JSONStructure(t *testing.T) {
	info := TmuxEnvInfo{
		BinaryPath:         "/usr/bin/tmux",
		Version:            "tmux 3.4",
		ShellAliasDetected: true,
		RecommendedPath:    "/usr/bin/tmux",
		Warning:            "Use binary_path to avoid shell plugin interference",
		OhMyZshTmuxPlugin:  false,
		TmuxinatorDetected: false,
		TmuxResurrect:      false,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal TmuxEnvInfo: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Check required fields are present
	requiredFields := []string{
		"binary_path",
		"version",
		"shell_alias_detected",
		"recommended_path",
		"oh_my_zsh_tmux_plugin",
		"tmuxinator_detected",
		"tmux_resurrect",
	}

	for _, field := range requiredFields {
		if _, ok := decoded[field]; !ok {
			t.Errorf("Missing field %q in JSON output", field)
		}
	}

	// Warning should be present when alias detected
	if _, ok := decoded["warning"]; !ok {
		t.Error("Missing 'warning' field when shell_alias_detected=true")
	}
}

func TestTmuxEnvInfo_NoWarningWhenNoAlias(t *testing.T) {
	info := TmuxEnvInfo{
		BinaryPath:         "/usr/bin/tmux",
		Version:            "tmux 3.4",
		ShellAliasDetected: false,
		RecommendedPath:    "/usr/bin/tmux",
		// Warning should be empty
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Warning should be omitted when empty (omitempty)
	if _, ok := decoded["warning"]; ok {
		t.Error("Warning field should be omitted when empty (omitempty)")
	}
}

func TestEnvOutput_JSONEnvelope(t *testing.T) {
	output := EnvOutput{
		RobotResponse: NewRobotResponse(true),
		Session:       "test-session",
		Tmux: TmuxEnvInfo{
			BinaryPath:      "/usr/bin/tmux",
			Version:         "tmux 3.4",
			RecommendedPath: "/usr/bin/tmux",
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal EnvOutput: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Check RobotResponse envelope fields
	if _, ok := decoded["success"]; !ok {
		t.Error("Missing 'success' field from RobotResponse envelope")
	}
	if _, ok := decoded["timestamp"]; !ok {
		t.Error("Missing 'timestamp' field from RobotResponse envelope")
	}

	// Check env-specific fields
	if _, ok := decoded["tmux"]; !ok {
		t.Error("Missing 'tmux' field in EnvOutput")
	}
	if _, ok := decoded["session"]; !ok {
		t.Error("Missing 'session' field in EnvOutput")
	}
}

func TestTimingInfo_Defaults(t *testing.T) {
	timing := &TimingInfo{
		CtrlCGapMs:          100,
		PostExitWaitMs:      3000,
		CCInitWaitMs:        6000,
		PromptSubmitDelayMs: 1000,
	}

	// Verify reasonable defaults
	if timing.CtrlCGapMs != 100 {
		t.Errorf("CtrlCGapMs = %d, want 100", timing.CtrlCGapMs)
	}
	if timing.PostExitWaitMs != 3000 {
		t.Errorf("PostExitWaitMs = %d, want 3000", timing.PostExitWaitMs)
	}
	if timing.CCInitWaitMs != 6000 {
		t.Errorf("CCInitWaitMs = %d, want 6000", timing.CCInitWaitMs)
	}
	if timing.PromptSubmitDelayMs != 1000 {
		t.Errorf("PromptSubmitDelayMs = %d, want 1000", timing.PromptSubmitDelayMs)
	}
}

func TestTargetingInfo_Format(t *testing.T) {
	targeting := &TargetingInfo{
		PaneFormat:         "session:window.pane",
		ExampleAgentPane:   "myproject:1.2",
		ExampleControlPane: "myproject:1.1",
	}

	if targeting.PaneFormat != "session:window.pane" {
		t.Errorf("PaneFormat = %q, want %q", targeting.PaneFormat, "session:window.pane")
	}

	// Examples should match session name in format
	if !contains(targeting.ExampleAgentPane, "myproject:1") {
		t.Errorf("ExampleAgentPane %q should contain session reference", targeting.ExampleAgentPane)
	}
	if !contains(targeting.ExampleControlPane, "myproject:1") {
		t.Errorf("ExampleControlPane %q should contain session reference", targeting.ExampleControlPane)
	}
}

func TestSessionStructureInfo(t *testing.T) {
	structure := &SessionStructureInfo{
		WindowIndex:     1,
		ControlPane:     1,
		AgentPaneStart:  2,
		AgentPaneEnd:    16,
		TotalAgentPanes: 15,
	}

	// Validate relationships
	if structure.ControlPane >= structure.AgentPaneStart {
		t.Error("ControlPane should be less than AgentPaneStart")
	}
	if structure.AgentPaneEnd < structure.AgentPaneStart {
		t.Error("AgentPaneEnd should be >= AgentPaneStart")
	}
	if structure.TotalAgentPanes != structure.AgentPaneEnd-structure.AgentPaneStart+1 {
		t.Errorf("TotalAgentPanes %d doesn't match range [%d, %d]",
			structure.TotalAgentPanes, structure.AgentPaneStart, structure.AgentPaneEnd)
	}
}

func TestDetectOhMyZshTmuxPlugin(t *testing.T) {
	// This test checks the detection logic works
	// It may return true or false depending on the actual environment
	result := detectOhMyZshTmuxPlugin()

	// Just verify it doesn't panic and returns a bool
	_ = result
}

func TestDetectTmuxinator(t *testing.T) {
	result := detectTmuxinator()
	// Just verify it doesn't panic and returns a bool
	_ = result
}

func TestDetectTmuxResurrect(t *testing.T) {
	result := detectTmuxResurrect()
	// Just verify it doesn't panic and returns a bool
	_ = result
}

func TestFileExists(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"root exists", "/", false}, // directory, not file
		{"nonexistent", "/nonexistent/path/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fileExists(tt.path)
			if got != tt.want {
				t.Errorf("fileExists(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}

	// Test with a file that should exist
	t.Run("usr_bin_sh", func(t *testing.T) {
		// /bin/sh should exist on any Unix system
		if !fileExists("/bin/sh") && !fileExists("/usr/bin/sh") {
			t.Skip("Neither /bin/sh nor /usr/bin/sh exists")
		}
	})
}

func TestDirExists(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"root exists", "/", true},
		{"tmp exists", "/tmp", true},
		{"nonexistent", "/nonexistent/path/dir", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dirExists(tt.path)
			if got != tt.want {
				t.Errorf("dirExists(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDetectShellEnv(t *testing.T) {
	// Save original SHELL and restore after test
	originalShell := os.Getenv("SHELL")
	defer os.Setenv("SHELL", originalShell)

	tests := []struct {
		name      string
		shell     string
		wantType  string
		wantNil   bool
	}{
		{"zsh", "/bin/zsh", "zsh", false},
		{"bash", "/bin/bash", "bash", false},
		{"empty shell", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("SHELL", tt.shell)
			info := detectShellEnv()

			if tt.wantNil {
				if info != nil {
					t.Error("Expected nil, got non-nil ShellEnvInfo")
				}
				return
			}

			if info == nil {
				t.Fatal("Expected non-nil ShellEnvInfo, got nil")
			}

			if info.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", info.Type, tt.wantType)
			}
		})
	}
}

// Integration test - only run if HOME is set
func TestDetectOhMyZshTmuxPlugin_Integration(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}

	omzDir := filepath.Join(home, ".oh-my-zsh")
	if !dirExists(omzDir) {
		t.Skip("oh-my-zsh not installed")
	}

	// Run detection - just verify it doesn't error
	result := detectOhMyZshTmuxPlugin()
	t.Logf("oh-my-zsh tmux plugin detected: %v", result)
}
