package rules

import (
	"context"

	"github.com/dutifuldev/slophammer/go/internal/config"
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

func hasDry4GoCommand(snapshot repo.Snapshot) bool {
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
				(hasConfiguredThreshold && contentHasSlophammerGoCommand(content, "crap", "")) {
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
	hasConfiguredTargets := hasConfiguredMutationTargets(snapshot)
	for _, file := range commandFiles(snapshot) {
		for _, content := range commandSections(file) {
			if contentHasSlophammerGoCommand(content, "mutate", "--target") ||
				(hasConfiguredTargets && contentHasSlophammerGoCommand(content, "mutate", "")) {
				return true
			}
		}
	}
	return false
}

func hasConfiguredCRAPThreshold(snapshot repo.Snapshot) bool {
	cfg, err := config.Load(snapshot)
	return err == nil && cfg.Go.CRAPMaxScore > 0
}

func hasConfiguredMutationTargets(snapshot repo.Snapshot) bool {
	cfg, err := config.Load(snapshot)
	return err == nil && len(cfg.Go.MutationTargets) > 0
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
