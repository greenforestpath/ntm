//go:build !windows

package testutil

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func acquireGlobalTmuxTestLock(t *testing.T) {
	t.Helper()

	lockPath := filepath.Join(os.TempDir(), "ntm_tmux_tests.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("open tmux test lock: %v", err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		t.Fatalf("flock tmux test lock: %v", err)
	}

	t.Cleanup(func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	})
}
