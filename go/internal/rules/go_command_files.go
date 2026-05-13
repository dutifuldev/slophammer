package rules

import (
	"strings"

	"github.com/dutifuldev/slophammer/go/internal/repo"
)

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
	if isWorkflowFilePath(file.Path) {
		return workflowCommandSections(file.Content)
	}
	return []string{file.Content}
}

func workflowCommandSections(content string) []string {
	blocks := workflowStepBlocks(content)
	if strings.Contains(content, workflowStepBoundary) {
		blocks = splitNonEmpty(content, workflowStepBoundary)
	}
	sections := make([]string, 0, len(blocks))
	for _, block := range blocks {
		runContent := workflowRunContent(block)
		if strings.TrimSpace(runContent) != "" {
			sections = append(sections, runContent)
		}
	}
	return sections
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
		if indicator == '-' || indicator == '+' || (indicator >= '1' && indicator <= '9') {
			continue
		}
		return false, false
	}
	return folded, true
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
