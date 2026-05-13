package rules

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/dutifuldev/slophammer/go/internal/config"
	"github.com/dutifuldev/slophammer/go/internal/repo"
)

type goDependencyBoundariesRule struct {
	definition Definition
}

func newGoDependencyBoundariesRule(definition Definition) Rule {
	return goDependencyBoundariesRule{definition: definition}
}

func (r goDependencyBoundariesRule) Metadata() Metadata {
	return r.definition.Metadata()
}

func (r goDependencyBoundariesRule) Check(context.Context, repo.Snapshot) []Finding {
	return nil
}

func (r goDependencyBoundariesRule) CheckWithConfig(_ context.Context, snapshot repo.Snapshot, cfg config.Config) []Finding {
	if len(cfg.Go.DependencyBoundaries) == 0 {
		return nil
	}
	moduleRoots := goModuleRoots(snapshot)
	modulePaths := goModulePaths(snapshot, moduleRoots)
	findingsByKey := map[string]Finding{}
	for _, file := range goSourceFiles(snapshot) {
		root := sourceModuleRoot(file.Path, moduleRoots)
		for _, importPath := range goImports(file) {
			localPath, ok := localImportPath(importPath, root, modulePaths[root])
			if !ok {
				continue
			}
			for _, boundary := range cfg.Go.DependencyBoundaries {
				if boundaryAllowsImport(file.Path, localPath, root, boundary) {
					continue
				}
				key := file.Path + "\x00" + localPath
				findingsByKey[key] = Finding{
					RuleID:   r.definition.ID,
					Severity: r.definition.Severity,
					Path:     file.Path,
					Message:  fmt.Sprintf("%s imports %s outside configured dependency boundaries", file.Path, localPath),
				}
			}
		}
	}
	return sortedFindings(findingsByKey)
}

func goModuleRoots(snapshot repo.Snapshot) []string {
	roots := goProjectRoots(snapshot)
	if len(roots) == 0 {
		return []string{""}
	}
	return roots
}

func goModulePaths(snapshot repo.Snapshot, roots []string) map[string]string {
	paths := map[string]string{}
	for _, root := range roots {
		filePath := path.Join(root, "go.mod")
		if root == "" {
			filePath = "go.mod"
		}
		file, ok := snapshot.Files[filePath]
		if !ok {
			continue
		}
		if modulePath := parseModulePath(file.Content); modulePath != "" {
			paths[root] = modulePath
		}
	}
	return paths
}

func parseModulePath(content string) string {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return strings.Trim(fields[1], `"`)
		}
	}
	return ""
}

func goSourceFiles(snapshot repo.Snapshot) []repo.File {
	files := make([]repo.File, 0)
	for filePath, file := range snapshot.Files {
		if isEmbeddedFixturePath(filePath) || !strings.HasSuffix(filePath, ".go") {
			continue
		}
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files
}

func sourceModuleRoot(filePath string, roots []string) string {
	for _, root := range roots {
		if root != "" && strings.HasPrefix(filePath, root+"/") {
			return root
		}
	}
	return ""
}

func goImports(file repo.File) []string {
	parsed, err := parser.ParseFile(token.NewFileSet(), file.Path, file.Content, parser.ImportsOnly)
	if err != nil {
		return nil
	}
	imports := make([]string, 0, len(parsed.Imports))
	for _, spec := range parsed.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err == nil {
			imports = append(imports, importPath)
		}
	}
	return imports
}

func localImportPath(importPath string, root string, modulePath string) (string, bool) {
	if modulePath == "" {
		return "", false
	}
	if importPath == modulePath {
		return cleanBoundaryPath(root), true
	}
	prefix := modulePath + "/"
	if !strings.HasPrefix(importPath, prefix) {
		return "", false
	}
	return cleanBoundaryPath(path.Join(root, strings.TrimPrefix(importPath, prefix))), true
}

func boundaryAllowsImport(filePath string, localImport string, moduleRoot string, boundary config.DependencyBoundary) bool {
	if !pathMatchesBoundary(filePath, moduleRoot, boundary.From) {
		return true
	}
	for _, allow := range boundary.Allow {
		if pathMatchesBoundary(localImport, moduleRoot, allow) {
			return true
		}
	}
	return false
}

func pathMatchesBoundary(filePath string, moduleRoot string, boundaryPath string) bool {
	candidate := cleanBoundaryPath(boundaryPath)
	if pathHasPrefix(filePath, candidate) {
		return true
	}
	if moduleRoot == "" {
		return false
	}
	return pathHasPrefix(filePath, cleanBoundaryPath(path.Join(moduleRoot, candidate)))
}

func pathHasPrefix(filePath string, prefix string) bool {
	return filePath == prefix || strings.HasPrefix(filePath, prefix+"/")
}

func cleanBoundaryPath(value string) string {
	return strings.TrimPrefix(path.Clean(strings.ReplaceAll(value, "\\", "/")), "./")
}

func sortedFindings(findingsByKey map[string]Finding) []Finding {
	keys := make([]string, 0, len(findingsByKey))
	for key := range findingsByKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	findings := make([]Finding, 0, len(keys))
	for _, key := range keys {
		findings = append(findings, findingsByKey[key])
	}
	return findings
}
