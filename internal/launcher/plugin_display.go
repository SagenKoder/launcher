package launcher

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/SagenKoder/launcher/internal/plugins"
)

type pluginDisplay struct {
	container      *fyne.Container
	title          *widget.Label
	messages       *fyne.Container
	scroll         *container.Scroll
	clipboard      fyne.Clipboard
	onSubmit       func(text string) (string, error)
	onSubmitStream plugins.StreamFunc
	closeOnSubmit  bool
	entries        []*messageEntry
	streamCancel   context.CancelFunc
	streamEntry    *messageEntry
	streamToken    int
	streamBuffer   strings.Builder
}

type messageEntry struct {
	container *fyne.Container
	rich      *widget.RichText
	entry     *selectableEntry
	text      string
	markdown  bool
	copyBtn   *widget.Button
}

type selectableEntry struct {
	widget.Entry
	onFocus func(bool)
}

type tapArea struct {
	widget.BaseWidget
	onTap func()
}

func newTapArea(onTap func()) *tapArea {
	area := &tapArea{onTap: onTap}
	area.ExtendBaseWidget(area)
	return area
}

func (t *tapArea) CreateRenderer() fyne.WidgetRenderer {
	rect := canvas.NewRectangle(color.Transparent)
	return widget.NewSimpleRenderer(rect)
}

func (t *tapArea) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tapArea) TappedSecondary(*fyne.PointEvent) {}

func newSelectableEntry(onFocus func(bool)) *selectableEntry {
	e := &selectableEntry{onFocus: onFocus}
	e.MultiLine = true
	e.Wrapping = fyne.TextWrapWord
	e.ExtendBaseWidget(e)
	return e
}

func (e *selectableEntry) FocusGained() {
	e.Entry.FocusGained()
	if e.onFocus != nil {
		e.onFocus(true)
	}
}

func (e *selectableEntry) FocusLost() {
	e.Entry.FocusLost()
	if e.onFocus != nil {
		e.onFocus(false)
	}
}

func (e *selectableEntry) TypedRune(r rune) {
	// prevent edits
}

func (e *selectableEntry) TypedKey(event *fyne.KeyEvent) {
	switch event.Name {
	case fyne.KeyBackspace, fyne.KeyDelete, fyne.KeyReturn, fyne.KeyEnter:
		return
	}
	e.Entry.TypedKey(event)
}

func (e *selectableEntry) TypedShortcut(shortcut fyne.Shortcut) {
	switch shortcut.(type) {
	case *fyne.ShortcutCut, *fyne.ShortcutPaste, *fyne.ShortcutUndo, *fyne.ShortcutRedo:
		return
	}
	e.Entry.TypedShortcut(shortcut)
}

func newMessageEntry(text string, markdown bool, clip fyne.Clipboard) *messageEntry {
	rich := widget.NewRichText()
	rich.Wrapping = fyne.TextWrapWord

	entry := &messageEntry{rich: rich}

	selEntry := newSelectableEntry(nil)
	selEntry.Hide()
	selEntry.Wrapping = fyne.TextWrapWord
	entry.entry = selEntry

	copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		if clip != nil {
			clip.SetContent(entry.text)
		}
	})
	copyBtn.Importance = widget.LowImportance
	copyBtn.Resize(copyBtn.MinSize())
	copyBtn.Hide()

	var tap *tapArea

	toggleFocus := func(focused bool) {
		if focused {
			rich.Hide()
			if tap != nil {
				tap.Hide()
			}
			entry.entry.Show()
			entry.entry.Refresh()
			entry.copyBtn.Show()
		} else {
			entry.entry.Hide()
			rich.Show()
			if tap != nil {
				tap.Show()
			}
			entry.copyBtn.Hide()
		}
	}

	selEntry.onFocus = toggleFocus
	tap = newTapArea(nil)

	entry.setText(text, markdown)

	content := container.NewMax(rich, selEntry, tap)
	textBox := container.NewPadded(content)
	bubble := container.NewBorder(nil, nil, nil, copyBtn, textBox)
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(0, theme.Padding()/4))
	entry.container = container.NewVBox(bubble, spacer)
	entry.copyBtn = copyBtn

	tap.onTap = func() {
		toggleFocus(true)
		if canv := fyne.CurrentApp().Driver().CanvasForObject(entry.container); canv != nil {
			canv.Focus(entry.entry)
		}
	}

	toggleFocus(false)
	return entry
}

func (m *messageEntry) setText(text string, markdown bool) {
	m.text = text
	m.markdown = markdown
	if markdown {
		m.rich.ParseMarkdown(text)
		if m.entry != nil {
			tmp := widget.NewRichText()
			tmp.ParseMarkdown(text)
			m.entry.SetText(tmp.String())
		}
	} else {
		m.rich.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: text, Style: widget.RichTextStyleParagraph}}
		m.rich.Refresh()
		if m.entry != nil {
			m.entry.SetText(text)
		}
	}
}

