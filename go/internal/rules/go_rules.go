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
	return goStaticRule{definition: definition, satisfied: hasCRAP4GoGate}
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
	return hasGolangCIConfig(snapshot) && hasGolangCICommand(snapshot)
}

func hasGoCoverageGate(snapshot repo.Snapshot) bool {
	for _, file := range commandFiles(snapshot) {
		for _, content := range commandSections(file) {
			if strings.Contains(content, "-coverprofile") &&
				strings.Contains(content, "go tool cover") &&
				hasCoverageThreshold(content) {
				return true
			}
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
	for _, file := range commandFiles(snapshot) {
		for _, content := range commandSections(file) {
			if contentHasGoToolCommand(content, "dry4go", "github.com/unclebob/dry4go/cmd/dry4go") ||
				contentHasSlophammerGoCommand(content, "dry", "") {
				return true
			}
		}
	}
	return false
}

func hasCRAP4GoGate(snapshot repo.Snapshot) bool {
	for _, file := range commandFiles(snapshot) {
		for _, content := range commandSections(file) {
			if contentHasSlophammerGoCommand(content, "crap", "--max-score") {
				return true
			}
			if !hasCRAPThreshold(content) {
				continue
			}
			if contentHasGoToolCommand(content, "crap4go", "github.com/unclebob/crap4go/cmd/crap4go") {
				return true
			}
		}
	}
	return false
}

func hasMutate4GoCommand(snapshot repo.Snapshot) bool {
	if hasDirectMutate4GoCommand(snapshot) {
		return true
	}
	for _, file := range commandFiles(snapshot) {
		for _, content := range commandSections(file) {
			if contentHasSlophammerGoCommand(content, "mutate", "--target") {
				return true
			}
		}
	}
	return false
}

func hasDirectMutate4GoCommand(snapshot repo.Snapshot) bool {
	for _, file := range commandFiles(snapshot) {
		for _, content := range commandSections(file) {
			if contentHasDirectMutate4GoCommand(content) {
				return true
			}
		}
	}
	return false
}

func contentHasDirectMutate4GoCommand(content string) bool {
	tokensByLine := commandTokensByLine(content)
	for _, tokens := range tokensByLine {
		if lineHasDirectMutate4GoCommand(tokens) {
			return true
		}
	}
	return false
}

func lineHasDirectMutate4GoCommand(tokens []string) bool {
	for i, token := range tokens {
		token = cleanCommandToken(token)
		if isGoToolPackageToken(token, "github.com/unclebob/mutate4go/cmd/mutate4go") && isGoRunPackage(tokens, i) {
			return hasMutationTargetAfter(tokens, i)
		}
		if isToolBinaryToken(token, "mutate4go") && isCommandToken(tokens, i) {
			return hasMutationTargetAfter(tokens, i)
		}
	}
	return false
}

func contentHasGoToolCommand(content string, binaryName string, packageNeedle string) bool {
	for _, tokens := range commandTokensByLine(content) {
		if lineHasGoToolCommand(tokens, binaryName, packageNeedle) {
			return true
		}
	}
	return false
}

func lineHasGoToolCommand(tokens []string, binaryName string, packageNeedle string) bool {
	for i, token := range tokens {
		token = cleanCommandToken(token)
		if isGoToolPackageToken(token, packageNeedle) {
			return isGoRunPackage(tokens, i)
		}
		if isToolBinaryToken(token, binaryName) {
			return isCommandToken(tokens, i)
		}
	}
	return false
}

func contentHasSlophammerGoCommand(content string, subcommand string, requiredFlag string) bool {
	return contentHasCommandLine(content, func(tokens []string) bool {
		return lineHasSlophammerGoCommand(tokens, subcommand, requiredFlag)
	})
}

func contentHasCommandLine(content string, match func([]string) bool) bool {
	for _, tokens := range commandTokensByLine(content) {
		if match(tokens) {
			return true
		}
	}
	return false
}

func lineHasSlophammerGoCommand(tokens []string, subcommand string, requiredFlag string) bool {
	for i := 0; i+2 < len(tokens); i++ {
		if !isSlophammerCommandToken(cleanCommandToken(tokens[i])) {
			continue
		}
		if !isCommandToken(tokens, i) && !isGoRunPackage(tokens, i) {
			continue
		}
		if cleanCommandToken(tokens[i+1]) == "go" &&
			cleanCommandToken(tokens[i+2]) == subcommand &&
			lineHasRequiredFlag(tokens[i+3:], requiredFlag) {
			return true
		}
	}
	return false
}

func commandTokensByLine(content string) [][]string {
	lines := strings.Split(content, "\n")
	tokenLines := make([][]string, 0, len(lines))
	for _, line := range lines {
		tokens := strings.Fields(line)
		if len(tokens) > 0 {
			tokenLines = append(tokenLines, tokens)
		}
	}
	return tokenLines
}

func isGoRunPackage(tokens []string, packageIndex int) bool {
	if packageIndex < 2 {
		return false
	}
	return cleanCommandToken(tokens[packageIndex-2]) == "go" && cleanCommandToken(tokens[packageIndex-1]) == "run"
}

func isCommandToken(tokens []string, commandIndex int) bool {
	if commandIndex == 0 {
		return true
	}
	previous := cleanCommandToken(tokens[commandIndex-1])
	return isWorkflowRunToken(previous) || isShellSeparator(previous)
}

func hasMutationTargetAfter(tokens []string, commandIndex int) bool {
	for _, token := range tokens[commandIndex+1:] {
		target := cleanCommandToken(token)
		if target == "" {
			continue
		}
		if isShellSeparator(target) {
			return false
		}
		if strings.HasPrefix(target, "-") {
			continue
		}
		if strings.HasSuffix(target, ".go") {
			return true
		}
	}
	return false
}

func lineHasRequiredFlag(tokens []string, requiredFlag string) bool {
	if requiredFlag == "" {
		return true
	}
	for _, token := range tokens {
		if cleanCommandToken(token) == requiredFlag {
			return true
		}
	}
	return false
}

func isGoToolPackageToken(token string, packageNeedle string) bool {
	return strings.Contains(token, packageNeedle)
}

func isToolBinaryToken(token string, binaryName string) bool {
	return path.Base(token) == binaryName
}

func isSlophammerCommandToken(token string) bool {
	base := path.Base(token)
	return base == "slophammer" || base == "slophammer.exe"
}

func isWorkflowRunToken(token string) bool {
	return strings.TrimSuffix(token, ":") == "run"
}

func cleanCommandToken(token string) string {
	return strings.Trim(token, " \t\r\n'\";")
}

func isShellSeparator(token string) bool {
	switch token {
	case "|", "||", "&", "&&":
		return true
	default:
		return false
	}
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
	return hasCommandPattern(goProjectSnapshot(snapshot, "", []string{""}), goCommandPattern)
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
	priorities := map[string]int{}
	hasLocalGoConfig := hasModuleLocalGoConfig(snapshot, root)
	for filePath, file := range snapshot.Files {
		if hasLocalGoConfig && isRepoRootGoConfigFile(filePath) {
			continue
		}
		scopedFile, ok := scopedGoProjectFile(filePath, file, root, roots)
		if !ok {
			continue
		}
		priority := scopedGoProjectFilePriority(filePath, root)
		if priorities[scopedFile.Path] > priority {
			continue
		}
		files[scopedFile.Path] = scopedFile
		priorities[scopedFile.Path] = priority
	}
	return repo.NewSnapshot(snapshot.Root, files)
}

func hasModuleLocalGoConfig(snapshot repo.Snapshot, root string) bool {
	if root == "" {
		return false
	}
	return snapshot.HasFileFold(root+"/.golangci.yml") || snapshot.HasFileFold(root+"/.golangci.yaml")
}

func scopedGoProjectFilePriority(filePath string, root string) int {
	if root != "" && strings.HasPrefix(filePath, root+"/") {
		return 2
	}
	return 1
}

func scopedGoProjectFile(filePath string, file repo.File, root string, roots []string) (repo.File, bool) {
	if isEmbeddedFixturePath(filePath) {
		return repo.File{}, false
	}
	if isWorkflowFilePath(filePath) {
		content, ok := scopedWorkflowContent(file.Content, root, roots)
		if !ok {
			return repo.File{}, false
		}
		return repo.File{Path: file.Path, Content: content}, true
	}
	if root == "" {
		return file, !isUnderOtherGoRoot(filePath, root, roots)
	}
	if isRepoRootGoConfigFile(filePath) {
		return file, true
	}
	if isRepoRootCommandFile(filePath) {
		content, ok := scopedRootCommandContent(filePath, file.Content, root, roots)
		if !ok {
			return repo.File{}, false
		}
		return repo.File{Path: file.Path, Content: content}, true
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

func scopedWorkflowContent(content, root string, roots []string) (string, bool) {
	if onlyGoRoot(root, roots) {
		return content, true
	}
	return filterWorkflowContentForRoot(content, root, roots)
}

func scopedRootCommandContent(filePath string, content string, root string, roots []string) (string, bool) {
	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines))
	inRootBlock := false
	carryRootContext := carriesRootCommandContext(filePath)
	for _, line := range lines {
		if workflowMentionsOtherGoRoot(line, root, roots) {
			inRootBlock = false
			continue
		}
		if workflowMentionsGoRoot(line, root, roots) {
			inRootBlock = carryRootContext
			kept = append(kept, line)
			continue
		}
		if carryRootContext && inRootBlock {
			kept = append(kept, line)
		}
	}
	scoped := strings.Join(kept, "\n")
	return scoped, strings.TrimSpace(scoped) != ""
}

func filterWorkflowContentForRoot(content, root string, roots []string) (string, bool) {
	blocks := workflowStepBlocks(content)
	kept := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if workflowStepAppliesToRoot(block, root, roots) {
			kept = append(kept, block)
		}
	}
	scoped := strings.Join(kept, workflowStepBoundary)
	return scoped, strings.TrimSpace(scoped) != ""
}

func onlyGoRoot(root string, roots []string) bool {
	return root == "" && len(roots) == 1 && roots[0] == root
}

func workflowStepAppliesToRoot(content, root string, roots []string) bool {
	if root == "" {
		return !workflowMentionsOtherGoRoot(content, root, roots)
	}
	return workflowMentionsGoRoot(content, root, roots)
}

func workflowStepBlocks(content string) []string {
	lines := strings.Split(content, "\n")
	scan := workflowStepScan{}
	for _, line := range lines {
		scan.visitLine(line)
	}
	blocks := appendWorkflowStepBlock(scan.blocks, scan.current)
	if len(blocks) == 0 {
		return []string{content}
	}
	return blocks
}

type workflowStepScan struct {
	blocks        []string
	globalContext []string
	jobContext    []string
	current       []string
	inJobs        bool
	inSteps       bool
	seenJob       bool
}

func (s *workflowStepScan) visitLine(line string) {
	if s.visitWorkflowStructure(line) {
		return
	}
	if len(s.current) == 0 && isWorkflowWorkingDirectory(line) {
		s.recordWorkingDirectory(line)
	}
	if s.inJobs && s.inSteps && isWorkflowStepStart(line) {
		s.startStep(line)
		return
	}
	if len(s.current) > 0 {
		s.current = append(s.current, line)
	}
}

func (s *workflowStepScan) visitWorkflowStructure(line string) bool {
	if s.enterJobs(line) {
		return true
	}
	if s.inJobs && isWorkflowJobStart(line) {
		s.startJob()
		return true
	}
	if s.inJobs && isWorkflowStepsStart(line) {
		s.inSteps = true
		return true
	}
	return false
}

func (s *workflowStepScan) enterJobs(line string) bool {
	if strings.TrimSpace(line) != "jobs:" {
		return false
	}
	s.inJobs = true
	return true
}

func (s *workflowStepScan) startJob() {
	s.blocks = appendWorkflowStepBlock(s.blocks, s.current)
	s.current = nil
	s.jobContext = append([]string{}, s.globalContext...)
	s.inSteps = false
	s.seenJob = true
}

func (s *workflowStepScan) recordWorkingDirectory(line string) {
	if s.seenJob {
		s.jobContext = append(s.jobContext, line)
		return
	}
	s.globalContext = append(s.globalContext, line)
	s.jobContext = append([]string{}, s.globalContext...)
}

func (s *workflowStepScan) startStep(line string) {
	s.blocks = appendWorkflowStepBlock(s.blocks, s.current)
	s.current = append(append([]string{}, s.jobContext...), line)
}

func appendWorkflowStepBlock(blocks []string, lines []string) []string {
	if len(lines) == 0 {
		return blocks
	}
	return append(blocks, strings.Join(lines, "\n"))
}

func workflowMentionsOtherGoRoot(content, root string, roots []string) bool {
	for _, otherRoot := range roots {
		if otherRoot != "" && otherRoot != root && workflowMentionsGoRoot(content, otherRoot, roots) {
			return true
		}
	}
	return false
}

func workflowMentionsGoRoot(content, root string, roots []string) bool {
	normalized := strings.ReplaceAll(content, "\\", "/")
	return workflowReferencesRootExact(normalized, root, roots, workingDirectoryPattern(root)) ||
		workflowReferencesRootExact(normalized, root, roots, cdRootPattern(root)) ||
		workflowReferencesRootSubpath(normalized, root, roots)
}

func workflowReferencesRootExact(content, root string, roots []string, pattern *regexp.Regexp) bool {
	for _, match := range pattern.FindAllStringIndex(content, -1) {
		if !rootPathIsNestedModule(content[match[0]:], root, roots) {
			return true
		}
	}
	return false
}

func workflowReferencesRootSubpath(content, root string, roots []string) bool {
	for _, match := range rootSubpathPattern(root).FindAllStringIndex(content, -1) {
		if !rootPathIsNestedModule(content[match[0]:], root, roots) {
			return true
		}
	}
	return false
}

func rootPathIsNestedModule(match, root string, roots []string) bool {
	start := strings.TrimLeft(match, " \t\r\n'\";:&|()[]{}")
	start = strings.TrimPrefix(start, "./")
	for _, otherRoot := range roots {
		if otherRoot == "" || otherRoot == root || !strings.HasPrefix(otherRoot, root+"/") {
			continue
		}
		if strings.HasPrefix(start, otherRoot+"/") || hasRootPathBoundary(start, otherRoot) {
			return true
		}
	}
	return false
}

func hasRootPathBoundary(value, root string) bool {
	if !strings.HasPrefix(value, root) {
		return false
	}
	if len(value) == len(root) {
		return true
	}
	return strings.ContainsRune(" \t\r\n'\";:&|)]}", rune(value[len(root)]))
}

func workingDirectoryPattern(root string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)\bworking-directory:\s*['"]?(?:\./)?` + regexp.QuoteMeta(root) + `['"]?(?:[[:space:]]|$)`)
}

func cdRootPattern(root string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)(?:^|[[:space:];&|])cd\s+['"]?(?:\./)?` + regexp.QuoteMeta(root) + `['"]?(?:[[:space:];&|]|$)`)
}

func rootSubpathPattern(root string) *regexp.Regexp {
	return regexp.MustCompile(`(?:^|[^[:alnum:]_./-])(?:\./)?` + regexp.QuoteMeta(root) + `/`)
}

func isWorkflowStepStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "- ")
}

