package launcher

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	fynedesktop "fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"golang.design/x/hotkey"

	"github.com/SagenKoder/launcher/internal/applications"
	"github.com/SagenKoder/launcher/internal/ipc"
	"github.com/SagenKoder/launcher/internal/plugins"
	"github.com/SagenKoder/launcher/internal/search"
)

func Run(startHidden bool) {
	application := app.New()
	window := application.NewWindow("Launcher")

	window.Resize(fyne.NewSize(600, 400))
	window.CenterOnScreen()
	window.SetFixedSize(false)

	// Load cached apps immediately for instant startup
	apps, hasCached := applications.LoadCached()
	if !hasCached {
		apps = []applications.Application{}
	}

	// Channel for receiving refreshed apps
	appsChan := make(chan []applications.Application, 1)

	// Background refresh
	go func() {
		fresh, err := applications.List()
		if err != nil {
			log.Printf("failed to load applications: %v", err)
		}
		if len(fresh) > 0 {
			applications.SaveCache(fresh)
			appsChan <- fresh
		}
	}()

	preloadIcons(apps)

	apps = append(apps, pluginApplications()...)
	apps = append(apps, settingsApplication())
	sort.Slice(apps, func(i, j int) bool {
		nameI := strings.ToLower(apps[i].Name)
		nameJ := strings.ToLower(apps[j].Name)
		if nameI == nameJ {
			return apps[i].Exec < apps[j].Exec
		}
		return nameI < nameJ
	})

	filtered := make([]applications.Application, 0)

	// Track window visibility for toggle
	var windowVisible atomic.Bool
	windowVisible.Store(!startHidden) // Window starts visible unless --hidden flag

	// hideWindow hides the window instead of closing (for daemon mode)
	hideWindow := func() {
		window.Hide()
		windowVisible.Store(false)
	}

	list := newLauncherList(hideWindow)
	pluginDisplay := newPluginDisplay(window)
	badge := newPluginBadge()
	body := container.NewMax(list)
	var activePlugin *plugins.Info
	var settingsActive bool
	var settings *settingsPanel

	defaultPlaceholder := "Type to search applications"

	var entry *launcherEntry
	var topBar *fyne.Container

	clearEntry := func() {
		if entry != nil {
			entry.SetText("")
		}
	}

	// resetToHome resets the launcher to its initial state
	resetToHome := func() {
		activePlugin = nil
		settingsActive = false
		body.Objects = []fyne.CanvasObject{list}
		body.Refresh()
		badge.Hide()
		if entry != nil {
			entry.SetPlaceHolder(defaultPlaceholder)
			clearEntry()
		}
		if topBar != nil {
			topBar.Refresh()
		}
		// Reset list to show limited apps for performance
		displayApps := apps
		if len(displayApps) > 50 {
			displayApps = displayApps[:50]
		}
		list.SetApplications(displayApps)
	}

	registry := buildPluginRegistry()

	showPlugin := func(id string) {
		info, ok := registry[id]
		if !ok {
			log.Printf("unknown plugin id %q", id)
			return
		}
		infoCopy := info
		activePlugin = &infoCopy
		pluginDisplay.SetPlugin(infoCopy)
		body.Objects = []fyne.CanvasObject{pluginDisplay.Container()}
		body.Refresh()
		badge.Show()
		badge.Set(iconResource(infoCopy.IconPath), infoCopy.Name)
		if entry != nil {
			if infoCopy.Hint != "" {
				entry.SetPlaceHolder(infoCopy.Hint)
			} else {
				entry.SetPlaceHolder(defaultPlaceholder)
			}
			clearEntry()
			window.Canvas().Focus(entry)
		}
		if topBar != nil {
			topBar.Refresh()
		}
		if infoCopy.OnInit != nil {
			markdown, err := infoCopy.OnInit()
			if strings.TrimSpace(markdown) != "" {
				pluginDisplay.AppendMarkdown(markdown)
			}
			if err != nil {
				pluginDisplay.AppendMarkdown(fmt.Sprintf("**Error:** %s", err.Error()))
			} else if infoCopy.CloseOnSubmit {
				hideWindow()
				return
			}
		}
	}

	showSettings := func() {
		if settings == nil {
			settings = newSettingsPanel(func() {
				// Callback after save - settings saved
			})
		}
		settings.Refresh()
		settingsActive = true
		activePlugin = nil
		body.Objects = []fyne.CanvasObject{settings.Container()}
		body.Refresh()
		badge.Show()
		badge.Set(theme.SettingsIcon(), "Settings")
		if entry != nil {
			entry.SetPlaceHolder("Settings")
			clearEntry()
		}
		if topBar != nil {
			topBar.Refresh()
		}
	}

	entry = newLauncherEntry(hideWindow)
	entry.SetPlaceHolder(defaultPlaceholder)
	entry.SetOnMoveSelection(func(delta int) {
		list.MoveSelection(delta)
	})
	runSelected := func() {
		if settingsActive {
			// Settings is active, don't process input
			return
		}
		if activePlugin != nil {
			text := entry.Text
			if strings.TrimSpace(text) != "" {
				pluginCopy := *activePlugin
				processed := pluginDisplay.HandleInput(text, func(success bool, err error) {
					if success && pluginCopy.CloseOnSubmit && err == nil {
						hideWindow()
					}
				})
				clearEntry()
				if processed {
					return
				}
			}
			clearEntry()
			return
		}
		if app, ok := list.SelectedApplication(); ok {
			launchApplication(window, app, showPlugin, showSettings, hideWindow)
		}
	}
	entry.SetOnActivate(runSelected)
	list.SetOnActivate(func(app applications.Application) {
		launchApplication(window, app, showPlugin, showSettings, hideWindow)
	})

	// Debounced search to prevent UI lag while typing
	var debounceTimer *time.Timer
	var debounceMu sync.Mutex
	const debounceDelay = 16 * time.Millisecond // ~1 frame at 60fps
	const maxDisplayResults = 50               // Limit displayed results for performance

	limitResults := func(results []applications.Application) []applications.Application {
		if len(results) > maxDisplayResults {
			return results[:maxDisplayResults]
		}
		return results
	}

	updateFilter := func(text string) {
		if activePlugin != nil {
			if activePlugin.OnChange != nil {
				activePlugin.OnChange(text)
			}
			return
		}

		// Show all apps when query cleared
		if text == "" {
			list.SetApplications(limitResults(apps))
			list.ScrollToTop()
			return
		}

		// Debounce all keystrokes uniformly
		debounceMu.Lock()
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		searchText := text
		debounceTimer = time.AfterFunc(debounceDelay, func() {
			fyne.Do(func() {
				filtered = search.Filter(apps, searchText)
				list.SetApplications(limitResults(filtered))
				if len(filtered) > 0 {
					list.ScrollToTop()
				}
			})
		})
		debounceMu.Unlock()
	}
	entry.OnChanged = updateFilter
	entry.OnSubmitted = func(string) {
		// For now we just clear the entry to make it obvious input was received.
		clearEntry()
	}

	topBar = container.NewBorder(nil, nil, badge.Object(), nil, entry)
	content := container.NewBorder(topBar, nil, nil, nil, body)
	window.SetContent(container.NewPadded(content))

	// Hide window instead of close on Escape
	window.Canvas().AddShortcut(&fynedesktop.CustomShortcut{KeyName: fyne.KeyEscape}, func(fyne.Shortcut) {
		hideWindow()
	})
	window.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyEscape {
			hideWindow()
		}
	})

	// Intercept window close to hide instead
	window.SetCloseIntercept(func() {
		hideWindow()
	})

	// showWindow shows the window and prepares it for input
	showWindow := func() {
		resetToHome()
		window.Show()
		window.CenterOnScreen()
		window.RequestFocus()
		window.Canvas().Focus(entry)
		windowVisible.Store(true)
	}

	// Register global hotkey: Option+Space (deferred until after event loop starts)
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModOption}, hotkey.KeySpace)
	var hotkeyRegistered atomic.Bool
	go func() {
		// Wait for the Fyne event loop to be ready
		time.Sleep(500 * time.Millisecond)
		if err := hk.Register(); err != nil {
			log.Printf("failed to register hotkey (Option+Space): %v", err)
			return
		}
		hotkeyRegistered.Store(true)
		log.Printf("registered global hotkey: Option+Space")
		for range hk.Keydown() {
			// Toggle window visibility
			if windowVisible.Load() {
				hideWindow()
			} else {
				showWindow()
			}
		}
	}()

	// Start IPC server for daemon communication
	ipcServer := ipc.NewServer(
		showWindow,
		hideWindow,
		func() {
			ipc.Cleanup()
			application.Quit()
		},
	)
	if err := ipcServer.Start(); err != nil {
		log.Printf("failed to start IPC server: %v", err)
	}

	// Ensure cleanup on exit
	defer func() {
		if hotkeyRegistered.Load() {
			hk.Unregister()
		}
		ipcServer.Close()
		ipc.Cleanup()
	}()

	// Initialize list with limited apps for performance
	list.SetApplications(limitResults(apps))

	// Listen for background app refresh
	go func() {
		select {
		case fresh := <-appsChan:
			// Add plugins and settings
			fresh = append(fresh, pluginApplications()...)
			fresh = append(fresh, settingsApplication())
			sort.Slice(fresh, func(i, j int) bool {
				nameI := strings.ToLower(fresh[i].Name)
				nameJ := strings.ToLower(fresh[j].Name)
				if nameI == nameJ {
					return fresh[i].Exec < fresh[j].Exec
				}
				return nameI < nameJ
			})
			// Update the shared apps slice
			apps = fresh
			// Refresh the list if entry is empty (showing all apps)
			if entry.Text == "" {
				fyne.Do(func() {
					list.SetApplications(limitResults(apps))
				})
			}
			// Preload icons for new apps
			preloadIcons(fresh)
			// Invalidate search cache since apps changed
			search.InvalidateCache()
		}
	}()

	window.Canvas().Focus(entry)

	if startHidden {
		// Start hidden - show briefly then hide (required for Fyne initialization)
		go func() {
			time.Sleep(100 * time.Millisecond)
			fyne.Do(hideWindow)
		}()
		// Pre-warm caches when starting hidden (daemon mode)
		go prewarmCaches(apps)
	}
	window.ShowAndRun()
}

