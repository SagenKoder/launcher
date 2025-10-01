package plugins

import (
	"context"
	"os/exec"
	"runtime"
)

type Info struct {
	ID             string
	Name           string
	IconPath       string
	Intro          string
	Hint           string
	OnInit         func() (string, error)
	OnSubmit       func(input string) (string, error)
	OnChange       func(input string)
	OnSubmitStream StreamFunc
	CloseOnSubmit  bool
}

type StreamFunc func(ctx context.Context, input string, emit func(markdown string, done bool)) error

var registry []Info

func Register(info Info) {
	registry = append(registry, info)
}

func All() []Info {
	return append([]Info(nil), registry...)
}

func openURL(link string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", link)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", link)
	default:
		cmd = exec.Command("xdg-open", link)
	}
	return cmd.Start()
}
