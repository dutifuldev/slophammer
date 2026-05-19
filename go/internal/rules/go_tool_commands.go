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
			configRootPath := workflowBlockConfigRootPath(block)
			if contentHasConfigBackedSlophammerGoCommand(workflowRunContent(block), subcommand, configRootPath) {
				return true
			}
		}
		return false
	}
	for _, content := range commandSections(file) {
		if contentHasConfigBackedSlophammerGoCommand(content, subcommand, ".") {
			return true
		}
	}
	return false
}

func fileHasConfigBackedSlophammerGoCheckExecuteCommand(file repo.File) bool {
	if isWorkflowFilePath(file.Path) {
		for _, block := range workflowCommandBlocks(file.Content) {
			configRootPath := workflowBlockConfigRootPath(block)
			if contentHasConfigBackedSlophammerGoCheckExecuteCommand(workflowRunContent(block), configRootPath) {
				return true
			}
		}
		return false
	}
	for _, content := range commandSections(file) {
		if contentHasConfigBackedSlophammerGoCheckExecuteCommand(content, ".") {
			return true
		}
	}
	return false
}

func contentHasConfigBackedSlophammerGoCommand(content string, subcommand string, configRootPath string) bool {
	matcher := configBackedSlophammerGoCommandMatcher{
		subcommand:     subcommand,
		configRootPath: configRootPath,
	}
	return contentHasCommandLine(content, matcher.match)
}

func contentHasConfigBackedSlophammerGoCheckExecuteCommand(content string, configRootPath string) bool {
	return contentHasCommandLine(content, func(tokens []string) bool {
		return lineHasConfigBackedSlophammerGoCheckExecuteCommand(tokens, configRootPath)
	})
}

type configBackedSlophammerGoCommandMatcher struct {
	subcommand     string
	configRootPath string
}

func (m configBackedSlophammerGoCommandMatcher) match(tokens []string) bool {
	return lineHasConfigBackedSlophammerGoCommand(tokens, m.subcommand, m.configRootPath)
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
	for i := 0; i < len(tokens); i++ {
		argsStart, ok := slophammerGoCommandArgsStart(tokens, i, subcommand)
		if ok && lineHasRequiredFlag(tokens[argsStart:], requiredFlag) {
			return true
		}
	}
	return false
}

func lineHasConfigBackedSlophammerGoCommand(tokens []string, subcommand string, configRootPath string) bool {
	for i := 0; i < len(tokens); i++ {
		argsStart, ok := slophammerGoCommandArgsStart(tokens, i, subcommand)
		if !ok {
			continue
		}
		if lineHasConfigRootArgument(tokens[:i], tokens[argsStart:], configRootPath) {
			return true
		}
	}
	return false
}

func lineHasConfigBackedSlophammerGoCheckExecuteCommand(tokens []string, configRootPath string) bool {
	for i := 0; i < len(tokens); i++ {
		argsStart, ok := slophammerGoCommandArgsStart(tokens, i, "check")
		if !ok {
			continue
		}
		args := tokens[argsStart:]
		if !lineHasBooleanFlag(args, "--execute") {
			continue
		}
		if lineHasConfigRootArgument(tokens[:i], args, configRootPath) {
			return true
		}
	}
	return false
}

func slophammerGoCommandArgsStart(tokens []string, commandIndex int, subcommand string) (int, bool) {
	token := cleanCommandToken(tokens[commandIndex])
	if !isSlophammerCommandToken(token) {
		return 0, false
	}
	if !isCommandToken(tokens, commandIndex) && !isGoRunPackage(tokens, commandIndex) {
		return 0, false
	}
	if argsStart, ok := directSlophammerGoArgsStart(tokens, commandIndex, subcommand); ok {
		return argsStart, true
	}
	return legacySlophammerGoArgsStart(tokens, commandIndex, subcommand)
}

func directSlophammerGoArgsStart(tokens []string, commandIndex int, subcommand string) (int, bool) {
	if commandIndex+1 >= len(tokens) || cleanCommandToken(tokens[commandIndex+1]) != subcommand {
		return 0, false
	}
	return commandIndex + 2, true
}

