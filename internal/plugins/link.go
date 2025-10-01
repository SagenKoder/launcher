package plugins

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/SagenKoder/launcher/internal/applications"
	"github.com/SagenKoder/launcher/internal/config"
)

func init() {
	cfg, err := config.Load()
	if err != nil {
		return
	}

	for _, link := range cfg.Links {
		if strings.TrimSpace(link.Name) == "" || strings.TrimSpace(link.URL) == "" {
			continue
		}

		linkCopy := link
		iconPath := applications.DebugResolveIcon(linkCopy.Icon)

		replacement := strings.TrimSpace(linkCopy.Replacement)
		if replacement == "" {
			Register(Info{
				ID:            "link-" + slugify(linkCopy.Name),
				Name:          linkCopy.Name,
				IconPath:      iconPath,
				Intro:         fmt.Sprintf("Opening %s…", linkCopy.Name),
				CloseOnSubmit: true,
				OnInit: func() (string, error) {
					err := openURL(linkCopy.URL)
					return fmt.Sprintf("[%s](%s)", linkCopy.Name, linkCopy.URL), err
				},
			})
			continue
		}

		replacementValue := replacement
		Register(Info{
			ID:            "link-" + slugify(linkCopy.Name),
			Name:          linkCopy.Name,
			IconPath:      iconPath,
			Intro:         fmt.Sprintf("Enter text to open %s.", linkCopy.Name),
			Hint:          fmt.Sprintf("Search %s", linkCopy.Name),
			CloseOnSubmit: true,
			OnSubmit: func(input string) (string, error) {
				trimmed := strings.TrimSpace(input)
				if trimmed == "" {
					return "", fmt.Errorf("please enter text to search %s", linkCopy.Name)
				}
				encoded := url.QueryEscape(trimmed)
				encoded = strings.ReplaceAll(encoded, "+", "%20")
				linkWithQuery := strings.ReplaceAll(linkCopy.URL, replacementValue, encoded)
				display := fmt.Sprintf("[%s](%s)", linkCopy.Name, linkWithQuery)
				if err := openURL(linkWithQuery); err != nil {
					return display, fmt.Errorf("launch browser: %w", err)
				}
				return display, nil
			},
		})
	}
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, s)
	return strings.Trim(s, "-")
}
