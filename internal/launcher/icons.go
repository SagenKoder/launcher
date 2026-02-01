package launcher

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/SagenKoder/launcher/internal/applications"
)

var (
	iconCache    = make(map[string]fyne.Resource)
	iconCacheMu  sync.Mutex
	iconCacheDir string
	iconDirOnce  sync.Once
)

func getIconCacheDir() string {
	iconDirOnce.Do(func() {
		cacheBase := ""
		if runtime.GOOS == "darwin" {
			if home, err := os.UserHomeDir(); err == nil {
				cacheBase = filepath.Join(home, "Library", "Caches", "Launcher", "icons")
			}
		} else if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
			cacheBase = filepath.Join(xdgCache, "launcher", "icons")
		} else if home, err := os.UserHomeDir(); err == nil {
			cacheBase = filepath.Join(home, ".cache", "launcher", "icons")
		}
		if cacheBase != "" {
			if err := os.MkdirAll(cacheBase, 0755); err == nil {
				iconCacheDir = cacheBase
			}
		}
	})
	return iconCacheDir
}

func preloadIcons(apps []applications.Application) {
	if runtime.GOOS != "darwin" {
		return
	}
	go func() {
		for _, app := range apps {
			if app.IconPath == "" {
				continue
			}
			if !strings.EqualFold(filepath.Ext(app.IconPath), ".icns") {
				continue
			}
			convertIcnsToPng(app.IconPath)
		}
	}()
}

func iconResource(path string) fyne.Resource {
	if path == "" {
		return theme.FileApplicationIcon()
	}
	// Handle special icon identifiers
	if strings.HasPrefix(path, "theme:") {
		return themeIcon(strings.TrimPrefix(path, "theme:"))
	}

	iconCacheMu.Lock()
	defer iconCacheMu.Unlock()

	if res, ok := iconCache[path]; ok {
		if res == nil {
			return theme.FileApplicationIcon()
		}
		return res
	}

	loadPath := path
	if runtime.GOOS == "darwin" && strings.EqualFold(filepath.Ext(path), ".icns") {
		if converted := convertIcnsToPngCached(path); converted != "" {
			loadPath = converted
		} else {
			return theme.FileApplicationIcon()
		}
	}

	data, err := os.ReadFile(loadPath)
	if err != nil {
		log.Printf("failed to read icon %s: %v", loadPath, err)
		iconCache[path] = nil
		return theme.FileApplicationIcon()
	}

	res := fyne.NewStaticResource(filepath.Base(loadPath), data)
	iconCache[path] = res
	return res
}

func convertIcnsToPngCached(icnsPath string) string {
	cacheDir := getIconCacheDir()
	if cacheDir == "" {
		return ""
	}

	hash := sha256.Sum256([]byte(icnsPath))
	pngName := hex.EncodeToString(hash[:8]) + ".png"
	pngPath := filepath.Join(cacheDir, pngName)

	if info, err := os.Stat(pngPath); err == nil && !info.IsDir() {
		return pngPath
	}
	return ""
}

func convertIcnsToPng(icnsPath string) string {
	cacheDir := getIconCacheDir()
	if cacheDir == "" {
		return ""
	}

	hash := sha256.Sum256([]byte(icnsPath))
	pngName := hex.EncodeToString(hash[:8]) + ".png"
	pngPath := filepath.Join(cacheDir, pngName)

	if info, err := os.Stat(pngPath); err == nil && !info.IsDir() {
		return pngPath
	}

	cmd := exec.Command("sips", "-s", "format", "png", icnsPath, "--out", pngPath)
	if err := cmd.Run(); err != nil {
		log.Printf("failed to convert icns %s: %v", icnsPath, err)
		return ""
	}

	return pngPath
}

func themeIcon(name string) fyne.Resource {
	switch name {
	case "settings":
		return theme.SettingsIcon()
	case "search":
		return theme.SearchIcon()
	case "home":
		return theme.HomeIcon()
	case "file":
		return theme.FileIcon()
	case "folder":
		return theme.FolderIcon()
	case "document":
		return theme.DocumentIcon()
	case "mail":
		return theme.MailComposeIcon()
	case "info":
		return theme.InfoIcon()
	case "help":
		return theme.HelpIcon()
	case "computer":
		return theme.ComputerIcon()
	case "storage":
		return theme.StorageIcon()
	case "download":
		return theme.DownloadIcon()
	case "upload":
		return theme.UploadIcon()
	case "account":
		return theme.AccountIcon()
	case "history":
		return theme.HistoryIcon()
	case "list":
		return theme.ListIcon()
	case "grid":
		return theme.GridIcon()
	case "visibility":
		return theme.VisibilityIcon()
	default:
		return theme.FileApplicationIcon()
	}
}
