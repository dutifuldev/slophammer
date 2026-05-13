package rules

import (
	"path"
	"strings"

	"github.com/dutifuldev/slophammer/go/internal/gotools"
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
