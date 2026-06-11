package rules

import (
	"context"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/dutifuldev/slophammer/go/internal/config"
	"github.com/dutifuldev/slophammer/go/internal/repo"
)

// conventionalGoPathSegments is the path-level form of the conventional
// non-production list in specs/CONFIG.md.
var conventionalGoPathSegments = map[string]struct{}{
	"tests":        {},
	"fixtures":     {},
	"templates":    {},
	"testdata":     {},
	"dist":         {},
	"build":        {},
	"coverage":     {},
	"target":       {},
	"node_modules": {},
	"vendor":       {},
	"scripts":      {},
	"benches":      {},
}

// goScopeRule enforces that configured scope accounts for every production
// Go file: each one is either inside a configured scope or covered by a
// conventional or reasoned exclude, so narrowing scope cannot hide code.
type goScopeRule struct {
	definition Definition
}

func newGoScopeRule(definition Definition) Rule {
	return goScopeRule{definition: definition}
}

func (r goScopeRule) Metadata() Metadata {
	return r.definition.Metadata()
}

func (r goScopeRule) Check(context.Context, repo.Snapshot) []Finding {
	return nil
}

func (r goScopeRule) CheckWithConfig(_ context.Context, snapshot repo.Snapshot, cfg config.Config) []Finding {
	if !cfg.GoScopeConfigured() {
		return nil
	}
	uncovered := uncoveredProductionGoDirs(snapshot, cfg)
	if len(uncovered) == 0 {
		return nil
	}
	return []Finding{{
		RuleID:   r.definition.ID,
		Severity: r.definition.Severity,
		Path:     r.definition.Path,
		Message:  r.definition.Message + ": " + strings.Join(uncovered, ", "),
	}}
}

// GoScopeCoverage counts production files the configured scope covers versus
// all production files. Nil when no scope is configured.
func GoScopeCoverage(snapshot repo.Snapshot, cfg config.Config) *ScopeCoverage {
	if !cfg.GoScopeConfigured() {
		return nil
	}
	scopes, _ := cfg.GoScopeUnion()
	production := productionGoFiles(snapshot)
	scanned := 0
	for _, filePath := range production {
		if inScopeTargets(filePath, scopes) {
			scanned++
		}
	}
	return &ScopeCoverage{Scanned: scanned, ProductionFiles: len(production)}
}

func uncoveredProductionGoDirs(snapshot repo.Snapshot, cfg config.Config) []string {
	scopes, excludes := cfg.GoScopeUnion()
	dirs := map[string]struct{}{}
	for _, filePath := range productionGoFiles(snapshot) {
		if inScopeTargets(filePath, scopes) || matchesAnyGlob(filePath, excludes) {
			continue
		}
		dirs[parentDir(filePath)] = struct{}{}
	}
	return sortedRootSet(dirs)
}

func productionGoFiles(snapshot repo.Snapshot) []string {
	files := make([]string, 0)
	for filePath := range snapshot.Files {
		if strings.HasSuffix(filePath, ".go") && !conventionalGoPath(filePath) {
			files = append(files, filePath)
		}
	}
	sort.Strings(files)
	return files
}

func conventionalGoPath(filePath string) bool {
	if strings.HasSuffix(filePath, "_test.go") || strings.Contains(filePath, "generated") {
		return true
	}
	return pathHasAnySegment(filePath, conventionalGoPathSegments)
}

func pathHasAnySegment(filePath string, segments map[string]struct{}) bool {
	for _, segment := range strings.Split(strings.ReplaceAll(filePath, "\\", "/"), "/") {
		if _, ok := segments[segment]; ok {
			return true
		}
	}
	return false
}

func inScopeTargets(filePath string, targets []string) bool {
	for _, target := range targets {
		normalized := strings.TrimRight(target, "/")
		if normalized == "." || pathHasPrefix(filePath, normalized) {
			return true
		}
	}
	return false
}

func matchesAnyGlob(filePath string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, err := doublestar.Match(pattern, filePath); err == nil && matched {
			return true
		}
	}
	return false
}

func parentDir(filePath string) string {
	if index := strings.LastIndex(filePath, "/"); index >= 0 {
		return filePath[:index]
	}
	return "."
}
