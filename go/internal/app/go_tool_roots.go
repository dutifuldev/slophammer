package app

import (
	"context"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/dutifuldev/slophammer/go/internal/repo"
	"github.com/dutifuldev/slophammer/go/internal/toolchecks"
)

func checkDryInModules(
	ctx context.Context,
	snapshot repo.Snapshot,
	options toolchecks.DryOptions,
	out io.Writer,
	errOut io.Writer,
	runner toolchecks.Runner,
) int {
	exitCode := ExitOK
	for _, moduleRoot := range goModuleRootsOrDefault(snapshot) {
		moduleOptions, ok := dryOptionsForModule(options, snapshot, moduleRoot)
		if !ok {
			continue
		}
		code := toolchecks.CheckDry(ctx, moduleOptions, out, errOut, runner)
		if code == ExitError {
			return ExitError
		}
		if code == ExitFindings {
			exitCode = ExitFindings
		}
	}
	return exitCode
}

func checkCRAPInModules(
	ctx context.Context,
	snapshot repo.Snapshot,
	options toolchecks.CRAPOptions,
	out io.Writer,
	errOut io.Writer,
	runner toolchecks.Runner,
) int {
	return checkInModules(ctx, snapshot, options, out, errOut, runner, setCRAPRoot, toolchecks.CheckCRAP)
}

func checkInModules[T any](
	ctx context.Context,
	snapshot repo.Snapshot,
	options T,
	out io.Writer,
	errOut io.Writer,
	runner toolchecks.Runner,
	setRoot func(*T, string),
	check func(context.Context, T, io.Writer, io.Writer, toolchecks.Runner) int,
) int {
	exitCode := ExitOK
	for _, root := range goToolRoots(optionRoot(options), snapshot) {
		moduleOptions := options
		setRoot(&moduleOptions, root)
		code := check(ctx, moduleOptions, out, errOut, runner)
		if code == ExitError {
			return ExitError
		}
		if code == ExitFindings {
			exitCode = ExitFindings
		}
	}
	return exitCode
}

func optionRoot(options any) string {
	switch typed := options.(type) {
	case toolchecks.DryOptions:
		return typed.Root
	case toolchecks.CRAPOptions:
		return typed.Root
	default:
		return ""
	}
}

func setCRAPRoot(options *toolchecks.CRAPOptions, root string) {
	options.Root = root
}

func checkMutationInModules(
	ctx context.Context,
	snapshot repo.Snapshot,
	options toolchecks.MutationOptions,
	out io.Writer,
	errOut io.Writer,
	runner toolchecks.Runner,
) int {
	resolvedOptions, err := resolveGoMutationTargets(snapshot, options)
	if err != nil {
		_, _ = io.WriteString(errOut, err.Error()+"\n")
		return ExitError
	}
	exitCode := ExitOK
	for _, moduleOptions := range mutationOptionsForModules(resolvedOptions, snapshot) {
		code := toolchecks.CheckMutation(ctx, moduleOptions, out, errOut, runner)
		if code == ExitError {
			return ExitError
		}
		if code == ExitFindings {
			exitCode = ExitFindings
		}
	}
	return exitCode
}

func goToolRoots(root string, snapshot repo.Snapshot) []string {
	moduleRoots := goModuleRootsOrDefault(snapshot)
	roots := make([]string, 0, len(moduleRoots))
	for _, moduleRoot := range moduleRoots {
		roots = append(roots, moduleToolRoot(root, moduleRoot))
	}
	return roots
}

func dryOptionsForModule(options toolchecks.DryOptions, snapshot repo.Snapshot, moduleRoot string) (toolchecks.DryOptions, bool) {
	moduleOptions := options
	moduleOptions.Root = moduleToolRoot(options.Root, moduleRoot)
	if len(options.Paths) == 0 && len(options.Exclude) == 0 {
		return moduleOptions, true
	}
	paths := dryFilePathsForModule(snapshot, moduleRoot, options.Paths, options.Exclude)
	if len(paths) == 0 {
		return toolchecks.DryOptions{}, false
	}
	moduleOptions.Paths = paths
	return moduleOptions, true
}

func dryFilePathsForModule(snapshot repo.Snapshot, moduleRoot string, includes []string, excludes []string) []string {
	roots := dryIncludeRoots(moduleRoot, includes)
	moduleRoots := goModuleRoots(snapshot)
	files := make([]string, 0)
	for filePath := range snapshot.Files {
		if !strings.HasSuffix(filePath, ".go") ||
			isUnderOtherModuleRoot(filePath, moduleRoot, moduleRoots) ||
			!isUnderDryRoot(filePath, roots) ||
			isDryExcluded(filePath, moduleRoot, excludes) {
			continue
		}
		files = append(files, trimModuleRoot(filePath, moduleRoot))
	}
	sort.Strings(files)
	return files
}

func dryIncludeRoots(moduleRoot string, includes []string) []string {
	if len(includes) == 0 {
		return []string{moduleRoot}
	}
	roots := make([]string, 0, len(includes))
	for _, include := range includes {
		root, ok := dryIncludeRoot(moduleRoot, include)
		if ok {
			roots = append(roots, root)
		}
	}
	return roots
}

