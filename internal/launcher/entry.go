package launcher

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type launcherEntry struct {
	widget.Entry
	onEscape        func()
	onMoveSelection func(delta int)
	onActivate      func()
}

func newLauncherEntry(onEscape func()) *launcherEntry {
	entry := &launcherEntry{onEscape: onEscape}
	entry.ExtendBaseWidget(entry)
	return entry
}

func (e *launcherEntry) TypedKey(event *fyne.KeyEvent) {
	switch event.Name {
	case fyne.KeyEscape:
		if e.onEscape != nil {
			e.onEscape()
		}
	case fyne.KeyDown:
		if e.onMoveSelection != nil {
			e.onMoveSelection(1)
			return
		}
		e.Entry.TypedKey(event)
	case fyne.KeyUp:
		if e.onMoveSelection != nil {
			e.onMoveSelection(-1)
			return
		}
		e.Entry.TypedKey(event)
	case fyne.KeyReturn, fyne.KeyEnter:
		if e.onActivate != nil {
			e.onActivate()
			return
		}
		e.Entry.TypedKey(event)
	default:
		e.Entry.TypedKey(event)
	}
}

func (e *launcherEntry) SetOnMoveSelection(fn func(int)) {
	e.onMoveSelection = fn
}

func (e *launcherEntry) SetOnActivate(fn func()) {
	e.onActivate = fn
}
