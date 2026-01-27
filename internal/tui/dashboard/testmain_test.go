package dashboard

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Many dashboard rendering helpers assume colors are enabled.
	// The repo's default environment sets NO_COLOR=1, so force colors on
	// for this package's tests to keep expectations stable.
	_ = os.Setenv("NTM_NO_COLOR", "0")
	_ = os.Setenv("NTM_THEME", "mocha")

	os.Exit(m.Run())
}
