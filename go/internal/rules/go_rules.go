package rules

import (
	"context"
	"strings"

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
	return goStaticRule{definition: definition, satisfied: hasCRAP4GoCommand}
}

func newGoMutationRule(definition Definition) Rule {
	return goStaticRule{definition: definition, satisfied: hasMutate4GoCommand}
}

func (r goStaticRule) Metadata() Metadata {
	return r.definition.Metadata()
}

func (r goStaticRule) Check(_ context.Context, snapshot repo.Snapshot) []Finding {
	if !isGoProject(snapshot) || r.satisfied(snapshot) {
		return nil
	}
	return []Finding{finding(r.definition)}
}

func hasGoModule(snapshot repo.Snapshot) bool {
	return snapshot.HasFileNamedFold("go.mod")
}

func hasGoTestCommand(snapshot repo.Snapshot) bool {
	return hasCommand(snapshot, "go test ./...")
}

func hasGoVetCommand(snapshot repo.Snapshot) bool {
	return hasCommand(snapshot, "go vet ./...")
}

func hasGoLintConfigAndCommand(snapshot repo.Snapshot) bool {
	return hasGolangCIConfig(snapshot) && hasCommand(snapshot, "golangci-lint", "golangci/golangci-lint-action")
}

func hasGoCoverageGate(snapshot repo.Snapshot) bool {
	return hasCommand(snapshot, "-coverprofile", "go tool cover", "check-go-coverage.sh")
}

func hasGoComplexityLint(snapshot repo.Snapshot) bool {
	return repo.ContainsAny(golangCIConfigFiles(snapshot), "cyclop", "gocognit", "gocyclo")
}

func hasDry4GoCommand(snapshot repo.Snapshot) bool {
	return hasCommand(snapshot, "dry4go", "github.com/unclebob/dry4go/cmd/dry4go")
}

func hasCRAP4GoCommand(snapshot repo.Snapshot) bool {
	return hasCommand(snapshot, "crap4go", "github.com/unclebob/crap4go/cmd/crap4go")
}

func hasMutate4GoCommand(snapshot repo.Snapshot) bool {
	return hasCommand(snapshot, "mutate4go", "github.com/unclebob/mutate4go/cmd/mutate4go")
}

func isGoProject(snapshot repo.Snapshot) bool {
	return hasGoModule(snapshot) || len(snapshot.FilesWithSuffix(".go")) > 0 || hasCommand(snapshot, "go test", "go vet")
}

func hasCommand(snapshot repo.Snapshot, needles ...string) bool {
	return repo.ContainsAny(commandFiles(snapshot), needles...)
}

func hasGolangCIConfig(snapshot repo.Snapshot) bool {
	return len(golangCIConfigFiles(snapshot)) > 0
}

func golangCIConfigFiles(snapshot repo.Snapshot) []repo.File {
	return snapshot.FilesNamedFold(".golangci.yml", ".golangci.yaml")
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
	files := make([]repo.File, 0, len(filesByPath))
	for _, file := range filesByPath {
		if strings.TrimSpace(file.Content) != "" {
			files = append(files, file)
		}
	}
	return files
}

func finding(definition Definition) Finding {
	return Finding{
		RuleID:   definition.ID,
		Severity: definition.Severity,
		Path:     definition.Path,
		Message:  definition.Message,
	}
}
