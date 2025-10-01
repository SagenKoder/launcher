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
	l.box.Objects = l.box.Objects[:0]
	l.items = l.items[:0]
	l.apps = append(l.apps[:0], apps...)
	for idx, app := range apps {
		item := ui.NewAppListItem()
		item.Set(iconResource(app.IconPath), app.Name)
		item.SetOnTapped(l.makeSelectHandler(idx))
		l.box.Objects = append(l.box.Objects, item)
		l.items = append(l.items, item)
	}
	if len(apps) > 0 {
		l.selected = 0
		l.updateSelection()
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
	l.selected = next
	l.updateSelection()
}

func (l *launcherList) updateSelection() {
	for idx, item := range l.items {
		item.SetSelected(idx == l.selected)
	}
}

func (l *launcherList) MoveSelection(delta int) {
	l.moveSelection(delta)
}

func (l *launcherList) makeSelectHandler(idx int) func() {
	return func() {
		l.selected = idx
		l.updateSelection()
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
