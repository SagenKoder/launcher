package launcher

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var (
	iconCache   = make(map[string]fyne.Resource)
	iconCacheMu sync.Mutex
)

func iconResource(path string) fyne.Resource {
	if path == "" {
		return theme.FileApplicationIcon()
	}

	iconCacheMu.Lock()
	defer iconCacheMu.Unlock()

	if res, ok := iconCache[path]; ok {
		if res == nil {
			return theme.FileApplicationIcon()
		}
		return res
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("failed to read icon %s: %v", path, err)
		iconCache[path] = nil
		return theme.FileApplicationIcon()
	}

	res := fyne.NewStaticResource(filepath.Base(path), data)
	iconCache[path] = res
	return res
}