func isWorkflowStepsStart(line string) bool {
	return strings.TrimSpace(line) == "steps:"
}

func isWorkflowJobStart(line string) bool {
	if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
		trimmed := strings.TrimSpace(line)
		return strings.HasSuffix(trimmed, ":")
	}
	return false
}

func isWorkflowWorkingDirectory(line string) bool {
	return strings.Contains(strings.TrimSpace(line), "working-directory:")
}

func isWorkflowFilePath(filePath string) bool {
	dir, name := path.Split(filePath)
	return dir == ".github/workflows/" && (strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml"))
}

func isRepoRootGoConfigFile(filePath string) bool {
	return filePath == ".golangci.yml" || filePath == ".golangci.yaml"
}

func isRepoRootCommandFile(filePath string) bool {
	switch filePath {
	case "Makefile", "Taskfile.yml", "Taskfile.yaml", "justfile":
		return true
	default:
		return strings.HasPrefix(filePath, "scripts/")
	}
}

func carriesRootCommandContext(filePath string) bool {
	return strings.HasPrefix(filePath, "scripts/")
}

func isEmbeddedFixturePath(filePath string) bool {
	for _, segment := range strings.Split(strings.ReplaceAll(filePath, "\\", "/"), "/") {
		switch segment {
		case "examples", "fixtures", "samples", "templates", "testdata", "vendor":
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

func hasGolangCICommand(snapshot repo.Snapshot) bool {
	if hasCommand(snapshot, "golangci/golangci-lint-action") {
		return true
	}
	return hasCommandPattern(snapshot, golangCILintRunPattern)
}

func golangCIConfigFiles(snapshot repo.Snapshot) []repo.File {
	return snapshot.FilesNamedFold(".golangci.yml", ".golangci.yaml")
}

func hasCoverageThreshold(content string) bool {
	return coverageThresholdPattern.MatchString(content) || strictCoverageThresholdPattern.MatchString(content)
}

func hasCRAPThreshold(content string) bool {
	return crapThresholdPattern.MatchString(content) || strictCRAPThresholdPattern.MatchString(content)
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
	return yamlSequenceContainsEnabled(enable, disable, "cyclop", "gocognit", "gocyclo")
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

func yamlSequenceContainsEnabled(enable *yaml.Node, disable *yaml.Node, values ...string) bool {
	for _, value := range values {
		if yamlSequenceContains(enable, value) && !yamlSequenceContains(disable, value) {
			return true
		}
	}
	return false
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

func commandSections(file repo.File) []string {
	if isWorkflowFilePath(file.Path) && strings.Contains(file.Content, workflowStepBoundary) {
		return splitNonEmpty(file.Content, workflowStepBoundary)
	}
	if isWorkflowFilePath(file.Path) {
		return workflowStepBlocks(file.Content)
	}
	return []string{file.Content}
}

func splitNonEmpty(content string, separator string) []string {
	parts := strings.Split(content, separator)
	sections := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			sections = append(sections, part)
		}
	}
	return sections
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
	goCommandPattern               = regexp.MustCompile(`(?m)\bgo\s+(test|vet|build|run|tool|mod)\b`)
	goTestAllPackagesPattern       = regexp.MustCompile(`(?m)\bgo\s+test\b[^\n#;&|]*\./\.\.`)
	goVetAllPackagesPattern        = regexp.MustCompile(`(?m)\bgo\s+vet\b[^\n#;&|]*\./\.\.`)
	golangCILintRunPattern         = regexp.MustCompile(`(?m)(?:^|[[:space:];&|])golangci-lint\s+run(?:[[:space:];&|]|$)|\bgo\s+run\b[^\n#;&|]*github\.com/golangci/golangci-lint(?:/v[0-9]+)?/cmd/golangci-lint[^\n#;&|]*\srun(?:[[:space:];&|]|$)`)
	coverageThresholdPattern       = regexp.MustCompile(`(?im)\b(total|cover|coverage|minimum|threshold|required)\b[^\n]*(>=|<=|-ge\b|-le\b|-gt\b|-lt\b)|(?:>=|<=|-ge\b|-le\b|-gt\b|-lt\b)[^\n]*\b(total|cover|coverage|minimum|threshold|required)\b`)
	crapThresholdPattern           = regexp.MustCompile(`(?im)\b(crap|maximum|minimum|threshold|required|score)\b[^\n]*(>=|<=|-ge\b|-le\b|-gt\b|-lt\b)|(?:>=|<=|-ge\b|-le\b|-gt\b|-lt\b)[^\n]*\b(crap|maximum|minimum|threshold|required|score)\b`)
	strictCoverageThresholdPattern = regexp.MustCompile(`(?im)\b(total|minimum|threshold|required)\b[^\n]*(>|<)[^\n]*(\b(total|minimum|threshold|required)\b|[0-9]+(?:\.[0-9]+)?)|([0-9]+(?:\.[0-9]+)?|\b(total|minimum|threshold|required)\b)[^\n]*(>|<)[^\n]*\b(total|minimum|threshold|required)\b`)
	strictCRAPThresholdPattern     = regexp.MustCompile(`(?im)\b(score|maximum|minimum|threshold|required)\b[^\n]*(>|<)[^\n]*(\b(score|maximum|minimum|threshold|required)\b|[0-9]+(?:\.[0-9]+)?)|([0-9]+(?:\.[0-9]+)?|\b(score|maximum|minimum|threshold|required)\b)[^\n]*(>|<)[^\n]*\b(score|maximum|minimum|threshold|required)\b`)
)

const workflowStepBoundary = "\nSLOPHAMMER_WORKFLOW_STEP_BOUNDARY\n"
