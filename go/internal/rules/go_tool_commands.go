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
	if scanOnlyMutationLine(tokens) {
		return false
	}
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

// A scan-only mutation command counts mutation sites and cannot fail on a
// surviving mutant, so it is not mutation-testing evidence.
func scanOnlyMutationLine(tokens []string) bool {
	for _, token := range tokens {
		if cleanCommandToken(token) == "--scan" {
			return true
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
			if contentHasConfigBackedSlophammerGoCommand(workflowBlockRunContent(block), subcommand, configRootPath) {
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
			if contentHasConfigBackedSlophammerGoCheckExecuteCommand(workflowBlockRunContent(block), configRootPath) {
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
	if subcommand == "mutate" && scanOnlyMutationLine(tokens) {
		return false
	}
	for i := 0; i < len(tokens); i++ {
		argsStart, ok := slophammerGoCommandArgsStart(tokens, i, subcommand)
		if ok && lineHasRequiredFlag(tokens[argsStart:], requiredFlag) {
			return true
		}
	}
	return false
}

func lineHasConfigBackedSlophammerGoCommand(tokens []string, subcommand string, configRootPath string) bool {
	if subcommand == "mutate" && scanOnlyMutationLine(tokens) {
		return false
	}
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

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-06-12T21:41:26+08:00","module_hash":"facbf01a07215eb7683779ee21714e734bd4c1d122522d1fcd5afdee337e44dc","functions":[{"id":"func/contentHasDirectMutate4GoCommand","name":"contentHasDirectMutate4GoCommand","line":11,"end_line":19,"hash":"3c16ce843ddc303b0503cce52341eeb700477077159aaa2850388dbd39557a1d"},{"id":"func/lineHasDirectMutate4GoCommand","name":"lineHasDirectMutate4GoCommand","line":21,"end_line":38,"hash":"1c1589eba2aecc91f8efc67c0941f36501582b0132cefd10d2b3365cc8df4dde"},{"id":"func/scanOnlyMutationLine","name":"scanOnlyMutationLine","line":42,"end_line":49,"hash":"3dd28619c467a3a5ea1e9c45476ac69084a7dff4a8cdd5cfe44d32aaace71bb1"},{"id":"func/contentHasGoToolCommand","name":"contentHasGoToolCommand","line":51,"end_line":58,"hash":"104429191b5008441c044633b923868d5ff295cb8bff2a5cb6ce8f21a269aa71"},{"id":"func/lineHasGoToolCommand","name":"lineHasGoToolCommand","line":60,"end_line":74,"hash":"0b487471731781d84807a80e88a3948b4d5daef5615bdc79827dd558dd261c0b"},{"id":"func/contentHasSlophammerGoCommand","name":"contentHasSlophammerGoCommand","line":76,"end_line":80,"hash":"f8fe9a87d3277c2226f0b42c21138287ebde497faf8a85a2fe0e5232a4dd071b"},{"id":"func/fileHasConfigBackedSlophammerGoCommand","name":"fileHasConfigBackedSlophammerGoCommand","line":82,"end_line":98,"hash":"f264c4493e9b0b87f01dfe7ec67ec4868eb2e726921e1f096022ea372d636e3c"},{"id":"func/fileHasConfigBackedSlophammerGoCheckExecuteCommand","name":"fileHasConfigBackedSlophammerGoCheckExecuteCommand","line":100,"end_line":116,"hash":"8c752202311ebb8521a4e3812d154e8bfcad3654a2756706c3d90374b038991c"},{"id":"func/contentHasConfigBackedSlophammerGoCommand","name":"contentHasConfigBackedSlophammerGoCommand","line":118,"end_line":124,"hash":"e63b1767b66982140fadb9c211c8092d427c6412d53b08d93c8209d8cf66df25"},{"id":"func/contentHasConfigBackedSlophammerGoCheckExecuteCommand","name":"contentHasConfigBackedSlophammerGoCheckExecuteCommand","line":126,"end_line":130,"hash":"3906afe0d879bc15e25cefbb3575262a68a22c4e180725737822b648fe5f0bc7"},{"id":"func/configBackedSlophammerGoCommandMatcher.match","name":"configBackedSlophammerGoCommandMatcher.match","line":137,"end_line":139,"hash":"51a71c93530e312d5527af28d2157d09e378223761430c7a9c12d6c7a19556c2"},{"id":"func/contentHasCommandLine","name":"contentHasCommandLine","line":141,"end_line":148,"hash":"8760eab5fd355e7bd0633a65512a64542c119e887c3bde2a0c3b7ab5e4e8b463"},{"id":"func/lineHasSlophammerGoCommand","name":"lineHasSlophammerGoCommand","line":150,"end_line":161,"hash":"3d0b2fde905b3dcba5222386bda94f9ac6502ba7645624dac4e00cd1d9947c0b"},{"id":"func/lineHasConfigBackedSlophammerGoCommand","name":"lineHasConfigBackedSlophammerGoCommand","line":163,"end_line":177,"hash":"2f7c4e2e2f31edf08bdada4f4d149ef2a0d84db0b99329c5c7bccb35ff9d7a66"},{"id":"func/lineHasConfigBackedSlophammerGoCheckExecuteCommand","name":"lineHasConfigBackedSlophammerGoCheckExecuteCommand","line":179,"end_line":194,"hash":"b66ed019f8ec01e294c546c4e06cd2c068db02279090449dc5c001c4dc9fc999"},{"id":"func/slophammerGoCommandArgsStart","name":"slophammerGoCommandArgsStart","line":196,"end_line":208,"hash":"13d5d03e8cb1e0a2e7f42f0c7a75c46cc040b1ee218a8d6079369ae7bc8f7e16"},{"id":"func/directSlophammerGoArgsStart","name":"directSlophammerGoArgsStart","line":210,"end_line":215,"hash":"202a33c978d699f6365cd9ae04a90eec76f4e2fd0f42f6f0eb03e94464430652"},{"id":"func/legacySlophammerGoArgsStart","name":"legacySlophammerGoArgsStart","line":217,"end_line":224,"hash":"a559b3808f720923de30d24cbd94ef9bf98ffca6c8d52ad6e2121a806023253a"},{"id":"func/lineHasConfigRootArgument","name":"lineHasConfigRootArgument","line":226,"end_line":234,"hash":"724f64529281ebff6e90be515e721026eb254651260a0f35c83dd5937ecad4b5"},{"id":"func/firstSlophammerGoPathArgument","name":"firstSlophammerGoPathArgument","line":236,"end_line":253,"hash":"805f96c1e228fd7f208eedbaa559532e2a708643d7681390bb8974d2a2b1104b"},{"id":"func/slophammerGoFlagConsumesNext","name":"slophammerGoFlagConsumesNext","line":255,"end_line":257,"hash":"e20175d7d16d39877db384f744dca3a7172bdb56e96fe6c5a3e5f3eaee3a9e77"},{"id":"func/priorCDWorkingDirectory","name":"priorCDWorkingDirectory","line":259,"end_line":272,"hash":"0ead31f84d1b2d6a8eea8a00886f4eb30be085e92274fbd72cb130fd4211daf8"},{"id":"func/slophammerGoFlagNeedsValue","name":"slophammerGoFlagNeedsValue","line":274,"end_line":282,"hash":"bffa90a5f4eb5a6cb01696485b23faecdd4cbd5afecab339a7e8a6041d2dc6b8"},{"id":"func/pathIsConfigRootArgument","name":"pathIsConfigRootArgument","line":284,"end_line":287,"hash":"2c04bfe68cfd97eb4f3f6c04b53fc35ff9790adb40cd8cd3bb058b291f8b87d7"},{"id":"func/workflowBlockConfigRootPath","name":"workflowBlockConfigRootPath","line":289,"end_line":302,"hash":"d56affa7dec5bb1627e4ba5d24814151213746fb5f93f3760f3c7396676b9f49"},{"id":"func/configRootPathForWorkingDirectory","name":"configRootPathForWorkingDirectory","line":304,"end_line":315,"hash":"4dcdc3228d40dda80a853142e614aa0e79f37ed35e5913820f0f244cdf8f15d1"},{"id":"func/commandTokensByLine","name":"commandTokensByLine","line":317,"end_line":327,"hash":"473038f0426a24c6b4e363d959eede2e11c464546fcb63cfcd890bdf78202960"},{"id":"func/commandTokens","name":"commandTokens","line":329,"end_line":345,"hash":"f32fec70071f8663fa0419d71a03f5fb3366b435caec75882b32f349910fcf74"},{"id":"func/isGoRunPackage","name":"isGoRunPackage","line":347,"end_line":361,"hash":"ad3d5e36fe615c72533773df6e748424102f4325c76ec6aad1ea9e0a63389a33"},{"id":"func/goCommandIndex","name":"goCommandIndex","line":363,"end_line":365,"hash":"eb772c36d8fd0e0f5653b7df42df51af5f0006b66eb49846d823bcc3d8f184d0"},{"id":"func/goGlobalFlagNeedsValue","name":"goGlobalFlagNeedsValue","line":367,"end_line":370,"hash":"773b5ed6501aed46bc93ec5ec2d45a63fdd26e949e68c11efc801db9e804aeb4"},{"id":"func/goRunPackageIndex","name":"goRunPackageIndex","line":372,"end_line":374,"hash":"9783a968a8fed4a3d9d0c84a19aad85e923616c2883c87108265bf223bebcf26"},{"id":"func/goArgumentIndex","name":"goArgumentIndex","line":376,"end_line":394,"hash":"e898590f0a844d84ae742af52cf2bdceb2aa554f40ea6f5146eacb856f09d0b2"},{"id":"func/goRunFlagNeedsValue","name":"goRunFlagNeedsValue","line":396,"end_line":404,"hash":"873931bc6faa45419d8a2f4f4299de8a4311cc2bedb20aa10e82d1f1cb1070df"},{"id":"func/isCommandToken","name":"isCommandToken","line":406,"end_line":412,"hash":"a9b9fbcd907311524054882a43d5bad01f4151675c0dfb47d2a008cd47e52c33"},{"id":"func/hasCommandPrefix","name":"hasCommandPrefix","line":414,"end_line":427,"hash":"2e25b4507e788fc9ff74f12c192a9eb5284295d18b4daf19406b7f9aa9a596c2"},{"id":"func/isEnvAssignmentToken","name":"isEnvAssignmentToken","line":429,"end_line":432,"hash":"86d6bac4dc482b400b9a51d1ac7ca35b524e40ad5ae1c8485901158446c4dc9b"},{"id":"func/hasMutationTargetAfter","name":"hasMutationTargetAfter","line":434,"end_line":451,"hash":"12b4ef6818d9af7cbd4ca715504eb6a5c5fbcbbf2ca2d6a42040bd11d12449fc"},{"id":"func/lineHasRequiredFlag","name":"lineHasRequiredFlag","line":453,"end_line":466,"hash":"02adf5dd9094d88475f1c363add5b996b1dd602d17cc4c9e019d26dd9d468061"},{"id":"func/lineHasBooleanFlag","name":"lineHasBooleanFlag","line":468,"end_line":479,"hash":"ecd862dc9a099e064ff2cfcb381dfba1649411181f06ae645760e9299caf7cdc"},{"id":"func/hasFlagValue","name":"hasFlagValue","line":481,"end_line":487,"hash":"b720a7659efb56c5cf4579893bc6b82ee9e04c16e9d904b492cd08ca6ca63d20"},{"id":"func/isGoToolPackageToken","name":"isGoToolPackageToken","line":489,"end_line":491,"hash":"ea241843ddfbe22a80886214806aabb44093e2242e059e18ce71ffb0ce4b52f6"},{"id":"func/isToolBinaryToken","name":"isToolBinaryToken","line":493,"end_line":495,"hash":"af955e9348f52943509c3c84bd4db4e73043284ddd80fa339594f9d3bd75bfbb"},{"id":"func/isSlophammerCommandToken","name":"isSlophammerCommandToken","line":497,"end_line":505,"hash":"e9b24403598f4df1548d1d838a5f3729036c92c0679b2ad5df726f4da7242935"},{"id":"func/stripGoModuleVersion","name":"stripGoModuleVersion","line":507,"end_line":512,"hash":"12076cdee75472d106e824e660cacc6773161b9d8b7faad7ba25d7b14ff2f65a"},{"id":"func/cleanCommandToken","name":"cleanCommandToken","line":514,"end_line":516,"hash":"639174e51127b3ec50424aa7adcc2cbf6779bbc33ea5bcce69003430f6dea969"},{"id":"func/isShellSeparator","name":"isShellSeparator","line":518,"end_line":525,"hash":"b6931917d87be3dbbe6fc1bcb3bfa1ad5906f2ca331864555c880e6711d97bb9"}]}
// mutate4go-manifest-end
