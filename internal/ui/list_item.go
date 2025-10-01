package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type AppListItem struct {
	widget.BaseWidget
	icon     *widget.Icon
	label    *widget.Label
	bg       *canvas.Rectangle
	selected bool
	onTapped func()
}

func NewAppListItem() *AppListItem {
	item := &AppListItem{
		icon:  widget.NewIcon(theme.FileApplicationIcon()),
		label: widget.NewLabel(""),
	}
	item.label.Alignment = fyne.TextAlignLeading
	item.label.Truncation = fyne.TextTruncateEllipsis
	item.ExtendBaseWidget(item)
	return item
}

func (i *AppListItem) CreateRenderer() fyne.WidgetRenderer {
	i.bg = canvas.NewRectangle(color.Transparent)
	content := container.NewBorder(nil, nil, i.icon, nil, i.label)
	stack := container.NewMax(i.bg, content)
	return widget.NewSimpleRenderer(stack)
}

func (i *AppListItem) Set(icon fyne.Resource, text string) {
	if icon != nil {
		i.icon.SetResource(icon)
	} else {
		i.icon.SetResource(theme.FileApplicationIcon())
	}
	i.label.SetText(text)
	i.Refresh()
}

func (i *AppListItem) SetSelected(selected bool) {
	if i.selected == selected {
		return
	}
	i.selected = selected
	i.Refresh()
}

func (i *AppListItem) SetOnTapped(fn func()) {
	i.onTapped = fn
}

func (i *AppListItem) Tapped(*fyne.PointEvent) {
	if i.onTapped != nil {
		i.onTapped()
	}
}

func (i *AppListItem) TappedSecondary(*fyne.PointEvent) {}

func (i *AppListItem) Refresh() {
	if i.bg != nil {
		if i.selected {
			i.bg.FillColor = theme.SelectionColor()
			i.label.TextStyle = fyne.TextStyle{Bold: true}
		} else {
			i.bg.FillColor = color.Transparent
			i.label.TextStyle = fyne.TextStyle{}
		}
		i.bg.Refresh()
	}
	if i.icon != nil {
		i.icon.Refresh()
	}
	i.label.Refresh()
}
