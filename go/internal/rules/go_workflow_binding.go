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

// bindingFilteredWorkflow removes the source lines of neutralized jobs and
// steps from the raw workflow text, leaving every other byte untouched so
// downstream scoping sees the same shapes it always has. The second result
// is false when the workflow cannot be structurally filtered.
func bindingFilteredWorkflow(content string) (string, bool) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(content), &root); err != nil {
		return "", false
	}
	document := &root
	if document.Kind == yaml.DocumentNode && len(document.Content) > 0 {
		document = document.Content[0]
	}
	jobs := workflowJobsNode(document)
	if jobs == nil {
		return "", false
	}
	if !bindingWorkflowTriggers(workflowTriggerNode(document)) {
		return "", true
	}
	removals := neutralizedLineRanges(jobs)
	if len(removals) == 0 {
		return content, true
	}
	return removeLineRanges(content, removals), true
}

func workflowJobsNode(document *yaml.Node) *yaml.Node {
	jobs := yamlMappingValue(document, "jobs")
	if jobs == nil || jobs.Kind != yaml.MappingNode || len(jobs.Content) == 0 {
		return nil
	}
	return jobs
}

// workflowTriggerNode resolves the on: entry; YAML 1.1 parsers may resolve a
// plain on key as boolean true, so both spellings are checked.
func workflowTriggerNode(document *yaml.Node) yaml.Node {
	if node := yamlMappingValue(document, "on"); node != nil {
		return *node
	}
	if node := yamlMappingValue(document, "true"); node != nil {
		return *node
	}
	return yaml.Node{}
}

type lineRange struct {
	start int
	end   int
}

func neutralizedLineRanges(jobs *yaml.Node) []lineRange {
	ranges := make([]lineRange, 0)
	for i := 0; i+1 < len(jobs.Content); i += 2 {
		jobKey, jobValue := jobs.Content[i], jobs.Content[i+1]
		if neutralizedNode(jobValue) {
			ranges = append(ranges, lineRange{start: jobKey.Line, end: maxNodeLine(jobValue)})
			continue
		}
		ranges = append(ranges, neutralizedStepRanges(yamlMappingValue(jobValue, "steps"))...)
	}
	return ranges
}

func neutralizedStepRanges(steps *yaml.Node) []lineRange {
	if steps == nil || steps.Kind != yaml.SequenceNode {
		return nil
	}
	ranges := make([]lineRange, 0)
	for _, step := range steps.Content {
		if neutralizedNode(step) {
			ranges = append(ranges, lineRange{start: step.Line, end: maxNodeLine(step)})
		}
	}
	return ranges
}

// neutralizedNode reports whether a job or step mapping cannot run or cannot
// fail: a literal-false if condition or a literal continue-on-error.
func neutralizedNode(node *yaml.Node) bool {
	if value := yamlMappingValue(node, "continue-on-error"); value != nil && literalTrueNode(*value) {
		return true
	}
	if value := yamlMappingValue(node, "if"); value != nil && literalFalseCondition(value.Value) {
		return true
	}
	return false
}

func maxNodeLine(node *yaml.Node) int {
	highest := node.Line + blockScalarLineCount(node)
	for _, child := range node.Content {
		if line := maxNodeLine(child); line > highest {
			highest = line
		}
	}
	return highest
}

// blockScalarLineCount covers literal and folded scalars, whose content
// lines follow the node's own line without child nodes of their own.
func blockScalarLineCount(node *yaml.Node) int {
	if node.Kind != yaml.ScalarNode {
		return 0
	}
	if node.Style != yaml.LiteralStyle && node.Style != yaml.FoldedStyle {
		return 0
	}
	return strings.Count(strings.TrimRight(node.Value, "\n"), "\n") + 1
}

func removeLineRanges(content string, removals []lineRange) string {
	removed := map[int]bool{}
	for _, span := range removals {
		for line := span.start; line <= span.end; line++ {
			removed[line] = true
		}
	}
	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines))
	for index, line := range lines {
		if !removed[index+1] {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
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
// Run defaults apply only to run steps, so action-only steps carry no
// inherited directory.
func (s workflowStep) evidenceBlock(defaultDirectory string) string {
	lines := make([]string, 0, 3)
	if strings.TrimSpace(s.Run) != "" {
		lines = append(lines, s.Run)
		if directory := s.effectiveDirectory(defaultDirectory); directory != "" {
			lines = append(lines, "working-directory: "+directory)
		}
	}
	if strings.TrimSpace(s.Uses) != "" {
		lines = append(lines, "uses: "+s.Uses)
	}
	return strings.Join(lines, "\n")
}

func (s workflowStep) effectiveDirectory(defaultDirectory string) string {
	if s.WorkingDirectory != "" {
		return s.WorkingDirectory
	}
	return defaultDirectory
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
	return literalExpressionValue(condition) == "false"
}

func literalTrueNode(node yaml.Node) bool {
	if node.Kind != yaml.ScalarNode {
		return false
	}
	return literalExpressionValue(node.Value) == "true"
}

func literalExpressionValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "${{") && strings.HasSuffix(trimmed, "}}") {
		trimmed = strings.TrimSpace(trimmed[3 : len(trimmed)-2])
	}
	return trimmed
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

var bindingTriggerNames = map[string]struct{}{
	"push":                {},
	"pull_request":        {},
	"pull_request_target": {},
	"merge_group":         {},
	"schedule":            {},
}

func bindingTriggerName(name string) bool {
	_, ok := bindingTriggerNames[name]
	return ok
}

func bindingPushFilter(value *yaml.Node) bool {
	if value == nil || value.Kind != yaml.MappingNode {
		return true
	}
	tagFiltered := false
	branchesIgnored := false
	for i := 0; i+1 < len(value.Content); i += 2 {
		switch value.Content[i].Value {
		case "branches":
			return bindingBranchFilter(value.Content[i+1])
		case "branches-ignore":
			branchesIgnored = true
		case "tags", "tags-ignore":
			tagFiltered = true
		}
	}
	// Defining only tags or tags-ignore stops the workflow from firing for
	// branch pushes entirely, so it is a release trigger, not integration
	// CI; a branches-ignore filter still fires for branches.
	return branchesIgnored || !tagFiltered
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
