package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigGenerateAgentCommand(t *testing.T) {
	cfg := &Config{}

	// Test no template
	cmd, err := cfg.GenerateAgentCommand("simple command", AgentTemplateVars{})
	if err != nil {
		t.Fatalf("GenerateAgentCommand failed: %v", err)
	}
	if cmd != "simple command" {
		t.Errorf("Expected 'simple command', got %q", cmd)
	}

	// Test with template
	tmpl := "echo {{.Model}}"
	vars := AgentTemplateVars{Model: "gpt-4"}
	cmd, err = cfg.GenerateAgentCommand(tmpl, vars)
	if err != nil {
		t.Fatalf("GenerateAgentCommand failed: %v", err)
	}
	if cmd != "echo gpt-4" {
		t.Errorf("Expected 'echo gpt-4', got %q", cmd)
	}
}

func TestIsPersonaName(t *testing.T) {
	cfg := &Config{}
	// Currently always returns false
	if cfg.IsPersonaName("architect") {
		t.Error("IsPersonaName should return false (not implemented)")
	}
}

func TestDetectPalettePath(t *testing.T) {
	// Test explicit path
	cfg := &Config{PaletteFile: "/custom/path.md"}
	if path := DetectPalettePath(cfg); path != "/custom/path.md" {
		t.Errorf("Expected /custom/path.md, got %s", path)
	}

	// Test nil config
	if path := DetectPalettePath(nil); path != "" {
		t.Errorf("Expected empty path for nil config, got %s", path)
	}
}

func TestScannerDefaultsGetTimeout(t *testing.T) {
	d := ScannerDefaults{Timeout: "60s"}
	if d.GetTimeout() != 60*time.Second {
		t.Errorf("Expected 60s, got %v", d.GetTimeout())
	}

	d = ScannerDefaults{Timeout: "invalid"}
	if d.GetTimeout() != 120*time.Second {
		t.Errorf("Expected default 120s for invalid, got %v", d.GetTimeout())
	}

	d = ScannerDefaults{Timeout: ""}
	if d.GetTimeout() != 120*time.Second {
		t.Errorf("Expected default 120s for empty, got %v", d.GetTimeout())
	}
}

func TestScannerToolsIsToolEnabled(t *testing.T) {
	// Default (empty) -> all enabled
	tools := ScannerTools{}
	if !tools.IsToolEnabled("semgrep") {
		t.Error("Empty config should enable all tools")
	}

	// Enabled list
	tools = ScannerTools{Enabled: []string{"semgrep"}}
	if !tools.IsToolEnabled("semgrep") {
		t.Error("Explicitly enabled tool should be enabled")
	}
	if tools.IsToolEnabled("gosec") {
		t.Error("Tool not in enabled list should be disabled")
	}

	// Disabled list
	tools = ScannerTools{Disabled: []string{"bandit"}}
	if tools.IsToolEnabled("bandit") {
		t.Error("Disabled tool should be disabled")
	}
	if !tools.IsToolEnabled("semgrep") {
		t.Error("Non-disabled tool should be enabled")
	}
}

func TestThresholdConfigShouldBlock(t *testing.T) {
	t.Run("block critical", func(t *testing.T) {
		tc := ThresholdConfig{BlockCritical: true}
		if !tc.ShouldBlock(1, 0) {
			t.Error("Should block on critical")
		}
		if tc.ShouldBlock(0, 5) {
			t.Error("Should not block on errors when BlockErrors=0")
		}
	})

	t.Run("block errors", func(t *testing.T) {
		tc := ThresholdConfig{BlockErrors: 5}
		if !tc.ShouldBlock(0, 5) {
			t.Error("Should block on 5 errors")
		}
		if tc.ShouldBlock(0, 4) {
			t.Error("Should not block on 4 errors")
		}
	})
}

func TestThresholdConfigShouldFail(t *testing.T) {
	t.Run("fail critical", func(t *testing.T) {
		tc := ThresholdConfig{FailCritical: true}
		if !tc.ShouldFail(1, 0) {
			t.Error("Should fail on critical")
		}
	})

	t.Run("fail errors", func(t *testing.T) {
		tc := ThresholdConfig{FailErrors: 0} // Any error fails
		if !tc.ShouldFail(0, 1) {
			t.Error("Should fail on 1 error")
		}

		tc = ThresholdConfig{FailErrors: -1} // Disabled
		if tc.ShouldFail(0, 100) {
			t.Error("Should not fail when disabled")
		}
	})
}

func TestLoadProjectScannerConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Test no config
	cfg, err := LoadProjectScannerConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadProjectScannerConfig failed: %v", err)
	}
	// Should return defaults
	if cfg.Defaults.Timeout != "120s" {
		t.Errorf("Expected default timeout 120s, got %s", cfg.Defaults.Timeout)
	}

	// Test .ntm.yaml
	yamlContent := `
scanner:
  defaults:
    timeout: 30s
`
	os.WriteFile(filepath.Join(tmpDir, ".ntm.yaml"), []byte(yamlContent), 0644)

	cfg, err = LoadProjectScannerConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadProjectScannerConfig failed: %v", err)
	}
	if cfg.Defaults.Timeout != "30s" {
		t.Errorf("Expected timeout 30s from yaml, got %s", cfg.Defaults.Timeout)
	}
}
