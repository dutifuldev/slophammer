package rules

import (
	"path"
	"strings"

	"github.com/dutifuldev/slophammer/go/internal/gotools"
	"github.com/dutifuldev/slophammer/go/internal/repo"
)

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
		if isGoToolPackageToken(token, gotools.Mutate4Go.Package) && isGoRunPackage(tokens, i) {
			if hasMutationTargetAfter(tokens, i) {
				return true
			}
			continue
		}
		if isToolBinaryToken(token, gotools.Mutate4Go.Binary) && isCommandToken(tokens, i) {
			return hasMutationTargetAfter(tokens, i)
		}
	}
	return false
}

func contentHasGoToolCommand(content string, tool gotools.Tool) bool {
	for _, tokens := range commandTokensByLine(content) {
		if lineHasGoToolCommand(tokens, tool) {
			return true
		}
	}
	return false
}

func lineHasGoToolCommand(tokens []string, tool gotools.Tool) bool {
	for i, token := range tokens {
		token = cleanCommandToken(token)
		if isGoToolPackageToken(token, tool.Package) {
			if isGoRunPackage(tokens, i) {
				return true
			}
			continue
		}
		if isToolBinaryToken(token, tool.Binary) {
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

func fileHasConfigBackedSlophammerGoCommand(file repo.File, subcommand string) bool {
	if isWorkflowFilePath(file.Path) {
		for _, block := range workflowCommandBlocks(file.Content) {
			needsParentRoot := workflowBlockNeedsParentConfigPath(block)
			if contentHasConfigBackedSlophammerGoCommand(workflowRunContent(block), subcommand, needsParentRoot) {
				return true
			}
		}
		return false
	}
	for _, content := range commandSections(file) {
		if contentHasConfigBackedSlophammerGoCommand(content, subcommand, false) {
			return true
		}
	}
	return false
}

func contentHasConfigBackedSlophammerGoCommand(content string, subcommand string, needsParentRoot bool) bool {
	matcher := configBackedSlophammerGoCommandMatcher{
		subcommand:      subcommand,
		needsParentRoot: needsParentRoot,
	}
	return contentHasCommandLine(content, matcher.match)
}

type configBackedSlophammerGoCommandMatcher struct {
	subcommand      string
	needsParentRoot bool
}

func (m configBackedSlophammerGoCommandMatcher) match(tokens []string) bool {
	return lineHasConfigBackedSlophammerGoCommand(tokens, m.subcommand, m.needsParentRoot)
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
		if isSlophammerGoSubcommand(tokens, i, subcommand) &&
			lineHasRequiredFlag(tokens[i+3:], requiredFlag) {
			return true
		}
	}
	return false
}

func lineHasConfigBackedSlophammerGoCommand(tokens []string, subcommand string, needsParentRoot bool) bool {
	for i := 0; i+2 < len(tokens); i++ {
		if !isSlophammerGoSubcommand(tokens, i, subcommand) {
			continue
		}
		if lineHasConfigRootArgument(tokens[:i], tokens[i+3:], needsParentRoot) {
			return true
		}
	}
	return false
}

func isSlophammerGoSubcommand(tokens []string, commandIndex int, subcommand string) bool {
	if !isSlophammerCommandToken(cleanCommandToken(tokens[commandIndex])) {
		return false
	}
	if !isCommandToken(tokens, commandIndex) && !isGoRunPackage(tokens, commandIndex) {
		return false
	}
	return cleanCommandToken(tokens[commandIndex+1]) == "go" &&
		cleanCommandToken(tokens[commandIndex+2]) == subcommand
}

func lineHasConfigRootArgument(prefix []string, tokens []string, needsParentRoot bool) bool {
	if lineHasPriorCDCommand(prefix) {
		needsParentRoot = true
	}
	for i := 0; i < len(tokens); i++ {
		token := cleanCommandToken(tokens[i])
		if token == "" {
			continue
		}
		if isShellSeparator(token) {
			return false
		}
		if strings.HasPrefix(token, "-") {
			if slophammerGoFlagNeedsValue(token) && !strings.Contains(token, "=") {
				i++
			}
			continue
		}
		return pathIsConfigRootArgument(token, needsParentRoot)
	}
	return false
}

func lineHasPriorCDCommand(tokens []string) bool {
	for i, token := range tokens {
		if cleanCommandToken(token) == "cd" && isCommandToken(tokens, i) {
			return true
		}
	}
	return false
}

func slophammerGoFlagNeedsValue(token string) bool {
	flag, _, _ := strings.Cut(token, "=")
	switch flag {
	case "--max-candidates", "--max-score", "--target":
		return true
	default:
		return false
	}
}

func pathIsConfigRootArgument(token string, needsParentRoot bool) bool {
	cleaned := path.Clean(strings.ReplaceAll(token, "\\", "/"))
	if cleaned == ".." {
		return true
	}
	return !needsParentRoot && cleaned == "."
}

func workflowBlockNeedsParentConfigPath(block string) bool {
	workingDirectory := ""
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "working-directory:") {
			continue
		}
		workingDirectory = strings.TrimSpace(strings.TrimPrefix(trimmed, "working-directory:"))
	}
	if workingDirectory == "" {
		return false
	}
	workingDirectory = strings.Trim(cleanCommandToken(workingDirectory), "/")
	return workingDirectory != "" && workingDirectory != "."
}

