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
	"github.com/SagenKoder/launcher/internal/ui"
)

var (
	iconCache    = make(map[string]fyne.Resource)
	iconCacheMu  sync.RWMutex
	iconCacheDir string
	iconDirOnce  sync.Once
)

type iconRequest struct {
	path   string
	widget *ui.AppListItem
}

var (
	iconQueue     chan iconRequest
	iconQueueOnce sync.Once
)

const numIconWorkers = 4

func initIconWorkers() {
	iconQueueOnce.Do(func() {
		iconQueue = make(chan iconRequest, 100)
		for i := 0; i < numIconWorkers; i++ {
			go iconWorker()
		}
	})
}

func iconWorker() {
	for req := range iconQueue {
		res := loadIconSync(req.path)
		if res != nil && req.widget != nil {
			widget := req.widget
			icon := res
			fyne.Do(func() {
				widget.SetIcon(icon)
			})
		}
	}
}

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

// iconResource returns the icon for the given path.
// For cached icons, returns immediately. For uncached icons, returns placeholder
// and loads asynchronously, updating the widget when ready.
func iconResource(path string) fyne.Resource {
	return iconResourceAsync(path, nil)
}

// iconResourceAsync returns the icon for the given path, optionally updating
// the widget asynchronously if the icon needs to be loaded.
func iconResourceAsync(path string, widget *ui.AppListItem) fyne.Resource {
	initIconWorkers()

	if path == "" {
		return theme.FileApplicationIcon()
	}
	// Handle special icon identifiers
	if strings.HasPrefix(path, "theme:") {
		return themeIcon(strings.TrimPrefix(path, "theme:"))
	}

	// Check cache first (read lock for performance)
	iconCacheMu.RLock()
	if res, ok := iconCache[path]; ok {
		iconCacheMu.RUnlock()
		if res == nil {
			return theme.FileApplicationIcon()
		}
		return res
	}
	iconCacheMu.RUnlock()

	// For ICNS files on macOS, check if PNG already exists in disk cache
	if runtime.GOOS == "darwin" && strings.EqualFold(filepath.Ext(path), ".icns") {
		if pngPath := convertIcnsToPngCached(path); pngPath != "" {
			// PNG exists, load it synchronously (fast)
			return loadIconSync(path)
		}
		// No PNG cache, queue for async conversion
		if widget != nil {
			select {
			case iconQueue <- iconRequest{path: path, widget: widget}:
			default:
				// Queue full, fall through to sync load
			}
			return theme.FileApplicationIcon()
		}
	}

	// For non-ICNS or when no widget provided, load synchronously
	return loadIconSync(path)
}

func loadIconSync(path string) fyne.Resource {
	iconCacheMu.Lock()
	defer iconCacheMu.Unlock()

	// Double-check cache after acquiring write lock
	if res, ok := iconCache[path]; ok {
		if res == nil {
			return theme.FileApplicationIcon()
		}
		return res
	}

	loadPath := path
	if runtime.GOOS == "darwin" && strings.EqualFold(filepath.Ext(path), ".icns") {
		if converted := convertIcnsToPng(path); converted != "" {
			loadPath = converted
		} else {
			iconCache[path] = nil
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
