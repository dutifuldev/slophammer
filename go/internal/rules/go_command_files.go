package rules

import (
	"strings"

	"github.com/dutifuldev/slophammer/go/internal/repo"
)

func commandFiles(snapshot repo.Snapshot) []repo.File {
	return preparedCommandFiles(commandFileMap(snapshot))
}

func commandFileMap(snapshot repo.Snapshot) map[string]repo.File {
	filesByPath := map[string]repo.File{}
	evidence := addBindingWorkflowFiles(snapshot, filesByPath)
	addReachableCommandFiles(snapshot, filesByPath, evidence)
	return filesByPath
}

// addBindingWorkflowFiles adds the binding command content of each workflow
// and returns the combined text used as the reachability root.
func addBindingWorkflowFiles(snapshot repo.Snapshot, filesByPath map[string]repo.File) string {
	var evidence strings.Builder
	for _, file := range snapshot.WorkflowFiles() {
		content := bindingWorkflowCommandText(file.Content)
		if strings.TrimSpace(content) == "" {
			continue
		}
		filesByPath[file.Path] = repo.File{Path: file.Path, Content: content}
		evidence.WriteString(content)
		evidence.WriteString("\n")
	}
	return evidence.String()
}

// addReachableCommandFiles credits scripts, Makefiles, Taskfiles, and
// justfiles only when binding workflow evidence invokes them, following
// script-to-script references one level deep.
func addReachableCommandFiles(snapshot repo.Snapshot, filesByPath map[string]repo.File, evidence string) {
	candidates := commandFileCandidates(snapshot)
	firstHop := reachableCandidates(candidates, evidence)
	extended := evidence + joinedContents(firstHop)
	for _, file := range reachableCandidates(candidates, extended) {
		filesByPath[file.Path] = file
	}
}

func commandFileCandidates(snapshot repo.Snapshot) map[string]repo.File {
	candidates := map[string]repo.File{}
	for _, file := range snapshot.FilesNamedFold("Makefile", "Taskfile.yml", "Taskfile.yaml", "justfile") {
		candidates[file.Path] = file
	}
	for _, file := range snapshot.FilesUnder("scripts") {
		candidates[file.Path] = file
	}
	for _, file := range snapshot.FilesUnder("go/scripts") {
		candidates[file.Path] = file
	}
	for path, file := range snapshot.Files {
		if isScriptPath(path) {
			candidates[file.Path] = file
		}
	}
	return candidates
}

func reachableCandidates(candidates map[string]repo.File, evidence string) []repo.File {
	reachable := make([]repo.File, 0, len(candidates))
	for _, file := range candidates {
		if commandFileReachable(file.Path, evidence) {
			reachable = append(reachable, file)
		}
	}
	return reachable
}

func commandFileReachable(filePath string, evidence string) bool {
	reference, ok := runnerReference(filePath)
	if !ok {
		reference = pathBaseName(filePath)
	}
	return containsCommandWord(evidence, reference)
}

// runnerReference maps runner-driven command files to the command that
// invokes them, since workflows reference the runner rather than the file.
func runnerReference(filePath string) (string, bool) {
	switch strings.ToLower(pathBaseName(filePath)) {
	case "makefile":
		return "make", true
	case "taskfile.yml", "taskfile.yaml":
		return "task", true
	case "justfile":
		return "just", true
	default:
		return "", false
	}
}

func pathBaseName(filePath string) string {
	normalized := strings.ReplaceAll(filePath, "\\", "/")
	if index := strings.LastIndex(normalized, "/"); index >= 0 {
		return normalized[index+1:]
	}
	return normalized
}

func containsCommandWord(evidence string, word string) bool {
	for index := strings.Index(evidence, word); index >= 0; {
		if commandWordBoundary(evidence, index, len(word)) {
			return true
		}
		next := strings.Index(evidence[index+1:], word)
		if next < 0 {
			return false
		}
		index += 1 + next
	}
	return false
}

func commandWordBoundary(evidence string, index int, length int) bool {
	if index > 0 && isCommandWordByte(evidence[index-1]) {
		return false
	}
	end := index + length
	return end >= len(evidence) || !isCommandWordByte(evidence[end])
}

