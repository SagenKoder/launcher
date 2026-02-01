package main

import (
	"github.com/SagenKoder/launcher/internal/ipc"
	"github.com/SagenKoder/launcher/internal/launcher"
)

func main() {
	// Try to connect to existing daemon and show window
	if ipc.TryShow() {
		return // Daemon exists, sent show command, done
	}

	// No daemon running, start as daemon
	launcher.Run()
}
