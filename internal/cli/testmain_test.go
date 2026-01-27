package cli

import (
	"os"
	"testing"

	"github.com/Dicklesworthstone/ntm/tests/testutil"
)

func TestMain(m *testing.M) {
	// Clean up any orphan test sessions from previous runs before starting.
	// This catches sessions left behind when tests are interrupted (Ctrl+C, timeout, etc.)
	testutil.KillAllTestSessionsSilent()

	code := m.Run()

	// Clean up after all tests complete
	testutil.KillAllTestSessionsSilent()

	os.Exit(code)
}
