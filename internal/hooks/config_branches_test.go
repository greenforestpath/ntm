package hooks

import "testing"

// ---------------------------------------------------------------------------
// LoadCommandHooksFromTOML — cover TOML parse error branch (83.3% → 100%)
// ---------------------------------------------------------------------------

func TestLoadCommandHooksFromTOML_MalformedTOML(t *testing.T) {
	t.Parallel()

	// This is malformed TOML (missing closing bracket, invalid syntax)
	malformedTOML := `[[command_hooks
event = "pre-spawn"
command = "echo hello"`

	_, err := LoadCommandHooksFromTOML(malformedTOML)
	if err == nil {
		t.Error("LoadCommandHooksFromTOML() should return error for malformed TOML")
	}
}

func TestLoadCommandHooksFromTOML_InvalidSyntax(t *testing.T) {
	t.Parallel()

	// Invalid TOML syntax (missing quotes around value)
	invalidSyntax := `[[command_hooks]]
event = pre-spawn
command = "echo"`

	_, err := LoadCommandHooksFromTOML(invalidSyntax)
	if err == nil {
		t.Error("LoadCommandHooksFromTOML() should return error for invalid syntax")
	}
}
