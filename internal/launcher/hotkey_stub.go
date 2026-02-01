//go:build !darwin

package launcher

import (
	"sync/atomic"
)

type hotkeyStub struct{}

func setupHotkey(windowVisible *atomic.Bool, showWindow, hideWindow func()) (*hotkeyStub, *atomic.Bool) {
	var registered atomic.Bool
	return nil, &registered
}

func cleanupHotkey(hk *hotkeyStub, registered *atomic.Bool) {
	// No-op on non-darwin platforms
}
