package rules

import (
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// bindingWorkflowCommandText returns the command evidence a workflow is
// allowed to contribute. Parsed workflows contribute only run scripts from
// steps that can execute and can fail on integration branches. Unparseable
// workflows fall back to the legacy line-based extraction so structural
// filtering can only remove false passes.
func bindingWorkflowCommandText(content string) string {
	var document workflowDocument
	if err := yaml.Unmarshal([]byte(content), &document); err != nil || len(document.Jobs) == 0 {
		return unparsedWorkflowCommandText(content)
	}
	if !bindingWorkflowTriggers(document.On) {
		return ""
	}
	return strings.Join(document.bindingStepBlocks(), workflowStepBoundary)
}

// unparsedWorkflowCommandText keeps both the extracted run lines (for command
// matching) and the raw text (for action-reference matching) of workflow
// content that cannot be structurally filtered, such as scoped step fragments.
func unparsedWorkflowCommandText(content string) string {
	sections := workflowCommandSections(content)
	if len(sections) == 0 {
		return content
	}
	return strings.Join(sections, "\n") + "\n" + content
}

type workflowDocument struct {
	On       yaml.Node              `yaml:"on"`
	Defaults workflowDefaults       `yaml:"defaults"`
	Jobs     map[string]workflowJob `yaml:"jobs"`
}

type workflowDefaults struct {
	Run workflowRunDefaults `yaml:"run"`
}

type workflowRunDefaults struct {
	WorkingDirectory string `yaml:"working-directory"`
}

type workflowJob struct {
	If              string           `yaml:"if"`
	ContinueOnError yaml.Node        `yaml:"continue-on-error"`
	Defaults        workflowDefaults `yaml:"defaults"`
	Steps           []workflowStep   `yaml:"steps"`
}

type workflowStep struct {
	If               string    `yaml:"if"`
	ContinueOnError  yaml.Node `yaml:"continue-on-error"`
	Run              string    `yaml:"run"`
	Uses             string    `yaml:"uses"`
	WorkingDirectory string    `yaml:"working-directory"`
}

// bindingStepBlocks renders each surviving step as a boundary-separated
// block of bare command text plus its effective working-directory context,
// so the block-level matchers keep their working-directory awareness.
func (d workflowDocument) bindingStepBlocks() []string {
	blocks := make([]string, 0)
	for _, name := range sortedJobNames(d.Jobs) {
		blocks = append(blocks, d.Jobs[name].bindingStepBlocks(d.Defaults.Run.WorkingDirectory)...)
	}
	return blocks
}

func sortedJobNames(jobs map[string]workflowJob) []string {
	names := make([]string, 0, len(jobs))
	for name := range jobs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (j workflowJob) bindingStepBlocks(workflowDirectory string) []string {
	if j.neutralized() {
		return nil
	}
	defaultDirectory := j.Defaults.Run.WorkingDirectory
	if defaultDirectory == "" {
		defaultDirectory = workflowDirectory
	}
	blocks := make([]string, 0, len(j.Steps))
	for _, step := range j.Steps {
		if step.neutralized() {
			continue
		}
		if block := step.evidenceBlock(defaultDirectory); block != "" {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

// evidenceBlock returns a step's command evidence: its run script and action
// reference, with the effective working-directory as a context line.
func (s workflowStep) evidenceBlock(defaultDirectory string) string {
	lines := make([]string, 0, 3)
	if strings.TrimSpace(s.Run) != "" {
		lines = append(lines, s.Run)
	}
	if strings.TrimSpace(s.Uses) != "" {
		lines = append(lines, "uses: "+s.Uses)
	}
	if len(lines) == 0 {
		return ""
	}
	directory := s.WorkingDirectory
	if directory == "" {
		directory = defaultDirectory
	}
	if directory != "" {
		lines = append(lines, "working-directory: "+directory)
	}
	return strings.Join(lines, "\n")
}

func (j workflowJob) neutralized() bool {
	return literalFalseCondition(j.If) || literalTrueNode(j.ContinueOnError)
}

func (s workflowStep) neutralized() bool {
	return literalFalseCondition(s.If) || literalTrueNode(s.ContinueOnError)
}

// literalFalseCondition reports whether an if: condition is the literal
// false, optionally wrapped in an expression. Non-literal expressions stay
// credited; the checker ships no expression evaluator.
func literalFalseCondition(condition string) bool {
	trimmed := strings.TrimSpace(condition)
	if strings.HasPrefix(trimmed, "${{") && strings.HasSuffix(trimmed, "}}") {
		trimmed = strings.TrimSpace(trimmed[3 : len(trimmed)-2])
	}
	return trimmed == "false"
}

func literalTrueNode(node yaml.Node) bool {
	if node.Kind != yaml.ScalarNode {
		return false
	}
	return strings.TrimSpace(node.Value) == "true"
}

// bindingWorkflowTriggers reports whether a workflow's triggers can fire for
// integration: pull requests, merge groups, schedules, or pushes whose branch
// filter is absent, wildcarded, or names a plausible integration branch.
func bindingWorkflowTriggers(on yaml.Node) bool {
	switch on.Kind {
	case yaml.ScalarNode:
		return bindingTriggerName(on.Value)
	case yaml.SequenceNode:
		return anyBindingTriggerName(on.Content)
	case yaml.MappingNode:
		return anyBindingTriggerEntry(on.Content)
	default:
		return false
	}
}

func anyBindingTriggerName(names []*yaml.Node) bool {
	for _, name := range names {
		if bindingTriggerName(name.Value) {
			return true
		}
	}
	return false
}

func anyBindingTriggerEntry(entries []*yaml.Node) bool {
	for i := 0; i+1 < len(entries); i += 2 {
		if bindingTriggerEntry(entries[i].Value, entries[i+1]) {
			return true
		}
	}
	return false
}

func bindingTriggerEntry(name string, value *yaml.Node) bool {
	switch name {
	case "pull_request", "pull_request_target", "merge_group", "schedule":
		return true
	case "push":
		return bindingPushFilter(value)
	default:
		return false
	}
}

func bindingTriggerName(name string) bool {
	switch name {
	case "push", "pull_request", "pull_request_target", "merge_group", "schedule":
		return true
	default:
		return false
	}
}

func bindingPushFilter(value *yaml.Node) bool {
	if value == nil || value.Kind != yaml.MappingNode {
		return true
	}
	for i := 0; i+1 < len(value.Content); i += 2 {
		if value.Content[i].Value == "branches" {
			return bindingBranchFilter(value.Content[i+1])
		}
	}
	return true
}

func bindingBranchFilter(branches *yaml.Node) bool {
	if branches.Kind != yaml.SequenceNode {
		return integrationBranchPattern(branches.Value)
	}
	for _, branch := range branches.Content {
		if integrationBranchPattern(branch.Value) {
			return true
		}
	}
	return false
}

func integrationBranchPattern(pattern string) bool {
	if strings.Contains(pattern, "*") {
		return true
	}
	switch pattern {
	case "main", "master", "trunk", "develop":
		return true
	default:
		return false
	}
}