func dryIncludeRoot(moduleRoot string, include string) (string, bool) {
	if strings.TrimSpace(include) == "" {
		return "", false
	}
	include = cleanSlashPath(include)
	switch {
	case include == ".":
		return moduleRoot, true
	case moduleRoot == ".":
		return include, true
	case include == moduleRoot || strings.HasPrefix(include, moduleRoot+"/"):
		return include, true
	default:
		return "", false
	}
}

func isUnderDryRoot(filePath string, roots []string) bool {
	for _, root := range roots {
		if root == "." || filePath == root || strings.HasPrefix(filePath, root+"/") {
			return true
		}
	}
	return false
}

func isUnderOtherModuleRoot(filePath string, moduleRoot string, moduleRoots []string) bool {
	for _, otherRoot := range moduleRoots {
		if otherRoot == "." || otherRoot == moduleRoot {
			continue
		}
		if moduleRoot != "." && !strings.HasPrefix(otherRoot, moduleRoot+"/") {
			continue
		}
		if strings.HasPrefix(filePath, otherRoot+"/") {
			return true
		}
	}
	return false
}

func isDryExcluded(filePath string, moduleRoot string, excludes []string) bool {
	modulePath := trimModuleRoot(filePath, moduleRoot)
	for _, exclude := range excludes {
		exclude = cleanSlashPath(exclude)
		if exclude == "" {
			continue
		}
		if pathMatchesDryPattern(filePath, exclude) || pathMatchesDryPattern(modulePath, exclude) {
			return true
		}
	}
	return false
}

func pathMatchesDryPattern(filePath string, pattern string) bool {
	matched, _ := doublestar.Match(pattern, filePath)
	return matched
}

func mutationOptionsForModules(options toolchecks.MutationOptions, snapshot repo.Snapshot) []toolchecks.MutationOptions {
	targets := mutationTargets(options)
	if len(targets) == 0 {
		options.Root = firstGoToolRoot(options.Root, snapshot)
		return []toolchecks.MutationOptions{options}
	}

	moduleRoots := goModuleRoots(snapshot)
	if len(moduleRoots) == 0 {
		options.Target = ""
		options.Targets = targets
		return []toolchecks.MutationOptions{options}
	}

	byRoot := map[string][]string{}
	for _, target := range targets {
		moduleRoot := targetModuleRoot(target, moduleRoots)
		toolRoot := moduleToolRoot(options.Root, moduleRoot)
		byRoot[toolRoot] = append(byRoot[toolRoot], trimModuleRoot(target, moduleRoot))
	}

	roots := make([]string, 0, len(byRoot))
	for root := range byRoot {
		roots = append(roots, root)
	}
	sort.Strings(roots)

	grouped := make([]toolchecks.MutationOptions, 0, len(roots))
	for _, root := range roots {
		grouped = append(grouped, toolchecks.MutationOptions{
			Root:    root,
			Targets: byRoot[root],
			Scan:    options.Scan,
		})
	}
	return grouped
}

func firstGoToolRoot(root string, snapshot repo.Snapshot) string {
	roots := goToolRoots(root, snapshot)
	return roots[0]
}

func moduleToolRoot(root string, moduleRoot string) string {
	if moduleRoot == "." {
		return commandRoot(root)
	}
	return filepath.Join(commandRoot(root), filepath.FromSlash(moduleRoot))
}

func goModuleRoots(snapshot repo.Snapshot) []string {
	roots := make([]string, 0)
	for filePath := range snapshot.Files {
		if path.Base(filePath) != "go.mod" || isSkippedGoModulePath(filePath) {
			continue
		}
		root := path.Dir(filePath)
		roots = append(roots, root)
	}
	sort.Strings(roots)
	return roots
}

func goModuleRootsOrDefault(snapshot repo.Snapshot) []string {
	roots := goModuleRoots(snapshot)
	if len(roots) > 0 {
		return roots
	}
	return []string{"."}
}

func targetModuleRoot(target string, moduleRoots []string) string {
	for i := len(moduleRoots) - 1; i >= 0; i-- {
		moduleRoot := moduleRoots[i]
		if moduleRoot == "." || target == moduleRoot || strings.HasPrefix(target, moduleRoot+"/") {
			return moduleRoot
		}
	}
	return "."
}

func trimModuleRoot(target string, moduleRoot string) string {
	if moduleRoot == "." {
		return target
	}
	return strings.TrimPrefix(strings.TrimPrefix(target, moduleRoot), "/")
}

func cleanSlashPath(filePath string) string {
	return path.Clean(strings.ReplaceAll(filePath, "\\", "/"))
}

func mutationTargets(options toolchecks.MutationOptions) []string {
	if options.Target != "" {
		return []string{options.Target}
	}
	targets := make([]string, 0, len(options.Targets))
	for _, target := range options.Targets {
		if strings.TrimSpace(target) != "" {
			targets = append(targets, target)
		}
	}
	return targets
}

func isSkippedGoModulePath(filePath string) bool {
	return filePath == "vendor/go.mod" ||
		strings.Contains(filePath, "/vendor/") ||
		strings.HasPrefix(filePath, "fixtures/") ||
		strings.HasPrefix(filePath, "templates/")
}
