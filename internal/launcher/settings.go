package launcher

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/SagenKoder/launcher/internal/config"
)

type settingsPanel struct {
	container    *fyne.Container
	tabs         *container.AppTabs
	linksTab     *fyne.Container
	chatTab      *fyne.Container
	linksList    *fyne.Container
	linksScroll  *container.Scroll
	cfg          config.Config
	onSave       func()
	editingIndex int // -1 = adding new, >= 0 = editing existing
	editForm     *fyne.Container
	editName     *widget.Entry
	editURL      *widget.Entry
	editIcon     *widget.Entry
	editReplace  *widget.Entry
}

func newSettingsPanel(onSave func()) *settingsPanel {
	s := &settingsPanel{
		onSave:       onSave,
		editingIndex: -2, // -2 = not editing
	}
	s.cfg, _ = config.Load()
	s.buildUI()
	return s
}

func (s *settingsPanel) buildUI() {
	// Build Links tab
	s.linksList = container.NewVBox()
	s.linksScroll = container.NewVScroll(s.linksList)
	s.linksScroll.SetMinSize(fyne.NewSize(0, 200))

	addBtn := widget.NewButtonWithIcon("Add Link", theme.ContentAddIcon(), func() {
		s.showLinkEditor(-1, config.LinkConfig{})
	})
	addBtn.Importance = widget.HighImportance

	s.editForm = container.NewVBox()
	s.editForm.Hide()

	linksContent := container.NewBorder(
		nil,
		container.NewVBox(s.editForm, addBtn),
		nil,
		nil,
		s.linksScroll,
	)
	s.linksTab = linksContent

	// Build Chat tab
	s.chatTab = s.buildChatTab()

	// Create tabbed interface
	s.tabs = container.NewAppTabs(
		container.NewTabItemWithIcon("Links", theme.ListIcon(), s.linksTab),
		container.NewTabItemWithIcon("Chat", theme.MailComposeIcon(), s.chatTab),
	)
	s.tabs.SetTabLocation(container.TabLocationTop)

	title := widget.NewLabelWithStyle("Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	s.container = container.NewBorder(
		container.NewVBox(title, widget.NewSeparator()),
		nil,
		nil,
		nil,
		s.tabs,
	)

	s.refreshLinksList()
}

func (s *settingsPanel) buildChatTab() *fyne.Container {
	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetPlaceHolder("sk-...")
	apiKeyEntry.SetText(s.cfg.Chat.APIKey)

	baseURLEntry := widget.NewEntry()
	baseURLEntry.SetPlaceHolder("https://api.openai.com/v1")
	baseURLEntry.SetText(s.cfg.Chat.BaseURL)

	modelEntry := widget.NewEntry()
	modelEntry.SetPlaceHolder("gpt-4")
	modelEntry.SetText(s.cfg.Chat.Model)

	saveBtn := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		s.cfg.Chat.APIKey = apiKeyEntry.Text
		s.cfg.Chat.BaseURL = baseURLEntry.Text
		s.cfg.Chat.Model = modelEntry.Text
		if err := config.Save(s.cfg); err == nil && s.onSave != nil {
			s.onSave()
		}
	})
	saveBtn.Importance = widget.HighImportance

	form := container.NewVBox(
		widget.NewLabel("API Key"),
		apiKeyEntry,
		widget.NewLabel("Base URL"),
		baseURLEntry,
		widget.NewLabel("Model"),
		modelEntry,
		widget.NewSeparator(),
		saveBtn,
	)

	return container.NewVBox(form)
}

func (s *settingsPanel) refreshLinksList() {
	s.linksList.Objects = nil
	for idx, link := range s.cfg.Links {
		item := s.createLinkItem(idx, link)
		s.linksList.Add(item)
	}
	s.linksList.Refresh()
}

func (s *settingsPanel) createLinkItem(idx int, link config.LinkConfig) fyne.CanvasObject {
	icon := widget.NewIcon(theme.ComputerIcon())
	if link.Icon != "" {
		if res := themeIcon(link.Icon); res != nil {
			icon.SetResource(res)
		}
	}

	name := widget.NewLabel(link.Name)
	name.Truncation = fyne.TextTruncateEllipsis

	editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
		s.showLinkEditor(idx, link)
	})
	editBtn.Importance = widget.LowImportance

	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		s.deleteLink(idx)
	})
	deleteBtn.Importance = widget.LowImportance

	buttons := container.NewHBox(editBtn, deleteBtn)
	row := container.NewBorder(nil, nil, icon, buttons, name)
	return row
}

func (s *settingsPanel) showLinkEditor(idx int, link config.LinkConfig) {
	s.editingIndex = idx

	s.editName = widget.NewEntry()
	s.editName.SetPlaceHolder("Link name")
	s.editName.SetText(link.Name)

	s.editURL = widget.NewEntry()
	s.editURL.SetPlaceHolder("https://example.com/search?q=__QUERY__")
	s.editURL.SetText(link.URL)

	s.editIcon = widget.NewEntry()
	s.editIcon.SetPlaceHolder("System icon name (optional)")
	s.editIcon.SetText(link.Icon)

	s.editReplace = widget.NewEntry()
	s.editReplace.SetPlaceHolder("Query placeholder (e.g., __QUERY__)")
	s.editReplace.SetText(link.Replacement)

	title := "Add Link"
	if idx >= 0 {
		title = "Edit Link"
	}

	saveBtn := widget.NewButtonWithIcon("Save", theme.ConfirmIcon(), func() {
		s.saveLinkEdit()
	})
	saveBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		s.hideLinkEditor()
	})

	s.editForm.Objects = []fyne.CanvasObject{
		widget.NewSeparator(),
		widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Name"),
		s.editName,
		widget.NewLabel("URL"),
		s.editURL,
		widget.NewLabel("Icon"),
		s.editIcon,
		widget.NewLabel("Query Placeholder"),
		s.editReplace,
		container.NewHBox(saveBtn, cancelBtn),
	}
	s.editForm.Show()
	s.editForm.Refresh()
}

func (s *settingsPanel) hideLinkEditor() {
	s.editingIndex = -2
	s.editForm.Hide()
	s.editForm.Refresh()
}

func (s *settingsPanel) saveLinkEdit() {
	link := config.LinkConfig{
		Name:        s.editName.Text,
		URL:         s.editURL.Text,
		Icon:        s.editIcon.Text,
		Replacement: s.editReplace.Text,
	}

	if s.editingIndex >= 0 && s.editingIndex < len(s.cfg.Links) {
		s.cfg.Links[s.editingIndex] = link
	} else {
		s.cfg.Links = append(s.cfg.Links, link)
	}

	if err := config.Save(s.cfg); err == nil {
		s.hideLinkEditor()
		s.refreshLinksList()
		if s.onSave != nil {
			s.onSave()
		}
	}
}

func (s *settingsPanel) deleteLink(idx int) {
	if idx < 0 || idx >= len(s.cfg.Links) {
		return
	}
	s.cfg.Links = append(s.cfg.Links[:idx], s.cfg.Links[idx+1:]...)
	if err := config.Save(s.cfg); err == nil {
		s.refreshLinksList()
		if s.onSave != nil {
			s.onSave()
		}
	}
}

func (s *settingsPanel) Container() fyne.CanvasObject {
	return s.container
}

func (s *settingsPanel) Refresh() {
	s.cfg, _ = config.Load()
	s.refreshLinksList()
}
