package config

import (
	"os"
	"path/filepath"
	"sort"
)

var sourceMountRoots = []string{
	"/run/media",
	"/media",
	"/Volumes",
}

func DetectSourceDefault() string {
	if source, ok := detectSourceDefault(sourceMountRoots, os.Getenv("USER")); ok {
		return source
	}
	return DefaultSource
}

func detectSourceDefault(roots []string, userName string) (string, bool) {
	var patterns []string
	for _, root := range roots {
		if userName != "" {
			patterns = append(patterns, filepath.Join(root, userName, "*", "DCIM"))
		}
		patterns = append(patterns, filepath.Join(root, "*", "DCIM"))
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		sort.Strings(matches)
		for _, match := range matches {
			if isDir(match) {
				return ensureTrailingSeparator(match), true
			}
		}
	}

	return "", false
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func ensureTrailingSeparator(path string) string {
	if path == "" || os.IsPathSeparator(path[len(path)-1]) {
		return path
	}
	return path + string(os.PathSeparator)
}
