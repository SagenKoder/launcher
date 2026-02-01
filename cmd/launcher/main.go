package main

import (
	"flag"

	"github.com/SagenKoder/launcher/internal/ipc"
	"github.com/SagenKoder/launcher/internal/launcher"
)

func main() {
	hidden := flag.Bool("hidden", false, "Start with window hidden (for auto-start on login)")
	flag.Parse()

	if *hidden {
		// Starting hidden (e.g., on login) - just check if daemon exists, don't show
		if ipc.TryPing() {
			return // Daemon already running, exit silently
		}
	} else {
		// Normal start - show window if daemon exists
		if ipc.TryShow() {
			return // Daemon exists, sent show command, done
		}
	}

	// No daemon running, start as daemon
	launcher.Run(*hidden)
}