func newPluginDisplay(win fyne.Window) *pluginDisplay {
	title := widget.NewLabel("")
	title.Alignment = fyne.TextAlignLeading
	title.TextStyle = fyne.TextStyle{Bold: true}
	messages := container.NewVBox()
	messages.Layout = layout.NewVBoxLayout()
	scroll := container.NewVScroll(messages)
	scroll.SetMinSize(fyne.NewSize(0, 0))
	head := container.NewVBox(title)
	root := container.NewBorder(
		head,
		nil,
		nil,
		nil,
		scroll,
	)
	d := &pluginDisplay{
		container: root,
		title:     title,
		messages:  messages,
		scroll:    scroll,
	}
	if win != nil {
		d.clipboard = win.Clipboard()
	}
	return d
}

func (d *pluginDisplay) addMessage(text string, markdown bool) *messageEntry {
	entry := newMessageEntry(text, markdown, d.clipboard)
	d.entries = append(d.entries, entry)
	d.messages.Add(entry.container)
	d.messages.Refresh()
	if d.scroll != nil {
		d.scroll.ScrollToBottom()
	}
	return entry
}

func (d *pluginDisplay) Container() fyne.CanvasObject {
	return d.container
}

func (d *pluginDisplay) SetPlugin(info plugins.Info) {
	d.cancelStream()
	d.onSubmit = info.OnSubmit
	d.onSubmitStream = info.OnSubmitStream
	d.closeOnSubmit = info.CloseOnSubmit
	d.title.SetText(info.Name)
	d.entries = d.entries[:0]
	d.messages.Objects = d.messages.Objects[:0]
	d.messages.Refresh()
	if strings.TrimSpace(info.Intro) != "" {
		d.addMessage(info.Intro, true)
	}
}

func (d *pluginDisplay) HandleInput(text string, onDone func(bool, error)) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	d.addMessage(fmt.Sprintf("**You:** %s", trimmed), true)
	if d.onSubmitStream != nil {
		d.startStream(trimmed, onDone)
		return true
	}
	if d.onSubmit != nil {
		resp, err := d.onSubmit(trimmed)
		if strings.TrimSpace(resp) != "" {
			d.addMessage(resp, true)
		}
		if err != nil {
			d.addMessage(fmt.Sprintf("**Error:** %s", err.Error()), true)
		}
		if onDone != nil {
			onDone(err == nil, err)
		}
		return true
	}
	if onDone != nil {
		onDone(true, nil)
	}
	return true
}

func (d *pluginDisplay) cancelStream() {
	if d.streamCancel != nil {
		d.streamCancel()
		d.streamCancel = nil
	}
	d.streamToken++
	d.streamEntry = nil
	d.streamBuffer.Reset()
}

func (d *pluginDisplay) startStream(input string, onDone func(bool, error)) {
	d.cancelStream()
	ctx, cancel := context.WithCancel(context.Background())
	d.streamCancel = cancel
	entry := d.addMessage("", false)
	d.streamEntry = entry
	d.streamBuffer.Reset()
	d.refresh()
	token := d.streamToken

	go func(entry *messageEntry, tok int, cancel context.CancelFunc) {
		err := d.onSubmitStream(ctx, input, func(markdown string, done bool) {
			fyne.CurrentApp().Driver().DoFromGoroutine(func() {
				if d.streamToken != tok {
					return
				}
				d.streamBuffer.WriteString(markdown)
				if d.streamEntry != nil {
					d.streamEntry.setText(d.streamBuffer.String(), false)
				}
				d.refresh()
			}, false)
		})
		fyne.CurrentApp().Driver().DoFromGoroutine(func() {
			if d.streamToken == tok {
				d.streamCancel = nil
			}
			cancel()
			if err != nil && !errors.Is(err, context.Canceled) {
				d.addMessage(fmt.Sprintf("**Error:** %s", err.Error()), true)
				if onDone != nil {
					onDone(false, err)
				}
				return
			}
			if err == nil {
				if d.streamEntry != nil {
					text := d.streamBuffer.String()
					d.streamEntry.setText(text, true)
					d.refresh()
				}
			}
			if onDone != nil {
				onDone(err == nil, err)
			}
		}, false)
	}(entry, token, cancel)
}

func (d *pluginDisplay) AppendMarkdown(markdown string) {
	if strings.TrimSpace(markdown) == "" {
		return
	}
	d.addMessage(markdown, true)
}

func (d *pluginDisplay) refresh() {
	d.messages.Refresh()
	if d.scroll != nil {
		d.scroll.ScrollToBottom()
	}
}
