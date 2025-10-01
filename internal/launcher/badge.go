package launcher

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type pluginBadge struct {
	container  *fyne.Container
	background *canvas.Rectangle
	icon       *widget.Icon
	label      *widget.Label
}

func newPluginBadge() *pluginBadge {
	background := canvas.NewRectangle(theme.InputBackgroundColor())
	background.CornerRadius = 16
	icon := widget.NewIcon(theme.SearchIcon())
	label := widget.NewLabel("")
	label.TextStyle = fyne.TextStyle{Bold: true}
	label.Alignment = fyne.TextAlignLeading
	label.Truncation = fyne.TextTruncateOff
	inner := container.NewHBox(icon, widget.NewLabel(" "), label)
	padded := container.NewPadded(inner)
	root := container.NewMax(background, padded)
	root.Hide()
	return &pluginBadge{
		container:  root,
		background: background,
		icon:       icon,
		label:      label,
	}
}

func (b *pluginBadge) Set(resource fyne.Resource, text string) {
	if resource != nil {
		b.icon.SetResource(resource)
	} else {
		b.icon.SetResource(theme.SearchIcon())
	}
	b.icon.Refresh()
	b.label.SetText(fmt.Sprintf("[%s]", text))
	b.label.Refresh()
	b.background.FillColor = theme.InputBackgroundColor()
	b.background.Refresh()
	if b.container.Visible() {
		b.container.Refresh()
		b.container.Resize(b.container.MinSize())
	}
}

func (b *pluginBadge) Show() {
	b.container.Show()
	b.container.Refresh()
}

func (b *pluginBadge) Hide() {
	b.container.Hide()
}

func (b *pluginBadge) Object() fyne.CanvasObject {
	return b.container
}