func buildPluginRegistry() map[string]plugins.Info {
	registry := make(map[string]plugins.Info, len(plugins.All()))
	for _, info := range plugins.All() {
		registry[info.ID] = info
	}
	return registry
}

func pluginApplications() []applications.Application {
	all := plugins.All()
	apps := make([]applications.Application, 0, len(all))
	for _, info := range all {
		apps = append(apps, applications.Application{
			Name:     info.Name,
			Exec:     fmt.Sprintf("plugin:%s", info.ID),
			Path:     fmt.Sprintf("plugin:%s", info.ID),
			IconPath: info.IconPath,
		})
	}
	return apps
}

func settingsApplication() applications.Application {
	return applications.Application{
		Name:     "Settings",
		Exec:     "settings:",
		Path:     "settings:",
		IconPath: "theme:settings",
	}
}

// prewarmCaches pre-warms search and icon caches for faster first interaction
func prewarmCaches(apps []applications.Application) {
	// Pre-warm common search prefixes (single letters)
	for c := 'a'; c <= 'z'; c++ {
		search.Filter(apps, string(c))
	}
	// Pre-warm number prefixes
	for c := '0'; c <= '9'; c++ {
		search.Filter(apps, string(c))
	}
}

func launchApplication(window fyne.Window, app applications.Application, showPlugin func(string), showSettings func(), hideWindow func()) {
	execCmd := strings.TrimSpace(app.Exec)
	if strings.HasPrefix(execCmd, "settings:") {
		if showSettings != nil {
			showSettings()
		}
		return
	}
	if strings.HasPrefix(execCmd, "plugin:") {
		if showPlugin != nil {
			showPlugin(strings.TrimPrefix(execCmd, "plugin:"))
		}
		return
	}
	if execCmd == "" {
		log.Printf("no executable defined for %s", app.Name)
		return
	}
	if runtime.GOOS == "darwin" {
		bundlePath := strings.TrimSpace(app.Path)
		if strings.HasSuffix(strings.ToLower(bundlePath), ".app") {
			if _, err := os.Stat(bundlePath); err == nil {
				cmd := exec.Command("open", bundlePath)
				if err := cmd.Start(); err != nil {
					log.Printf("failed to launch %s: %v", app.Name, err)
					return
				}
				hideWindow()
				return
			}
		}
	}
	cmd := exec.Command("sh", "-c", execCmd)
	if err := cmd.Start(); err != nil {
		log.Printf("failed to launch %s: %v", app.Name, err)
		return
	}
	hideWindow()
}
