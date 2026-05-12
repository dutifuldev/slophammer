package rules

import (
	"context"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/dutifuldev/slophammer/go/internal/repo"
	"gopkg.in/yaml.v3"
)

type goStaticRule struct {
	definition Definition
	satisfied  func(repo.Snapshot) bool
}

func newGoModuleRule(definition Definition) Rule {
	return goStaticRule{definition: definition, satisfied: hasGoModule}
}

func newGoTestsRule(definition Definition) Rule {
	return goStaticRule{definition: definition, satisfied: hasGoTestCommand}
}

func newGoVetRule(definition Definition) Rule {
	return goStaticRule{definition: definition, satisfied: hasGoVetCommand}
}

func newGoLintRule(definition Definition) Rule {
	return goStaticRule{definition: definition, satisfied: hasGoLintConfigAndCommand}
}

func newGoCoverageRule(definition Definition) Rule {
	return goStaticRule{definition: definition, satisfied: hasGoCoverageGate}
}

func newGoComplexityRule(definition Definition) Rule {
	return goStaticRule{definition: definition, satisfied: hasGoComplexityLint}
}

func newGoDryRule(definition Definition) Rule {
	return goStaticRule{definition: definition, satisfied: hasDry4GoCommand}
}

func newGoCRAPRule(definition Definition) Rule {
	return goStaticRule{definition: definition, satisfied: hasCRAP4GoCommand}
}

func newGoMutationRule(definition Definition) Rule {
	return goStaticRule{definition: definition, satisfied: hasMutate4GoCommand}
}

func (r goStaticRule) Metadata() Metadata {
	return r.definition.Metadata()
}

func (r goStaticRule) Check(_ context.Context, snapshot repo.Snapshot) []Finding {
	roots := goProjectRoots(snapshot)
	if len(roots) == 0 {
		return nil
	}
	for _, root := range roots {
		if !r.satisfied(goProjectSnapshot(snapshot, root, roots)) {
			return []Finding{finding(r.definition)}
		}
	}
	return nil
}

func hasGoModule(snapshot repo.Snapshot) bool {
	return snapshot.HasFileNamedFold("go.mod")
}

func hasGoTestCommand(snapshot repo.Snapshot) bool {
	return hasCommandPattern(snapshot, goTestAllPackagesPattern)
}

func hasGoVetCommand(snapshot repo.Snapshot) bool {
	return hasCommandPattern(snapshot, goVetAllPackagesPattern)
}

func hasGoLintConfigAndCommand(snapshot repo.Snapshot) bool {
	return hasGolangCIConfig(snapshot) && hasCommand(snapshot, "golangci-lint", "golangci/golangci-lint-action")
}

func hasGoCoverageGate(snapshot repo.Snapshot) bool {
	for _, file := range commandFiles(snapshot) {
		if strings.Contains(file.Content, "-coverprofile") &&
			strings.Contains(file.Content, "go tool cover") &&
			hasCoverageThreshold(file.Content) {
			return true
		}
	}
	return false
}

func hasGoComplexityLint(snapshot repo.Snapshot) bool {
	for _, file := range golangCIConfigFiles(snapshot) {
		if configEnablesComplexityLinter(file.Content) {
			return true
		}
	}
	return false
}

func hasDry4GoCommand(snapshot repo.Snapshot) bool {
	return hasCommand(snapshot, "dry4go", "github.com/unclebob/dry4go/cmd/dry4go")
}

func hasCRAP4GoCommand(snapshot repo.Snapshot) bool {
	return hasCommand(snapshot, "crap4go", "github.com/unclebob/crap4go/cmd/crap4go")
}

func hasMutate4GoCommand(snapshot repo.Snapshot) bool {
	return hasCommand(snapshot, "mutate4go", "github.com/unclebob/mutate4go/cmd/mutate4go")
}

func goProjectRoots(snapshot repo.Snapshot) []string {
	rootsByPath := map[string]struct{}{}
	for filePath := range snapshot.Files {
		if isEmbeddedFixturePath(filePath) {
			continue
		}
		if strings.EqualFold(path.Base(filePath), "go.mod") {
			root := path.Dir(filePath)
			if root == "." {
				root = ""
			}
			rootsByPath[root] = struct{}{}
		}
	}
	if len(rootsByPath) == 0 && hasUnscopedGoSignal(snapshot) {
		rootsByPath[""] = struct{}{}
	}
	if len(rootsByPath) > 0 && hasGoSourceOutsideModuleRoots(snapshot, rootsByPath) {
		rootsByPath[""] = struct{}{}
	}
	roots := make([]string, 0, len(rootsByPath))
	for root := range rootsByPath {
		roots = append(roots, root)
	}
	sort.Strings(roots)
	return roots
}