func isCommandWordByte(value byte) bool {
	switch {
	case value >= 'a' && value <= 'z', value >= 'A' && value <= 'Z', value >= '0' && value <= '9':
		return true
	case value == '_', value == '-':
		return true
	default:
		return false
	}
}

func joinedContents(files []repo.File) string {
	var joined strings.Builder
	for _, file := range files {
		joined.WriteString(file.Content)
		joined.WriteString("\n")
	}
	return joined.String()
}

func preparedCommandFiles(filesByPath map[string]repo.File) []repo.File {
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

// commandSections returns a file's command content. Workflow files are
// already reduced to their binding run scripts during preparation.
func commandSections(file repo.File) []string {
	return []string{file.Content}
}

func workflowCommandSections(content string) []string {
	blocks := workflowCommandBlocks(content)
	sections := make([]string, 0, len(blocks))
	for _, block := range blocks {
		runContent := workflowRunContent(block)
		if strings.TrimSpace(runContent) != "" {
			sections = append(sections, runContent)
		}
	}
	return sections
}

func workflowCommandBlocks(content string) []string {
	if strings.Contains(content, workflowStepBoundary) {
		return splitNonEmpty(content, workflowStepBoundary)
	}
	return workflowStepBlocks(content)
}

func workflowRunContent(block string) string {
	scan := workflowRunScan{}
	for _, line := range strings.Split(block, "\n") {
		scan.visitLine(line)
	}
	return scan.content()
}

type workflowRunScan struct {
	kept          []string
	foldedLines   []string
	inRunBlock    bool
	inFoldedBlock bool
	runIndent     int
	contentIndent int
}

func (s *workflowRunScan) visitLine(line string) {
	trimmed := strings.TrimSpace(line)
	if s.inRunBlock {
		if s.recordRunBlockLine(line, trimmed) {
			return
		}
		s.endRunBlock()
	}
	runLine, ok := workflowRunLine(trimmed)
	if ok {
		s.startRun(line, runLine)
		return
	}
}

func (s *workflowRunScan) startRun(line, runLine string) {
	s.flushFolded()
	folded, block := workflowRunBlockScalar(runLine)
	if block {
		s.inRunBlock = true
		s.inFoldedBlock = folded
		s.runIndent = leadingSpaceCount(line)
		s.contentIndent = 0
		return
	}
	s.kept = append(s.kept, runLine)
	s.endRunBlock()
}

func workflowRunBlockScalar(value string) (folded bool, ok bool) {
	if len(value) == 0 {
		return false, false
	}
	switch value[0] {
	case '|':
		folded = false
	case '>':
		folded = true
	default:
		return false, false
	}
	for _, indicator := range value[1:] {
		if !isWorkflowBlockScalarIndicator(indicator) {
			return false, false
		}
	}
	return folded, true
}

func isWorkflowBlockScalarIndicator(indicator rune) bool {
	return indicator == '-' || indicator == '+' || (indicator >= '1' && indicator <= '9')
}

func (s *workflowRunScan) recordRunBlockLine(line, trimmed string) bool {
	if trimmed == "" {
		return true
	}
	indent := leadingSpaceCount(line)
	if s.contentIndent == 0 {
		if indent <= s.runIndent {
			return false
		}
		s.contentIndent = indent
	}
	if indent < s.contentIndent {
		return false
	}
	s.recordRunLine(trimmed)
	return true
}

func (s *workflowRunScan) recordRunLine(line string) {
	if s.inFoldedBlock {
		if line != "" {
			s.foldedLines = append(s.foldedLines, line)
		}
		return
	}
	s.kept = append(s.kept, line)
}

func (s *workflowRunScan) endRunBlock() {
	s.inRunBlock = false
	s.inFoldedBlock = false
	s.runIndent = 0
	s.contentIndent = 0
}

func (s *workflowRunScan) content() string {
	s.flushFolded()
	return strings.Join(s.kept, "\n")
}

func (s *workflowRunScan) flushFolded() {
	if len(s.foldedLines) == 0 {
		return
	}
	s.kept = append(s.kept, strings.TrimSpace(strings.Join(s.foldedLines, " ")))
	s.foldedLines = s.foldedLines[:0]
}

func leadingSpaceCount(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

func workflowRunLine(trimmed string) (string, bool) {
	switch {
	case strings.HasPrefix(trimmed, "- run:"):
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "- run:")), true
	case strings.HasPrefix(trimmed, "run:"):
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "run:")), true
	default:
		return "", false
	}
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

