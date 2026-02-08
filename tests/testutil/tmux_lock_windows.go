//go:build windows

package testutil

import "testing"

func acquireGlobalTmuxTestLock(t *testing.T) {
	t.Helper()
}
