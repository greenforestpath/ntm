//go:build !windows

package testutil

import (
	"os"
	"path/filepath"
	"syscall"
)

func withGlobalTmuxTestLock(fn func()) {
	lockPath := filepath.Join(os.TempDir(), "ntm_tmux_tests.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		fn()
		return
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		fn()
		return
	}
	defer func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}()

	fn()
}

