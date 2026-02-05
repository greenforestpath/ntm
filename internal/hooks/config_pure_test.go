package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// DefaultCommandHooksPath
// ---------------------------------------------------------------------------

func TestDefaultCommandHooksPath_XDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")

	got := DefaultCommandHooksPath()
	want := filepath.Join("/custom/config", "ntm", "hooks.toml")
	if got != want {
		t.Errorf("DefaultCommandHooksPath() = %q, want %q", got, want)
	}
}

func TestDefaultCommandHooksPath_NoXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	got := DefaultCommandHooksPath()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "ntm", "hooks.toml")
	if got != want {
		t.Errorf("DefaultCommandHooksPath() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// DefaultCommandHooksDir
// ---------------------------------------------------------------------------

func TestDefaultCommandHooksDir_XDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")

	got := DefaultCommandHooksDir()
	want := filepath.Join("/custom/config", "ntm", "hooks")
	if got != want {
		t.Errorf("DefaultCommandHooksDir() = %q, want %q", got, want)
	}
}

func TestDefaultCommandHooksDir_NoXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	got := DefaultCommandHooksDir()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "ntm", "hooks")
	if got != want {
		t.Errorf("DefaultCommandHooksDir() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// ExpandWorkDir — tilde expansion path
// ---------------------------------------------------------------------------

func TestExpandWorkDir_TildePrefix(t *testing.T) {
	h := CommandHook{WorkDir: "~/projects/myapp"}
	got := h.ExpandWorkDir("mysession", "/default/project")

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, "projects/myapp")
	if got != want {
		t.Errorf("ExpandWorkDir() = %q, want %q", got, want)
	}
}

func TestExpandWorkDir_EnvVarExpansion(t *testing.T) {
	t.Setenv("MY_CUSTOM_DIR", "/opt/workspace")

	h := CommandHook{WorkDir: "$MY_CUSTOM_DIR/sub"}
	got := h.ExpandWorkDir("sess", "/default")

	if !strings.Contains(got, "/opt/workspace") {
		t.Errorf("ExpandWorkDir() = %q, want to contain /opt/workspace", got)
	}
}

func TestExpandWorkDir_CombinedVariables(t *testing.T) {
	h := CommandHook{WorkDir: "${SESSION}/${PROJECT}"}
	got := h.ExpandWorkDir("myses", "/home/proj")

	if !strings.Contains(got, "myses") {
		t.Errorf("ExpandWorkDir() = %q, want to contain session name", got)
	}
	if !strings.Contains(got, "/home/proj") {
		t.Errorf("ExpandWorkDir() = %q, want to contain project dir", got)
	}
}

// ---------------------------------------------------------------------------
// CommandHooksConfig.Validate — multi-hook validation
// ---------------------------------------------------------------------------

func TestCommandHooksConfigValidate_AllValid(t *testing.T) {
	t.Parallel()

	cfg := &CommandHooksConfig{
		Hooks: []CommandHook{
			{Event: EventPreSpawn, Command: "echo 1"},
			{Event: EventPostSend, Command: "echo 2"},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestCommandHooksConfigValidate_OneInvalid(t *testing.T) {
	t.Parallel()

	cfg := &CommandHooksConfig{
		Hooks: []CommandHook{
			{Event: EventPreSpawn, Command: "echo 1"},
			{Event: EventPreSpawn, Command: ""}, // invalid
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() expected error for empty command")
	}
	if !strings.Contains(err.Error(), "command_hooks[1]") {
		t.Errorf("error should reference index 1: %v", err)
	}
}

// ---------------------------------------------------------------------------
// MarshalText round-trip
// ---------------------------------------------------------------------------

func TestDurationMarshalTextRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dur  Duration
	}{
		{"30 seconds", Duration(30e9)},
		{"5 minutes", Duration(300e9)},
		{"1 hour", Duration(3600e9)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			text, err := tc.dur.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText() error: %v", err)
			}

			var decoded Duration
			if err := decoded.UnmarshalText(text); err != nil {
				t.Fatalf("UnmarshalText(%q) error: %v", string(text), err)
			}

			if decoded != tc.dur {
				t.Errorf("round-trip: got %v, want %v", decoded, tc.dur)
			}
		})
	}
}
