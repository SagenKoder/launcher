package launcher

import (
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	fynedesktop "fyne.io/fyne/v2/driver/desktop"

	"github.com/SagenKoder/launcher/internal/applications"
	"github.com/SagenKoder/launcher/internal/plugins"
	"github.com/SagenKoder/launcher/internal/search"
)

func Run() {
	application := app.New()
	window := application.NewWindow("Launcher")

	window.Resize(fyne.NewSize(600, 400))
	window.CenterOnScreen()
	window.SetFixedSize(false)

	apps, err := applications.List()
	if err != nil {
		log.Printf("failed to load applications: %v", err)
	}

	apps = append(apps, pluginApplications()...)
	sort.Slice(apps, func(i, j int) bool {
		nameI := strings.ToLower(apps[i].Name)
		nameJ := strings.ToLower(apps[j].Name)
		if nameI == nameJ {
			return apps[i].Exec < apps[j].Exec
		}
		return nameI < nameJ
	})

	filtered := make([]applications.Application, 0)
	list := newLauncherList(window.Close)
	pluginDisplay := newPluginDisplay(window)
	badge := newPluginBadge()
	body := container.NewMax(list)
	var activePlugin *plugins.Info

	defaultPlaceholder := "Type to search applications"

	var entry *launcherEntry
	var topBar *fyne.Container

	clearEntry := func() {
		if entry != nil {
			entry.SetText("")
		}
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
				window.Close()
				return
			}
		}
	}

	entry = newLauncherEntry(func() {
		window.Close()
	})
	entry.SetPlaceHolder(defaultPlaceholder)
	entry.SetOnMoveSelection(func(delta int) {
		list.MoveSelection(delta)
	})
	runSelected := func() {
		if activePlugin != nil {
			text := entry.Text
			if strings.TrimSpace(text) != "" {
				pluginCopy := *activePlugin
				processed := pluginDisplay.HandleInput(text, func(success bool, err error) {
					if success && pluginCopy.CloseOnSubmit && err == nil {
						window.Close()
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
			launchApplication(window, app, showPlugin)
		}
	}
	entry.SetOnActivate(runSelected)
	list.SetOnActivate(func(app applications.Application) {
		launchApplication(window, app, showPlugin)
	})

	updateFilter := func(text string) {
		if activePlugin != nil {
			if activePlugin.OnChange != nil {
				activePlugin.OnChange(text)
			}
			return
		}
		filtered = search.Filter(apps, text)
		list.SetApplications(filtered)
		if len(filtered) > 0 {
			list.ScrollToTop()
		}
	}
	entry.OnChanged = updateFilter
	entry.OnSubmitted = func(string) {
		// For now we just clear the entry to make it obvious input was received.
		clearEntry()
	}

	topBar = container.NewBorder(nil, nil, badge.Object(), nil, entry)
	content := container.NewBorder(topBar, nil, nil, nil, body)
	window.SetContent(container.NewPadded(content))

	window.Canvas().AddShortcut(&fynedesktop.CustomShortcut{KeyName: fyne.KeyEscape}, func(fyne.Shortcut) {
		window.Close()
	})
	window.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyEscape {
			window.Close()
		}
	})

	window.Canvas().Focus(entry)
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

func launchApplication(window fyne.Window, app applications.Application, showPlugin func(string)) {
	execCmd := strings.TrimSpace(app.Exec)
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
	cmd := exec.Command("sh", "-c", execCmd)
	if err := cmd.Start(); err != nil {
		log.Printf("failed to launch %s: %v", app.Name, err)
		return
	}
	window.Close()
}