func hasUnscopedGoSignal(snapshot repo.Snapshot) bool {
	for filePath := range snapshot.Files {
		if isEmbeddedFixturePath(filePath) {
			continue
		}
		if strings.HasSuffix(filePath, ".go") {
			return true
		}
	}
	return hasCommand(goProjectSnapshot(snapshot, "", []string{""}), "go test", "go vet")
}

func hasGoSourceOutsideModuleRoots(snapshot repo.Snapshot, rootsByPath map[string]struct{}) bool {
	for filePath := range snapshot.Files {
		if isEmbeddedFixturePath(filePath) || !strings.HasSuffix(filePath, ".go") {
			continue
		}
		if !isUnderAnyGoRoot(filePath, rootsByPath) {
			return true
		}
	}
	return false
}

func isUnderAnyGoRoot(filePath string, rootsByPath map[string]struct{}) bool {
	for root := range rootsByPath {
		if root == "" {
			return true
		}
		if strings.HasPrefix(filePath, root+"/") {
			return true
		}
	}
	return false
}

func goProjectSnapshot(snapshot repo.Snapshot, root string, roots []string) repo.Snapshot {
	files := map[string]repo.File{}
	for filePath, file := range snapshot.Files {
		scopedFile, ok := scopedGoProjectFile(filePath, file, root, roots)
		if !ok {
			continue
		}
		files[scopedFile.Path] = scopedFile
	}
	return repo.NewSnapshot(snapshot.Root, files)
}

func scopedGoProjectFile(filePath string, file repo.File, root string, roots []string) (repo.File, bool) {
	if isEmbeddedFixturePath(filePath) {
		return repo.File{}, false
	}
	if isWorkflowFilePath(filePath) {
		return file, workflowAppliesToGoRoot(file.Content, root, roots)
	}
	if root == "" {
		return file, !isUnderOtherGoRoot(filePath, root, roots)
	}
	prefix := root + "/"
	if !strings.HasPrefix(filePath, prefix) || isUnderOtherGoRoot(filePath, root, roots) {
		return repo.File{}, false
	}
	scopedPath := strings.TrimPrefix(filePath, prefix)
	return repo.File{Path: scopedPath, Content: file.Content}, true
}

func isUnderOtherGoRoot(filePath, root string, roots []string) bool {
	for _, otherRoot := range roots {
		if otherRoot == "" || otherRoot == root {
			continue
		}
		if root == "" && strings.HasPrefix(filePath, otherRoot+"/") {
			return true
		}
		if strings.HasPrefix(otherRoot, root+"/") && strings.HasPrefix(filePath, otherRoot+"/") {
			return true
		}
	}
	return false
}

func workflowAppliesToGoRoot(content, root string, roots []string) bool {
	if root != "" {
		if len(roots) == 1 && roots[0] == root {
			return true
		}
		return workflowMentionsGoRoot(content, root)
	}
	for _, otherRoot := range roots {
		if otherRoot != "" && workflowMentionsGoRoot(content, otherRoot) {
			return false
		}
	}
	return true
}

func workflowMentionsGoRoot(content, root string) bool {
	normalized := strings.ReplaceAll(content, "\\", "/")
	for _, needle := range []string{
		"working-directory: " + root,
		"working-directory: \"" + root + "\"",
		"working-directory: '" + root + "'",
		"cd " + root,
		"cd \"" + root + "\"",
		"cd '" + root + "'",
		root + "/",
	} {
		if strings.Contains(normalized, needle) {
			return true
		}
	}
	return false
}

