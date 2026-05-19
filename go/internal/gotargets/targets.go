package gotargets

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/dutifuldev/slophammer/go/internal/repo"
)

var (
	ErrNoTargets = errors.New("go targets are required")
	ErrNoFiles   = errors.New("go targets resolved to zero production files")
)

type Options struct {
	Targets []string
	Exclude []string
}

func Resolve(snapshot repo.Snapshot, options Options) ([]string, error) {
	targets := cleanList(options.Targets)
	if len(targets) == 0 {
		return nil, ErrNoTargets
	}

	excludes := cleanList(options.Exclude)
	files := map[string]struct{}{}
	for _, target := range targets {
		resolveTarget(snapshot, target, excludes, files)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoFiles, strings.Join(targets, ", "))
	}

	resolved := make([]string, 0, len(files))
	for filePath := range files {
		resolved = append(resolved, filePath)
	}
	sort.Strings(resolved)
	return resolved, nil
}

func resolveTarget(snapshot repo.Snapshot, target string, excludes []string, files map[string]struct{}) {
	if isGoFileTarget(target) {
		if _, ok := snapshot.Files[target]; ok && isProductionGoFileForTarget(target, target, excludes) {
			files[target] = struct{}{}
		}
		return
	}

	for filePath := range snapshot.Files {
		if isUnderTarget(filePath, target) && isProductionGoFileForTarget(filePath, target, excludes) {
			files[filePath] = struct{}{}
		}
	}
}

func isGoFileTarget(target string) bool {
	return strings.HasSuffix(target, ".go")
}

func isUnderTarget(filePath string, target string) bool {
	if target == "." {
		return true
	}
	return filePath == target || strings.HasPrefix(filePath, target+"/")
}

func isProductionGoFileForTarget(filePath string, target string, excludes []string) bool {
	if !strings.HasSuffix(filePath, ".go") || strings.HasSuffix(filePath, "_test.go") {
		return false
	}
	if hasDefaultExcludedSegment(filePath) {
		return false
	}
	return !matchesConfiguredExclude(filePath, target, excludes)
}

func hasDefaultExcludedSegment(filePath string) bool {
	for _, segment := range strings.Split(filePath, "/") {
		switch segment {
		case "", ".", "testdata", "fixtures", "vendor", "target", "node_modules":
			return true
		}
		if strings.HasPrefix(segment, ".") {
			return true
		}
	}
	return false
}

func matchesConfiguredExclude(filePath string, target string, excludes []string) bool {
	for _, exclude := range excludes {
		if matchesPattern(filePath, exclude) {
			return true
		}
		if targetRelativePath, ok := relativeToTarget(filePath, target); ok && matchesPattern(targetRelativePath, exclude) {
			return true
		}
	}
	return false
}

func relativeToTarget(filePath string, target string) (string, bool) {
	if target == "." || isGoFileTarget(target) {
		return filePath, target == "."
	}
	if filePath == target {
		return path.Base(filePath), true
	}
	if strings.HasPrefix(filePath, target+"/") {
		return strings.TrimPrefix(filePath, target+"/"), true
	}
	return "", false
}

func matchesPattern(filePath string, pattern string) bool {
	if matched, _ := doublestar.Match(pattern, filePath); matched {
		return true
	}
	if !strings.Contains(pattern, "/") {
		matched, _ := doublestar.Match(pattern, path.Base(filePath))
		return matched
	}
	return false
}

func cleanList(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		cleaned = append(cleaned, cleanPath(value))
	}
	return cleaned
}

func cleanPath(value string) string {
	cleaned := path.Clean(strings.ReplaceAll(value, "\\", "/"))
	if cleaned == "/" {
		return "."
	}
	return strings.TrimPrefix(cleaned, "./")
}
