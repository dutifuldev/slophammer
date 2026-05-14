package rules

import (
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/dutifuldev/slophammer/go/internal/repo"
)

func goProjectRoots(snapshot repo.Snapshot) []string {
	rootsByPath := goModuleRootSet(snapshot)
	if len(rootsByPath) == 0 && hasUnscopedGoSignal(snapshot) {
		rootsByPath[""] = struct{}{}
	}
	if len(rootsByPath) > 0 && hasGoSourceOutsideModuleRoots(snapshot, rootsByPath) {
		rootsByPath[""] = struct{}{}
	}
	return sortedRootSet(rootsByPath)
}

func goModuleRootSet(snapshot repo.Snapshot) map[string]struct{} {
	rootsByPath := map[string]struct{}{}
	for filePath := range snapshot.Files {
		if !isEmbeddedFixturePath(filePath) && strings.EqualFold(path.Base(filePath), "go.mod") {
			rootsByPath[goModuleRootPath(filePath)] = struct{}{}
		}
	}
	return rootsByPath
}

func goModuleRootPath(filePath string) string {
	root := path.Dir(filePath)
	if root == "." {
		return ""
	}
	return root
}

func sortedRootSet(rootsByPath map[string]struct{}) []string {
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
	return hasRunnableCommandLine(goProjectSnapshot(snapshot, "", []string{""}), lineHasGoCommandSignal)
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
	return scopedNestedGoProjectFile(filePath, file, root, roots)
}

func scopedNestedGoProjectFile(filePath string, file repo.File, root string, roots []string) (repo.File, bool) {
	if isRepoRootSlophammerConfigFile(filePath) {
		return file, true
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
			if lineHasCDCommand(line) {
				inRootBlock = false
				continue
			}
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
		scopedBlock, ok := scopedWorkflowStepBlock(block, root, roots)
		if ok {
			kept = append(kept, scopedBlock)
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

func scopedWorkflowStepBlock(content, root string, roots []string) (string, bool) {
	if !workflowStepAppliesToRoot(content, root, roots) {
		return "", false
	}
	if root == "" || !workflowMentionsOtherGoRoot(content, root, roots) {
		return content, true
	}
	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines))
	inRootBlock := false
	for _, line := range lines {
		keep, active := scopedWorkflowStepLine(line, root, roots, inRootBlock)
		inRootBlock = active
		if keep {
			kept = append(kept, line)
			continue
		}
		runLine, ok := workflowRunLine(strings.TrimSpace(line))
		if _, block := workflowRunBlockScalar(runLine); ok && block {
			kept = append(kept, line)
		}
	}
	scoped := strings.Join(kept, "\n")
	return scoped, strings.TrimSpace(scoped) != ""
}

func scopedWorkflowStepLine(line string, root string, roots []string, inRootBlock bool) (bool, bool) {
	if workflowMentionsOtherGoRoot(line, root, roots) || (inRootBlock && lineHasCDCommand(line)) {
		return false, false
	}
	if workflowMentionsGoRoot(line, root, roots) {
		return true, true
	}
	return inRootBlock, inRootBlock
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
	if !workflowBlockHasRun(lines) {
		lines = withoutWorkflowRunDefaults(lines)
	}
	if len(lines) == 0 {
		return blocks
	}
	return append(blocks, strings.Join(lines, "\n"))
}

func workflowBlockHasRun(lines []string) bool {
	for _, line := range lines {
		if _, ok := workflowRunLine(strings.TrimSpace(line)); ok {
			return true
		}
	}
	return false
}

func withoutWorkflowRunDefaults(lines []string) []string {
	for i, line := range lines {
		if isWorkflowStepStart(line) {
			return lines[i:]
		}
	}
	return lines
}

func workflowMentionsOtherGoRoot(content, root string, roots []string) bool {
	for _, otherRoot := range roots {
		if otherRoot != "" && otherRoot != root && workflowMentionsGoRoot(content, otherRoot, roots) {
			return true
		}
	}
	return false
}

func lineHasCDCommand(line string) bool {
	tokens := strings.Fields(line)
	for i, token := range tokens {
		if cleanCommandToken(token) == "cd" && isCommandToken(tokens, i) {
			return true
		}
	}
	return false
}

func workflowMentionsGoRoot(content, root string, roots []string) bool {
	normalized := strings.ReplaceAll(content, "\\", "/")
	return workflowReferencesRootExact(normalized, root, roots, workingDirectoryPattern(root)) ||
		workflowReferencesRootExact(normalized, root, roots, cdRootPattern(root)) ||
		workflowReferencesRootExact(normalized, root, roots, goCFlagRootPattern(root)) ||
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

func goCFlagRootPattern(root string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)(?:^|[[:space:];&|])go\s+-C(?:=|\s+)['"]?(?:\./)?` + regexp.QuoteMeta(root) + `['"]?(?:[[:space:];&|]|$)`)
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

func isRepoRootSlophammerConfigFile(filePath string) bool {
	return filePath == "slophammer.yml" || filePath == "slophammer.yaml"
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
