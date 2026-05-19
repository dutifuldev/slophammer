package rules

import (
	"context"
	"path"
	"strings"

	"github.com/dutifuldev/slophammer/go/internal/config"
	"github.com/dutifuldev/slophammer/go/internal/gotargets"
	"github.com/dutifuldev/slophammer/go/internal/gotools"
	"github.com/dutifuldev/slophammer/go/internal/repo"
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
	return goStaticRule{definition: definition, satisfied: hasDryCommand}
}

func newGoCRAPRule(definition Definition) Rule {
	return goStaticRule{definition: definition, satisfied: hasCRAP4GoGate}
}

func newGoMutationRule(definition Definition) Rule {
	return goMutationRule{definition: definition}
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

type goMutationRule struct {
	definition Definition
}

func (r goMutationRule) Metadata() Metadata {
	return r.definition.Metadata()
}

func (r goMutationRule) Check(_ context.Context, snapshot repo.Snapshot) []Finding {
	roots := goProjectRoots(snapshot)
	if len(roots) == 0 {
		return nil
	}
	for _, root := range roots {
		scoped := goProjectSnapshot(snapshot, root, roots)
		if !hasMutate4GoCommandForRoot(snapshot, scoped, root, roots) {
			return []Finding{finding(r.definition)}
		}
	}
	return nil
}

func hasGoModule(snapshot repo.Snapshot) bool {
	return snapshot.HasFileNamedFold("go.mod")
}

func hasGoTestCommand(snapshot repo.Snapshot) bool {
	return hasGoSubcommand(snapshot, "test")
}

func hasGoVetCommand(snapshot repo.Snapshot) bool {
	return hasGoSubcommand(snapshot, "vet")
}

func hasGoSubcommand(snapshot repo.Snapshot, subcommand string) bool {
	return hasRunnableCommandLine(snapshot, func(tokens []string) bool {
		return lineHasGoSubcommandAllPackages(tokens, subcommand)
	})
}

func hasGoLintConfigAndCommand(snapshot repo.Snapshot) bool {
	return hasGolangCIConfig(snapshot) && hasGolangCICommand(snapshot)
}

func hasGoCoverageGate(snapshot repo.Snapshot) bool {
	for _, file := range commandFiles(snapshot) {
		if fileHasGoCoverageGate(file) {
			return true
		}
	}
	return false
}

func fileHasGoCoverageGate(file repo.File) bool {
	combined := goCoverageEvidence{}
	for _, content := range commandSections(file) {
		section := coverageEvidence(content)
		if section.complete() {
			return true
		}
		combined = combined.merge(section)
	}
	return isWorkflowFilePath(file.Path) && combined.complete()
}

type goCoverageEvidence struct {
	hasProfile   bool
	hasCoverTool bool
	hasThreshold bool
}

func coverageEvidence(content string) goCoverageEvidence {
	return goCoverageEvidence{
		hasProfile:   contentHasCommandLine(content, lineHasGoTestCoverageProfileCommand),
		hasCoverTool: contentHasCommandLine(content, lineHasGoToolCoverCommand),
		hasThreshold: hasCoverageGateThreshold(content),
	}
}

func (e goCoverageEvidence) merge(other goCoverageEvidence) goCoverageEvidence {
	return goCoverageEvidence{
		hasProfile:   e.hasProfile || other.hasProfile,
		hasCoverTool: e.hasCoverTool || other.hasCoverTool,
		hasThreshold: e.hasThreshold || other.hasThreshold,
	}
}

func (e goCoverageEvidence) complete() bool {
	return e.hasProfile && e.hasCoverTool && e.hasThreshold
}

func hasGoComplexityLint(snapshot repo.Snapshot) bool {
	for _, file := range golangCIConfigFiles(snapshot) {
		if configEnablesComplexityLinter(file.Content) {
			return true
		}
	}
	return false
}

func hasDryCommand(snapshot repo.Snapshot) bool {
	for _, file := range commandFiles(snapshot) {
		for _, content := range commandSections(file) {
			if contentHasGoToolCommand(content, gotools.Dry4Go) ||
				contentHasSlophammerGoCommand(content, "dry", "") {
				return true
			}
		}
	}
	return false
}

func hasCRAP4GoGate(snapshot repo.Snapshot) bool {
	hasConfiguredThreshold := hasConfiguredCRAPThreshold(snapshot)
	for _, file := range commandFiles(snapshot) {
		for _, content := range commandSections(file) {
			if contentHasSlophammerGoCommand(content, "crap", "--max-score") ||
				(hasConfiguredThreshold && fileHasConfigBackedSlophammerGoCommand(file, "crap")) {
				return true
			}
			if !hasCRAPThreshold(content) {
				continue
			}
			if contentHasGoToolCommand(content, gotools.CRAP4Go) {
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
	hasConfiguredTargets := hasConfiguredGoMutationScope(snapshot, "", []string{""})
	for _, file := range commandFiles(snapshot) {
		for _, content := range commandSections(file) {
			if contentHasSlophammerGoCommand(content, "mutate", "--target") ||
				(hasConfiguredTargets && fileHasConfigBackedSlophammerGoCommand(file, "mutate")) {
				return true
			}
		}
	}
	return false
}

func hasMutate4GoCommandForRoot(full repo.Snapshot, scoped repo.Snapshot, root string, roots []string) bool {
	if hasDirectMutate4GoCommand(scoped) || hasSlophammerGoMutationTargetCommand(scoped) {
		return true
	}
	if hasModuleLocalSlophammerConfig(full, root) && hasConfiguredGoMutationScopeInSnapshot(scoped) {
		return hasLocalConfigBackedGoMutationCommand(full, scoped, root)
	}
	if hasConfiguredGoMutationScope(full, root, roots) {
		return hasConfigBackedSlophammerGoMutationCommand(scoped, false) ||
			hasRepoRootConfigBackedSlophammerGoMutationCommand(full)
	}
	return repoRootConfiguredGoMutationScopeExcludesRoot(full, root, roots)
}

func hasLocalConfigBackedGoMutationCommand(full repo.Snapshot, scoped repo.Snapshot, root string) bool {
	if !hasConfiguredGoMutationScopeInSnapshot(scoped) {
		return false
	}
	return hasModuleLocalConfigBackedSlophammerGoMutationCommand(full, root) ||
		hasConfigBackedSlophammerGoMutationCommandAtRoot(full, root) ||
		hasConfigBackedSlophammerGoMutationCommandInWorkingDir(full, root) ||
		hasConfigBackedSlophammerGoMutationCommandInWorkflowWorkingDir(full, root)
}

func hasSlophammerGoMutationTargetCommand(snapshot repo.Snapshot) bool {
	for _, file := range commandFiles(snapshot) {
		for _, content := range commandSections(file) {
			if contentHasSlophammerGoCommand(content, "mutate", "--target") {
				return true
			}
		}
	}
	return false
}

func hasConfigBackedSlophammerGoMutationCommand(snapshot repo.Snapshot, allowDefaultRoot bool) bool {
	for _, file := range commandFiles(snapshot) {
		if fileHasConfigBackedSlophammerGoCommand(file, "mutate") {
			return true
		}
		if allowDefaultRoot && !isWorkflowFilePath(file.Path) && fileHasConfigBackedSlophammerGoCommandAtRoot(file, "mutate", ".") {
			return true
		}
	}
	return false
}

func hasRepoRootConfigBackedSlophammerGoMutationCommand(snapshot repo.Snapshot) bool {
	moduleRoots := sortedRootSet(goModuleRootSet(snapshot))
	for _, file := range commandFiles(snapshot) {
		if commandFileIsUnderNestedGoModule(file.Path, moduleRoots) {
			continue
		}
		if fileHasConfigBackedSlophammerGoCommand(file, "mutate") {
			return true
		}
	}
	return false
}

func commandFileIsUnderNestedGoModule(filePath string, moduleRoots []string) bool {
	for _, root := range moduleRoots {
		if root != "" && strings.HasPrefix(filePath, root+"/") {
			return true
		}
	}
	return false
}

func hasConfigBackedSlophammerGoMutationCommandAtRoot(snapshot repo.Snapshot, root string) bool {
	if root == "" {
		root = "."
	}
	for _, file := range commandFiles(snapshot) {
		if fileHasConfigBackedSlophammerGoCommandAtRoot(file, "mutate", root) {
			return true
		}
	}
	return false
}

func hasConfigBackedSlophammerGoMutationCommandInWorkingDir(snapshot repo.Snapshot, root string) bool {
	if root == "" {
		root = "."
	}
	for _, file := range commandFiles(snapshot) {
		for _, content := range commandSections(file) {
			if contentHasConfigBackedSlophammerGoCommandInWorkingDir(content, "mutate", root) {
				return true
			}
		}
	}
	return false
}

func contentHasConfigBackedSlophammerGoCommandInWorkingDir(content string, subcommand string, workingDir string) bool {
	workingDir = cleanRuleSlashPath(workingDir)
	return contentHasCommandLine(content, func(tokens []string) bool {
		for i := 0; i < len(tokens); i++ {
			argsStart, ok := slophammerGoCommandArgsStart(tokens, i, subcommand)
			if !ok {
				continue
			}
			priorDir, ok := priorCDWorkingDirectory(tokens[:i])
			if !ok || cleanRuleSlashPath(priorDir) != workingDir {
				continue
			}
			if token, ok := firstSlophammerGoPathArgument(tokens[argsStart:]); ok {
				return pathIsConfigRootArgument(token, ".")
			}
			return true
		}
		return false
	})
}

func hasModuleLocalConfigBackedSlophammerGoMutationCommand(snapshot repo.Snapshot, root string) bool {
	if root == "" {
		return hasConfigBackedSlophammerGoMutationCommand(snapshot, true)
	}
	prefix := root + "/"
	for _, file := range commandFiles(snapshot) {
		if isWorkflowFilePath(file.Path) || !strings.HasPrefix(file.Path, prefix) {
			continue
		}
		if fileHasConfigBackedSlophammerGoCommand(file, "mutate") {
			return true
		}
	}
	return false
}

func hasConfigBackedSlophammerGoMutationCommandInWorkflowWorkingDir(snapshot repo.Snapshot, root string) bool {
	if root == "" {
		root = "."
	}
	root = cleanRuleSlashPath(root)
	for _, file := range commandFiles(snapshot) {
		if !isWorkflowFilePath(file.Path) {
			continue
		}
		for _, block := range workflowCommandBlocks(file.Content) {
			workingDir, ok := workflowBlockWorkingDirectory(block)
			if !ok || cleanRuleSlashPath(workingDir) != root {
				continue
			}
			if contentHasConfigBackedSlophammerGoCommand(workflowRunContent(block), "mutate", ".") {
				return true
			}
		}
	}
	return false
}

func workflowBlockWorkingDirectory(block string) (string, bool) {
	workingDirectory := ""
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "working-directory:") {
			continue
		}
		workingDirectory = strings.TrimSpace(strings.TrimPrefix(trimmed, "working-directory:"))
	}
	return workingDirectory, workingDirectory != ""
}

func cleanRuleSlashPath(value string) string {
	return path.Clean(strings.ReplaceAll(cleanCommandToken(value), "\\", "/"))
}

func fileHasConfigBackedSlophammerGoCommandAtRoot(file repo.File, subcommand string, configRootPath string) bool {
	if isWorkflowFilePath(file.Path) {
		for _, block := range workflowCommandBlocks(file.Content) {
			blockRootPath := configRootPath
			if workingDirectory, ok := workflowBlockWorkingDirectory(block); ok {
				blockRootPath = configRootPathFromWorkflowWorkingDirectory(configRootPath, workingDirectory)
			}
			if contentHasConfigBackedSlophammerGoCommand(workflowRunContent(block), subcommand, blockRootPath) {
				return true
			}
		}
		return false
	}
	for _, content := range commandSections(file) {
		if contentHasConfigBackedSlophammerGoCommand(content, subcommand, configRootPath) {
			return true
		}
	}
	return false
}

func configRootPathFromWorkflowWorkingDirectory(configRootPath string, workingDirectory string) string {
	return relativeRuleSlashPath(cleanRuleSlashPath(workingDirectory), cleanRuleSlashPath(configRootPath))
}

func relativeRuleSlashPath(from string, to string) string {
	fromParts := cleanRuleSlashPathParts(from)
	toParts := cleanRuleSlashPathParts(to)
	common := 0
	for common < len(fromParts) && common < len(toParts) && fromParts[common] == toParts[common] {
		common++
	}
	parts := make([]string, 0, len(fromParts)-common+len(toParts)-common)
	for range fromParts[common:] {
		parts = append(parts, "..")
	}
	parts = append(parts, toParts[common:]...)
	if len(parts) == 0 {
		return "."
	}
	return path.Join(parts...)
}

func cleanRuleSlashPathParts(value string) []string {
	cleaned := cleanRuleSlashPath(value)
	if cleaned == "." || cleaned == "/" {
		return nil
	}
	return strings.Split(strings.Trim(cleaned, "/"), "/")
}

func hasModuleLocalSlophammerConfig(snapshot repo.Snapshot, root string) bool {
	if root == "" {
		return snapshot.HasFileFold(config.DefaultFileName) || snapshot.HasFileFold(config.AltFileName)
	}
	return snapshot.HasFileFold(root+"/"+config.DefaultFileName) || snapshot.HasFileFold(root+"/"+config.AltFileName)
}

func hasConfiguredGoMutationScopeInSnapshot(snapshot repo.Snapshot) bool {
	cfg, err := config.Load(snapshot)
	if err != nil {
		return false
	}
	targets, exclude := cfg.GoMutationScope()
	if len(targets) == 0 {
		return false
	}
	_, err = resolveConfiguredGoMutationScope(snapshot, targets, exclude)
	return err == nil
}

func hasConfiguredCRAPThreshold(snapshot repo.Snapshot) bool {
	cfg, err := config.Load(snapshot)
	return err == nil && cfg.Go.CRAPMaxScore > 0
}

func hasConfiguredGoMutationScope(snapshot repo.Snapshot, root string, roots []string) bool {
	cfg, err := config.Load(snapshot)
	if err != nil {
		return false
	}
	targets, exclude := cfg.GoMutationScope()
	if len(targets) == 0 {
		return false
	}
	resolved, err := resolveConfiguredGoMutationScope(snapshot, targets, exclude)
	if err != nil {
		return false
	}
	for _, filePath := range resolved {
		if gotargets.ContainsPath(root, filePath) && !isUnderOtherGoRoot(filePath, root, roots) {
			return true
		}
	}
	return false
}

func repoRootConfiguredGoMutationScopeExcludesRoot(snapshot repo.Snapshot, root string, roots []string) bool {
	cfg, err := config.Load(snapshot)
	if err != nil || cfg.SourceDir != "." {
		return false
	}
	targets, exclude := cfg.GoMutationScope()
	if len(targets) == 0 {
		return false
	}
	resolved, err := resolveConfiguredGoMutationScope(snapshot, targets, exclude)
	if err != nil {
		return false
	}
	for _, filePath := range resolved {
		if gotargets.ContainsPath(root, filePath) && !isUnderOtherGoRoot(filePath, root, roots) {
			return false
		}
	}
	return true
}

func resolveConfiguredGoMutationScope(snapshot repo.Snapshot, targets []string, exclude []string) ([]string, error) {
	return gotargets.ResolveWithSingleModuleFallback(snapshot, gotargets.Options{
		Targets: targets,
		Exclude: exclude,
	}, sortedRootSet(goModuleRootSet(snapshot)), "")
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

func finding(definition Definition) Finding {
	return Finding{
		RuleID:   definition.ID,
		Severity: definition.Severity,
		Path:     definition.Path,
		Message:  definition.Message,
	}
}

const workflowStepBoundary = "\nSLOPHAMMER_WORKFLOW_STEP_BOUNDARY\n"