func hasRunnableCommandLine(snapshot repo.Snapshot, match func([]string) bool) bool {
	for _, file := range commandFiles(snapshot) {
		for _, content := range commandSections(file) {
			if contentHasCommandLine(content, match) {
				return true
			}
		}
	}
	return false
}

func lineHasGoSubcommandAllPackages(tokens []string, subcommand string) bool {
	for i := 0; i < len(tokens); i++ {
		if cleanCommandToken(tokens[i]) != "go" || !isCommandToken(tokens, i) {
			continue
		}
		commandIndex := goCommandIndex(tokens, i+1)
		if commandIndex == -1 || cleanCommandToken(tokens[commandIndex]) != subcommand {
			continue
		}
		if hasArgumentBeforeSeparator(tokens[commandIndex+1:], "./...") {
			return true
		}
	}
	return false
}

func lineHasGolangCICommand(tokens []string) bool {
	for i, token := range tokens {
		token = cleanCommandToken(token)
		if isToolBinaryToken(token, "golangci-lint") && isCommandToken(tokens, i) {
			return hasArgumentBeforeSeparator(tokens[i+1:], "run")
		}
		if isGoToolPackageToken(token, "github.com/golangci/golangci-lint") && isGoRunPackage(tokens, i) {
			return hasArgumentBeforeSeparator(tokens[i+1:], "run")
		}
	}
	return false
}

func lineHasGoToolCoverCommand(tokens []string) bool {
	for i := 0; i < len(tokens); i++ {
		if cleanCommandToken(tokens[i]) != "go" || !isCommandToken(tokens, i) {
			continue
		}
		commandIndex := goCommandIndex(tokens, i+1)
		if commandIndex == -1 || cleanCommandToken(tokens[commandIndex]) != "tool" {
			continue
		}
		if hasArgumentBeforeSeparator(tokens[commandIndex+1:], "cover") {
			return true
		}
	}
	return false
}

func lineHasGoTestCoverageProfileCommand(tokens []string) bool {
	for i := 0; i < len(tokens); i++ {
		if cleanCommandToken(tokens[i]) != "go" || !isCommandToken(tokens, i) {
			continue
		}
		commandIndex := goCommandIndex(tokens, i+1)
		if commandIndex == -1 || cleanCommandToken(tokens[commandIndex]) != "test" {
			continue
		}
		if hasCoverageProfileFlag(tokens[commandIndex+1:]) {
			return true
		}
	}
	return false
}

func lineHasGoCommandSignal(tokens []string) bool {
	for i := 0; i < len(tokens); i++ {
		if cleanCommandToken(tokens[i]) != "go" || !isCommandToken(tokens, i) {
			continue
		}
		commandIndex := goCommandIndex(tokens, i+1)
		if commandIndex == -1 {
			continue
		}
		switch cleanCommandToken(tokens[commandIndex]) {
		case "build", "mod", "run", "test", "tool", "vet":
			return true
		}
	}
	return false
}

func hasCoverageProfileFlag(tokens []string) bool {
	for i, token := range tokens {
		token = cleanCommandToken(token)
		if isShellSeparator(token) {
			return false
		}
		if strings.HasPrefix(token, "-coverprofile=") && strings.TrimPrefix(token, "-coverprofile=") != "" {
			return true
		}
		if token == "-coverprofile" {
			return hasFlagValue(tokens, i)
		}
	}
	return false
}

func hasArgumentBeforeSeparator(tokens []string, argument string) bool {
	for _, token := range tokens {
		token = cleanCommandToken(token)
		if isShellSeparator(token) {
			return false
		}
		if token == argument {
			return true
		}
	}
	return false
}
