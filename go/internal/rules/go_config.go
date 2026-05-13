package rules

import (
	"strings"

	"github.com/dutifuldev/slophammer/go/internal/repo"
	"gopkg.in/yaml.v3"
)

func hasGolangCIConfig(snapshot repo.Snapshot) bool {
	return len(golangCIConfigFiles(snapshot)) > 0
}

func hasGolangCICommand(snapshot repo.Snapshot) bool {
	if hasCommand(snapshot, "golangci/golangci-lint-action") {
		return true
	}
	return hasRunnableCommandLine(snapshot, lineHasGolangCICommand)
}

func golangCIConfigFiles(snapshot repo.Snapshot) []repo.File {
	return snapshot.FilesNamedFold(".golangci.yml", ".golangci.yaml")
}

func hasCoverageThreshold(content string) bool {
	return coverageThresholdPattern.MatchString(content) || strictCoverageThresholdPattern.MatchString(content)
}

func hasCoverageGateThreshold(content string) bool {
	if !hasCoverageThreshold(content) {
		return false
	}
	lower := strings.ToLower(content)
	return strings.Contains(lower, "cover") ||
		strings.Contains(lower, "coverage") ||
		strings.Contains(lower, "total")
}

func hasCRAPThreshold(content string) bool {
	return crapThresholdPattern.MatchString(content) || strictCRAPThresholdPattern.MatchString(content)
}

func configEnablesComplexityLinter(content string) bool {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return false
	}
	root := yamlRoot(&document)
	linters := yamlMappingValue(root, "linters")
	disable := yamlMappingValue(linters, "disable")
	if yamlScalarEquals(yamlMappingValue(linters, "default"), "all") {
		return !yamlSequenceContainsAll(disable, "cyclop", "gocognit", "gocyclo")
	}
	enable := yamlMappingValue(linters, "enable")
	return yamlSequenceContainsEnabled(enable, disable, "cyclop", "gocognit", "gocyclo")
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

func yamlSequenceContainsAll(node *yaml.Node, values ...string) bool {
	for _, value := range values {
		if !yamlSequenceContains(node, value) {
			return false
		}
	}
	return true
}

func yamlSequenceContainsEnabled(enable *yaml.Node, disable *yaml.Node, values ...string) bool {
	for _, value := range values {
		if yamlSequenceContains(enable, value) && !yamlSequenceContains(disable, value) {
			return true
		}
	}
	return false
}

func yamlScalarEquals(node *yaml.Node, value string) bool {
	return node != nil && node.Kind == yaml.ScalarNode && node.Value == value
}
