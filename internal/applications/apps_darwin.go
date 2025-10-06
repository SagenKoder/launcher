//go:build darwin

package applications

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type macInfoPlist struct {
	DisplayName string
	BundleName  string
	Executable  string
	IconFile    string
	IconFiles   []string
}

func listDarwin() ([]Application, error) {
	roots := []string{
		"/Applications",
		"/System/Applications",
	}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(home, "Applications"))
	}

	seen := make(map[string]struct{})
	apps := make([]Application, 0, 128)
	var errs []error

	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if errors.Is(walkErr, os.ErrNotExist) {
					return fs.SkipDir
				}
				errs = append(errs, fmt.Errorf("walk %s: %w", path, walkErr))
				return fs.SkipDir
			}
			if !d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".app") {
				return nil
			}
			bundlePath := path
			if _, ok := seen[bundlePath]; ok {
				return fs.SkipDir
			}
			seen[bundlePath] = struct{}{}

			app, err := parseAppBundle(bundlePath)
			if err != nil {
				errs = append(errs, fmt.Errorf("parse bundle %s: %w", bundlePath, err))
			}
			if strings.TrimSpace(app.Name) != "" {
				apps = append(apps, app)
			}
			return fs.SkipDir
		})
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Errorf("walk root %s: %w", root, err))
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

func parseAppBundle(bundlePath string) (Application, error) {
	info, err := readInfoPlist(bundlePath)
	name := firstNonEmpty(
		strings.TrimSpace(info.DisplayName),
		strings.TrimSpace(info.BundleName),
		strings.TrimSuffix(filepath.Base(bundlePath), ".app"),
	)

	iconPath := resolveMacIcon(bundlePath, info)
	iconName := ""
	if iconPath != "" {
		iconName = filepath.Base(iconPath)
	}

	return Application{
		Name:     name,
		Exec:     bundlePath,
		IconName: iconName,
		IconPath: iconPath,
		Path:     bundlePath,
	}, err
}

func resolveMacIcon(bundlePath string, info macInfoPlist) string {
	resources := filepath.Join(bundlePath, "Contents", "Resources")

	candidates := make([]string, 0, len(info.IconFiles)+1)
	if strings.TrimSpace(info.IconFile) != "" {
		candidates = append(candidates, strings.TrimSpace(info.IconFile))
	}
	for _, name := range info.IconFiles {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			candidates = append(candidates, trimmed)
		}
	}

	for _, name := range candidates {
		if filepath.Ext(name) == "" {
			if candidate := filepath.Join(resources, name+".icns"); fileExists(candidate) {
				return candidate
			}
		}
		candidate := filepath.Join(resources, name)
		if fileExists(candidate) {
			return candidate
		}
	}

	entries, err := os.ReadDir(resources)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".icns") {
			candidate := filepath.Join(resources, entry.Name())
			if fileExists(candidate) {
				return candidate
			}
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func readInfoPlist(bundlePath string) (macInfoPlist, error) {
	infoPath := filepath.Join(bundlePath, "Contents", "Info.plist")
	if !fileExists(infoPath) {
		return macInfoPlist{}, fmt.Errorf("info.plist not found: %s", infoPath)
	}

	cmd := exec.Command("plutil", "-convert", "json", "-o", "-", infoPath)
	data, err := cmd.Output()
	if err != nil {
		return macInfoPlist{}, fmt.Errorf("plutil convert %s: %w", infoPath, err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return macInfoPlist{}, fmt.Errorf("parse json %s: %w", infoPath, err)
	}

	info := macInfoPlist{
		DisplayName: stringFromMap(raw, "CFBundleDisplayName"),
		BundleName:  stringFromMap(raw, "CFBundleName"),
		Executable:  stringFromMap(raw, "CFBundleExecutable"),
		IconFile:    stringFromMap(raw, "CFBundleIconFile"),
		IconFiles:   stringSliceFromMap(raw, "CFBundleIconFiles"),
	}
	return info, nil
}

func stringFromMap(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func stringSliceFromMap(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			continue
		}
		if trimmed := strings.TrimSpace(s); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
