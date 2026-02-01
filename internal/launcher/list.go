package launcher

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/SagenKoder/launcher/internal/applications"
	"github.com/SagenKoder/launcher/internal/ui"
)

type launcherList struct {
	widget.BaseWidget
	box        *fyne.Container
	scroll     *container.Scroll
	items      []*ui.AppListItem
	pool       []*ui.AppListItem // Pre-allocated widget pool for reuse
	apps       []applications.Application
	selected   int
	onEscape   func()
	onActivate func(app applications.Application)
}

func newLauncherList(onEscape func()) *launcherList {
	l := &launcherList{onEscape: onEscape, selected: -1}
	l.ExtendBaseWidget(l)
	return l
}

func (l *launcherList) CreateRenderer() fyne.WidgetRenderer {
	l.box = container.NewVBox()
	l.scroll = container.NewVScroll(l.box)
	return widget.NewSimpleRenderer(l.scroll)
}

func (l *launcherList) SetApplications(apps []applications.Application) {
	if l.box == nil {
		return
	}

	needed := len(apps)
	current := len(l.items)

	// Return excess items to pool
	if current > needed {
		l.pool = append(l.pool, l.items[needed:]...)
		l.items = l.items[:needed]
	}

	// Update apps slice
	l.apps = append(l.apps[:0], apps...)

	// Reuse existing items and update their content
	for idx := 0; idx < needed; idx++ {
		var item *ui.AppListItem

		if idx < current {
			// Reuse existing widget
			item = l.items[idx]
		} else if len(l.pool) > 0 {
			// Pull from pool
			item = l.pool[len(l.pool)-1]
			l.pool = l.pool[:len(l.pool)-1]
			l.items = append(l.items, item)
		} else {
			// Create new widget only when necessary
			item = ui.NewAppListItem()
			l.items = append(l.items, item)
		}

		app := apps[idx]
		item.Set(iconResourceAsync(app.IconPath, item), app.Name)
		item.SetOnTapped(l.makeSelectHandler(idx))
		item.SetSelected(idx == 0 && needed > 0)
	}

	// Rebuild box objects slice (no allocations, just pointer assignments)
	l.box.Objects = l.box.Objects[:0]
	for _, item := range l.items {
		l.box.Objects = append(l.box.Objects, item)
	}

	if needed > 0 {
		l.selected = 0
	} else {
		l.selected = -1
	}
	l.box.Refresh()
}

func (l *launcherList) ScrollToTop() {
	if l.scroll != nil {
		l.scroll.ScrollToTop()
	}
}

func (l *launcherList) moveSelection(delta int) {
	if len(l.apps) == 0 {
		return
	}
	next := l.selected
	if next < 0 {
		next = 0
	} else {
		next += delta
		if next < 0 {
			next = 0
		}
		if next >= len(l.apps) {
			next = len(l.apps) - 1
		}
	}
	if next == l.selected {
		return
	}
	l.setSelection(next)
}

func (l *launcherList) setSelection(newIndex int) {
	// Only update the changed items for O(1) instead of O(n)
	if l.selected >= 0 && l.selected < len(l.items) {
		l.items[l.selected].SetSelected(false)
	}
	l.selected = newIndex
	if l.selected >= 0 && l.selected < len(l.items) {
		l.items[l.selected].SetSelected(true)
	}
}

func (l *launcherList) updateSelection() {
	// Full refresh - only used when list contents change
	for idx, item := range l.items {
		item.SetSelected(idx == l.selected)
	}
}

func (l *launcherList) MoveSelection(delta int) {
	l.moveSelection(delta)
}

func (l *launcherList) makeSelectHandler(idx int) func() {
	return func() {
		l.setSelection(idx)
	}
}

func (l *launcherList) SetOnActivate(fn func(app applications.Application)) {
	l.onActivate = fn
}

func (l *launcherList) ActivateSelection() {
	if l.onActivate != nil && l.selected >= 0 && l.selected < len(l.apps) {
		l.onActivate(l.apps[l.selected])
	}
}

func (l *launcherList) SelectedApplication() (applications.Application, bool) {
	if l.selected >= 0 && l.selected < len(l.apps) {
		return l.apps[l.selected], true
	}
	return applications.Application{}, false
}

func (l *launcherList) TypedKey(event *fyne.KeyEvent) {
	switch event.Name {
	case fyne.KeyEscape:
		if l.onEscape != nil {
			l.onEscape()
		}
	case fyne.KeyDown:
		l.moveSelection(1)
	case fyne.KeyUp:
		l.moveSelection(-1)
	case fyne.KeyReturn, fyne.KeyEnter:
		l.ActivateSelection()
	}
}
