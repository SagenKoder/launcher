//go:build darwin

package launcher

import (
	"log"
	"sync/atomic"
	"time"

	"golang.design/x/hotkey"
)

func setupHotkey(windowVisible *atomic.Bool, showWindow, hideWindow func()) (*hotkey.Hotkey, *atomic.Bool) {
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModOption}, hotkey.KeySpace)
	var hotkeyRegistered atomic.Bool

	go func() {
		time.Sleep(500 * time.Millisecond)
		if err := hk.Register(); err != nil {
			log.Printf("failed to register hotkey (Option+Space): %v", err)
			return
		}
		hotkeyRegistered.Store(true)
		log.Printf("registered global hotkey: Option+Space")
		for range hk.Keydown() {
			if windowVisible.Load() {
				hideWindow()
			} else {
				showWindow()
			}
		}
	}()

	return hk, &hotkeyRegistered
}

func cleanupHotkey(hk *hotkey.Hotkey, registered *atomic.Bool) {
	if registered.Load() {
		hk.Unregister()
	}
}
