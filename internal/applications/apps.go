package applications

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// Application represents a desktop launcher entry available on the system.
type Application struct {
	Name     string
	Exec     string
	IconName string
	IconPath string
	Path     string
}

// List returns the applications discovered on the current system by scanning
// standard freedesktop application directories.
func List() ([]Application, error) {
	if runtime.GOOS == "darwin" {
		return listDarwin()
	}

	dirs := desktopDirs()
	seenPaths := make(map[string]struct{})
	apps := make([]Application, 0, 128)
	var errs []error

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			errs = append(errs, fmt.Errorf("read dir %s: %w", dir, err))
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".desktop") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			if _, ok := seenPaths[path]; ok {
				continue
			}
			seenPaths[path] = struct{}{}

			app, err := parseDesktopFile(path)
			if err != nil {
				if errors.Is(err, errSkipApplication) {
					continue
				}
				errs = append(errs, fmt.Errorf("parse %s: %w", path, err))
				continue
			}
			apps = append(apps, app)
		}
	}

	sort.Slice(apps, func(i, j int) bool {
		nameI := strings.ToLower(apps[i].Name)
		nameJ := strings.ToLower(apps[j].Name)
		if nameI == nameJ {
			return apps[i].Exec < apps[j].Exec
		}
		return nameI < nameJ
	})

	if len(errs) > 0 {
		return apps, errors.Join(errs...)
	}
	return apps, nil
}

var errSkipApplication = errors.New("skip application")

func parseDesktopFile(path string) (Application, error) {
	file, err := os.Open(path)
	if err != nil {
		return Application{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)

	var (
		inDesktopEntry bool
		name           string
		exec           string
		iconName       string
		appType        string
		hidden         bool
		noDisplay      bool
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inDesktopEntry = line == "[Desktop Entry]"
			continue
		}
		if !inDesktopEntry {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch {
		case key == "Type":
			appType = value
		case key == "Name":
			name = value
		case strings.HasPrefix(key, "Name[") && name == "":
			name = value
		case key == "Exec":
			exec = sanitiseExec(value)
		case key == "Icon":
			iconName = value
		case key == "Hidden":
			hidden = strings.EqualFold(value, "true")
		case key == "NoDisplay":
			noDisplay = strings.EqualFold(value, "true")
		}
	}

	if err := scanner.Err(); err != nil {
		return Application{}, err
	}

	if appType != "Application" || hidden || noDisplay || name == "" || exec == "" {
		return Application{}, errSkipApplication
	}

	iconPath := resolveIcon(iconName, path)

	return Application{
		Name:     name,
		Exec:     exec,
		IconName: iconName,
		IconPath: iconPath,
		Path:     path,
	}, nil
}

func sanitiseExec(raw string) string {
	fields := strings.Fields(raw)
	cleaned := make([]string, 0, len(fields))
	for _, field := range fields {
		if strings.Contains(field, "%") {
			continue
		}
		cleaned = append(cleaned, field)
	}
	return strings.Join(cleaned, " ")
}

func desktopDirs() []string {
	dirs := make([]string, 0, 6)

	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		dirs = append(dirs, filepath.Join(dataHome, "applications"))
	} else if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local/share/applications"))
	}

	dataDirsEnv := os.Getenv("XDG_DATA_DIRS")
	if dataDirsEnv == "" {
		dataDirsEnv = "/usr/local/share:/usr/share"
	}
	for _, dir := range strings.Split(dataDirsEnv, ":") {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		dirs = append(dirs, filepath.Join(dir, "applications"))
	}

	dirs = append(dirs, "/var/lib/snapd/desktop/applications")
	return dirs
}

var (
	iconIndex     map[string]string
	iconIndexOnce sync.Once
)

func resolveIcon(iconValue, desktopPath string) string {
	if iconValue == "" {
		return ""
	}

	if filepath.IsAbs(iconValue) {
		if fileExists(iconValue) {
			return iconValue
		}
		return ""
	}

	desktopDir := filepath.Dir(desktopPath)

	if candidate := findIconWithExtensions(filepath.Join(desktopDir, iconValue)); candidate != "" {
		return candidate
	}

	if strings.Contains(iconValue, string(filepath.Separator)) {
		if candidate := findIconWithExtensions(filepath.Clean(filepath.Join(desktopDir, iconValue))); candidate != "" {
			return candidate
		}
	}

	index := loadIconIndex()
	name := iconKey(iconValue)
	if name == "" {
		return ""
	}
	if path, ok := index[name]; ok {
		return path
	}
	return ""
}

func findIconWithExtensions(base string) string {
	if fileExists(base) {
		return base
	}
	if filepath.Ext(base) != "" {
		return ""
	}
	for _, ext := range []string{".png", ".svg", ".xpm"} {
		candidate := base + ext
		if fileExists(candidate) {
			return candidate
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func loadIconIndex() map[string]string {
	iconIndexOnce.Do(func() {
		iconIndex = make(map[string]string)
		for _, dir := range iconDirs() {
			filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if entry.IsDir() {
					return nil
				}
				ext := strings.ToLower(filepath.Ext(entry.Name()))
				if ext != ".png" && ext != ".svg" && ext != ".xpm" {
					return nil
				}
				key := iconKey(entry.Name())
				if key == "" {
					return nil
				}
				existing, ok := iconIndex[key]
				if !ok || betterIconCandidate(path, existing) {
					iconIndex[key] = path
				}
				return nil
			})
		}
	})
	return iconIndex
}

func iconKey(name string) string {
	if name == "" {
		return ""
	}
	base := name
	if ext := strings.ToLower(filepath.Ext(base)); ext == ".png" || ext == ".svg" || ext == ".xpm" {
		base = strings.TrimSuffix(base, filepath.Ext(base))
	}
	base = strings.ToLower(base)
	return base
}

func betterIconCandidate(newPath, existingPath string) bool {
	newScore := iconScore(newPath)
	oldScore := iconScore(existingPath)
	if newScore == oldScore {
		return len(newPath) < len(existingPath)
	}
	return newScore > oldScore
}

func iconScore(path string) int {
	score := 0
	if strings.Contains(path, string(filepath.Separator)+"apps"+string(filepath.Separator)) {
		score += 2
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		score += 2
	case ".svg":
		score += 1
	}
	return score
}

func iconDirs() []string {
	dirs := make([]string, 0, 8)

	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		dirs = append(dirs,
			filepath.Join(dataHome, "icons"),
			filepath.Join(dataHome, "pixmaps"),
		)
	} else if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs,
			filepath.Join(home, ".local/share/icons"),
			filepath.Join(home, ".local/share/pixmaps"),
		)
	}

	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".icons"))
	}

	dataDirsEnv := os.Getenv("XDG_DATA_DIRS")
	if dataDirsEnv == "" {
		dataDirsEnv = "/usr/local/share:/usr/share"
	}
	for _, dir := range strings.Split(dataDirsEnv, ":") {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		dirs = append(dirs,
			filepath.Join(dir, "icons"),
			filepath.Join(dir, "pixmaps"),
		)
	}

	dirs = append(dirs, "/usr/share/pixmaps")
	return dirs
}

// DebugResolveIcon is a helper for diagnostics.
func DebugResolveIcon(iconName string) string {
	return resolveIcon(iconName, "")
}

func DebugResolveIconWithDesktop(iconName, desktopPath string) string {
	return resolveIcon(iconName, desktopPath)
}