func legacySlophammerGoArgsStart(tokens []string, commandIndex int, subcommand string) (int, bool) {
	if commandIndex+2 >= len(tokens) ||
		cleanCommandToken(tokens[commandIndex+1]) != "go" ||
		cleanCommandToken(tokens[commandIndex+2]) != subcommand {
		return 0, false
	}
	return commandIndex + 3, true
}

func lineHasConfigRootArgument(prefix []string, tokens []string, configRootPath string) bool {
	if workingDirectory, ok := priorCDWorkingDirectory(prefix); ok {
		configRootPath = configRootPathForWorkingDirectory(workingDirectory)
	}
	if token, ok := firstSlophammerGoPathArgument(tokens); ok {
		return pathIsConfigRootArgument(token, configRootPath)
	}
	return path.Clean(configRootPath) == "."
}

func firstSlophammerGoPathArgument(tokens []string) (string, bool) {
	for i := 0; i < len(tokens); i++ {
		token := cleanCommandToken(tokens[i])
		switch {
		case token == "":
			continue
		case isShellSeparator(token):
			return "", false
		case strings.HasPrefix(token, "-"):
			if slophammerGoFlagConsumesNext(token) {
				i++
			}
		default:
			return token, true
		}
	}
	return "", false
}

func slophammerGoFlagConsumesNext(token string) bool {
	return slophammerGoFlagNeedsValue(token) && !strings.Contains(token, "=")
}

func priorCDWorkingDirectory(tokens []string) (string, bool) {
	workingDirectory := ""
	for i := 0; i+1 < len(tokens); i++ {
		if cleanCommandToken(tokens[i]) != "cd" || !isCommandToken(tokens, i) {
			continue
		}
		next := cleanCommandToken(tokens[i+1])
		if next == "" || strings.HasPrefix(next, "-") || isShellSeparator(next) {
			continue
		}
		workingDirectory = next
	}
	return workingDirectory, workingDirectory != ""
}

func slophammerGoFlagNeedsValue(token string) bool {
	flag, _, _ := strings.Cut(token, "=")
	switch flag {
	case "--coverage-profile", "--format", "--max-candidates", "--max-score", "--profile", "--target":
		return true
	default:
		return false
	}
}

func pathIsConfigRootArgument(token string, configRootPath string) bool {
	cleaned := path.Clean(strings.ReplaceAll(token, "\\", "/"))
	return cleaned == path.Clean(configRootPath)
}

func workflowBlockConfigRootPath(block string) string {
	workingDirectory := ""
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "working-directory:") {
			continue
		}
		workingDirectory = strings.TrimSpace(strings.TrimPrefix(trimmed, "working-directory:"))
	}
	if workingDirectory == "" {
		return "."
	}
	return configRootPathForWorkingDirectory(workingDirectory)
}

func configRootPathForWorkingDirectory(workingDirectory string) string {
	cleaned := path.Clean(strings.ReplaceAll(cleanCommandToken(workingDirectory), "\\", "/"))
	if cleaned == "." || cleaned == "/" {
		return "."
	}
	parts := strings.Split(strings.Trim(cleaned, "/"), "/")
	parents := make([]string, 0, len(parts))
	for range parts {
		parents = append(parents, "..")
	}
	return path.Join(parents...)
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

func lineHasBooleanFlag(tokens []string, flag string) bool {
	for _, token := range tokens {
		cleaned := cleanCommandToken(token)
		if isShellSeparator(cleaned) {
			return false
		}
		if cleaned == flag {
			return true
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
	base := path.Base(stripGoModuleVersion(token))
	switch base {
	case "slophammer", "slophammer.exe", "slophammer-go", "slophammer-go.exe":
		return true
	default:
		return false
	}
}

func stripGoModuleVersion(token string) string {
	if before, _, ok := strings.Cut(token, "@"); ok {
		return before
	}
	return token
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
