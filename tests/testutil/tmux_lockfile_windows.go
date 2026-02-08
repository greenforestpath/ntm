//go:build windows

package testutil

func withGlobalTmuxTestLock(fn func()) {
	fn()
}