func isWorkflowFilePath(filePath string) bool {
	dir, name := path.Split(filePath)
	return dir == ".github/workflows/" && (strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml"))
}

func isEmbeddedFixturePath(filePath string) bool {
	for _, segment := range strings.Split(strings.ReplaceAll(filePath, "\\", "/"), "/") {
		switch segment {
		case "examples", "fixtures", "samples", "templates", "testdata":
			return true
		}
	}
	return false
}

func hasCommand(snapshot repo.Snapshot, needles ...string) bool {
	return repo.ContainsAny(commandFiles(snapshot), needles...)
}

func hasGolangCIConfig(snapshot repo.Snapshot) bool {
	return len(golangCIConfigFiles(snapshot)) > 0
}

func golangCIConfigFiles(snapshot repo.Snapshot) []repo.File {
	return snapshot.FilesNamedFold(".golangci.yml", ".golangci.yaml")
}

func hasCoverageThreshold(content string) bool {
	return coverageThresholdPattern.MatchString(content)
}

func configEnablesComplexityLinter(content string) bool {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return false
	}
	root := yamlRoot(&document)
	linters := yamlMappingValue(root, "linters")
	disable := yamlMappingValue(linters, "disable")
	if yamlScalarEquals(yamlMappingValue(linters, "default"), "all") {
		return !yamlSequenceContainsAll(disable, "cyclop", "gocognit", "gocyclo")
	}
	enable := yamlMappingValue(linters, "enable")
	return yamlSequenceContains(enable, "cyclop", "gocognit", "gocyclo")
}

func yamlRoot(node *yaml.Node) *yaml.Node {
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return node
}

func yamlMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func yamlSequenceContains(node *yaml.Node, values ...string) bool {
	if node == nil || node.Kind != yaml.SequenceNode {
		return false
	}
	for _, item := range node.Content {
		for _, value := range values {
			if item.Value == value {
				return true
			}
		}
	}
	return false
}

func yamlSequenceContainsAll(node *yaml.Node, values ...string) bool {
	for _, value := range values {
		if !yamlSequenceContains(node, value) {
			return false
		}
	}
	return true
}

func yamlScalarEquals(node *yaml.Node, value string) bool {
	return node != nil && node.Kind == yaml.ScalarNode && node.Value == value
}

func commandFiles(snapshot repo.Snapshot) []repo.File {
	filesByPath := map[string]repo.File{}
	for _, file := range snapshot.WorkflowFiles() {
		filesByPath[file.Path] = file
	}
	for _, file := range snapshot.FilesNamedFold("Makefile", "Taskfile.yml", "Taskfile.yaml", "justfile") {
		filesByPath[file.Path] = file
	}
	for _, file := range snapshot.FilesUnder("scripts") {
		filesByPath[file.Path] = file
	}
	for _, file := range snapshot.FilesUnder("go/scripts") {
		filesByPath[file.Path] = file
	}
	for path, file := range snapshot.Files {
		if isScriptPath(path) {
			filesByPath[file.Path] = file
		}
	}
	files := make([]repo.File, 0, len(filesByPath))
	for _, file := range filesByPath {
		file.Content = stripCommentLines(file.Content)
		file.Content = joinShellContinuations(file.Content)
		if strings.TrimSpace(file.Content) != "" {
			files = append(files, file)
		}
	}
	return files
}

func isScriptPath(filePath string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(filePath, "\\", "/"))
	return strings.HasPrefix(normalized, "scripts/") || strings.Contains(normalized, "/scripts/")
}

func stripCommentLines(content string) string {
	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		beforeComment, _, _ := strings.Cut(line, "#")
		if strings.TrimSpace(beforeComment) == "" {
			continue
		}
		kept = append(kept, beforeComment)
	}
	return strings.Join(kept, "\n")
}

func joinShellContinuations(content string) string {
	content = strings.ReplaceAll(content, "\\\r\n", " ")
	return strings.ReplaceAll(content, "\\\n", " ")
}

func hasCommandPattern(snapshot repo.Snapshot, pattern *regexp.Regexp) bool {
	for _, file := range commandFiles(snapshot) {
		if pattern.MatchString(file.Content) {
			return true
		}
	}
	return false
}

func finding(definition Definition) Finding {
	return Finding{
		RuleID:   definition.ID,
		Severity: definition.Severity,
		Path:     definition.Path,
		Message:  definition.Message,
	}
}

var (
	goTestAllPackagesPattern = regexp.MustCompile(`(?m)\bgo\s+test\b[^\n#;&|]*\./\.\.`)
	goVetAllPackagesPattern  = regexp.MustCompile(`(?m)\bgo\s+vet\b[^\n#;&|]*\./\.\.`)
	coverageThresholdPattern = regexp.MustCompile(`(?im)\b(total|cover|coverage|minimum|threshold|required)\b[^\n]*(>=|<=|-ge\b|-le\b|-gt\b|-lt\b)|(?:>=|<=|-ge\b|-le\b|-gt\b|-lt\b)[^\n]*\b(total|cover|coverage|minimum|threshold|required)\b`)
)