func commandTokensByLine(content string) [][]string {
	lines := strings.Split(content, "\n")
	tokenLines := make([][]string, 0, len(lines))
	for _, line := range lines {
		tokens := commandTokens(line)
		if len(tokens) > 0 {
			tokenLines = append(tokenLines, tokens)
		}
	}
	return tokenLines
}

func commandTokens(line string) []string {
	var normalized strings.Builder
	var quote rune
	for _, r := range line {
		switch {
		case quote == 0 && (r == '\'' || r == '"'):
			quote = r
		case quote == r:
			quote = 0
		case quote == 0 && r == ';':
			normalized.WriteString(" ; ")
			continue
		}
		normalized.WriteRune(r)
	}
	return strings.Fields(strings.ReplaceAll(normalized.String(), "$(", " ; "))
}

func isGoRunPackage(tokens []string, packageIndex int) bool {
	for goIndex := 0; goIndex < packageIndex; goIndex++ {
		if cleanCommandToken(tokens[goIndex]) != "go" || !isCommandToken(tokens, goIndex) {
			continue
		}
		commandIndex := goCommandIndex(tokens, goIndex+1)
		if commandIndex == -1 || cleanCommandToken(tokens[commandIndex]) != "run" {
			continue
		}
		if goRunPackageIndex(tokens, commandIndex+1) == packageIndex {
			return true
		}
	}
	return false
}

func goCommandIndex(tokens []string, start int) int {
	return goArgumentIndex(tokens, start, goGlobalFlagNeedsValue)
}

func goGlobalFlagNeedsValue(token string) bool {
	flag, _, _ := strings.Cut(token, "=")
	return flag == "-C"
}

func goRunPackageIndex(tokens []string, start int) int {
	return goArgumentIndex(tokens, start, goRunFlagNeedsValue)
}

func goArgumentIndex(tokens []string, start int, flagNeedsValue func(string) bool) int {
	for i := start; i < len(tokens); i++ {
		token := cleanCommandToken(tokens[i])
		if token == "" {
			continue
		}
		if isShellSeparator(token) {
			return -1
		}
		if strings.HasPrefix(token, "-") {
			if flagNeedsValue(token) && !strings.Contains(token, "=") {
				i++
			}
			continue
		}
		return i
	}
	return -1
}

func goRunFlagNeedsValue(token string) bool {
	flag, _, _ := strings.Cut(token, "=")
	switch flag {
	case "-asmflags", "-exec", "-gcflags", "-ldflags", "-mod", "-overlay", "-p", "-pkgdir", "-tags", "-toolexec":
		return true
	default:
		return false
	}
}

func isCommandToken(tokens []string, commandIndex int) bool {
	if commandIndex == 0 {
		return true
	}
	previous := cleanCommandToken(tokens[commandIndex-1])
	return isShellSeparator(previous) || hasCommandPrefix(tokens, commandIndex)
}

func hasCommandPrefix(tokens []string, commandIndex int) bool {
	prefixIndex := commandIndex - 1
	for prefixIndex >= 0 && isEnvAssignmentToken(cleanCommandToken(tokens[prefixIndex])) {
		prefixIndex--
	}
	if prefixIndex < 0 {
		return true
	}
	prefix := cleanCommandToken(tokens[prefixIndex])
	if isShellSeparator(prefix) {
		return true
	}
	return prefix == "env" && isCommandToken(tokens, prefixIndex)
}

func isEnvAssignmentToken(token string) bool {
	name, _, ok := strings.Cut(token, "=")
	return ok && shellNamePattern.MatchString(name)
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
	for i, token := range tokens {
		if isShellSeparator(cleanCommandToken(token)) {
			return false
		}
		if cleanCommandToken(token) == requiredFlag {
			return hasFlagValue(tokens, i)
		}
	}
	return false
}

func hasFlagValue(tokens []string, flagIndex int) bool {
	if flagIndex+1 >= len(tokens) {
		return false
	}
	value := cleanCommandToken(tokens[flagIndex+1])
	return value != "" && !strings.HasPrefix(value, "-") && !isShellSeparator(value)
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

func cleanCommandToken(token string) string {
	return strings.Trim(token, " \t\r\n'\"")
}

func isShellSeparator(token string) bool {
	switch token {
	case ";", "|", "||", "&", "&&":
		return true
	default:
		return false
	}
}
