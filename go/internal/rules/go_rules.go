package rules

import (
	"context"
	"regexp"
	"strings"

	"github.com/dutifuldev/slophammer/go/internal/repo"
	"gopkg.in/yaml.v3"
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
	return hasCommandPattern(snapshot, goTestAllPackagesPattern)
}

func hasGoVetCommand(snapshot repo.Snapshot) bool {
	return hasCommandPattern(snapshot, goVetAllPackagesPattern)
}

func hasGoLintConfigAndCommand(snapshot repo.Snapshot) bool {
	return hasGolangCIConfig(snapshot) && hasCommand(snapshot, "golangci-lint", "golangci/golangci-lint-action")
}

func hasGoCoverageGate(snapshot repo.Snapshot) bool {
	files := commandFiles(snapshot)
	return repo.ContainsAny(files, "-coverprofile") &&
		repo.ContainsAny(files, "go tool cover") &&
		hasCoverageThreshold(files)
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

func hasCoverageThreshold(files []repo.File) bool {
	for _, file := range files {
		if coverageThresholdPattern.MatchString(file.Content) {
			return true
		}
	}
	return false
}

func configEnablesComplexityLinter(content string) bool {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return false
	}
	root := yamlRoot(&document)
	linters := yamlMappingValue(root, "linters")
	if yamlScalarEquals(yamlMappingValue(linters, "default"), "all") {
		return true
	}
	enable := yamlMappingValue(linters, "enable")
	return yamlSequenceContains(enable, "cyclop", "gocognit", "gocyclo")
}

func yamlRoot(node *yaml.Node) *yaml.Node {
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return node
}

func yamlMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func yamlSequenceContains(node *yaml.Node, values ...string) bool {
	if node == nil || node.Kind != yaml.SequenceNode {
		return false
	}
	for _, item := range node.Content {
		for _, value := range values {
			if item.Value == value {
				return true
			}
		}
	}
	return false
}

func yamlScalarEquals(node *yaml.Node, value string) bool {
	return node != nil && node.Kind == yaml.ScalarNode && node.Value == value
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
		file.Content = stripCommentLines(file.Content)
		if strings.TrimSpace(file.Content) != "" {
			files = append(files, file)
		}
	}
	return files
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

func hasCommandPattern(snapshot repo.Snapshot, pattern *regexp.Regexp) bool {
	for _, file := range commandFiles(snapshot) {
		if pattern.MatchString(file.Content) {
			return true
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

var (
	goTestAllPackagesPattern = regexp.MustCompile(`(?m)\bgo\s+test\b[^\n#;&|]*\./\.\.`)
	goVetAllPackagesPattern  = regexp.MustCompile(`(?m)\bgo\s+vet\b[^\n#;&|]*\./\.\.`)
	coverageThresholdPattern = regexp.MustCompile(`(?im)\b(total|cover|coverage|minimum|threshold|required)\b[^\n]*(>=|<=|-ge\b|-le\b|-gt\b|-lt\b)|(?:>=|<=|-ge\b|-le\b|-gt\b|-lt\b)[^\n]*\b(total|cover|coverage|minimum|threshold|required)\b`)
)
