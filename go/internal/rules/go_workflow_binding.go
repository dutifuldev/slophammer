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
	if branches := yamlMappingValue(value, "branches"); branches != nil {
		return bindingBranchFilter(branches)
	}
	return pushFiltersAllowBranches(value)
}

// Defining only tags or tags-ignore stops the workflow from firing for
// branch pushes entirely, so it is a release trigger, not integration CI;
// a branches-ignore filter still fires for branches.
func pushFiltersAllowBranches(value *yaml.Node) bool {
	if yamlMappingValue(value, "branches-ignore") != nil {
		return true
	}
	return yamlMappingValue(value, "tags") == nil && yamlMappingValue(value, "tags-ignore") == nil
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

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-06-12T22:50:29+08:00","module_hash":"c952dbbb42f0cc0081036ff1b53d4b84036bea614f6b29e14a31351a72fe974c","functions":[{"id":"func/bindingWorkflowCommandText","name":"bindingWorkflowCommandText","line":15,"end_line":24,"hash":"70b7b3b663d54f482d540c9630d6249e17a86f1707baacd82bd12274e145be4d"},{"id":"func/bindingFilteredWorkflow","name":"bindingFilteredWorkflow","line":30,"end_line":51,"hash":"a12ed47c860032e51b7d0739edc54a7624fd38e48d320792baa96dfa3c0a6922"},{"id":"func/workflowJobsNode","name":"workflowJobsNode","line":53,"end_line":59,"hash":"aa7bc57b1018f3727e0139f24807fc0d15914c75de814fbb202709f396767a7d"},{"id":"func/workflowTriggerNode","name":"workflowTriggerNode","line":63,"end_line":71,"hash":"f376bcce12c1ad78ca5d2a1f057fc64d1cdbd225ed50f9474d9f4a9f137008f8"},{"id":"func/neutralizedLineRanges","name":"neutralizedLineRanges","line":78,"end_line":89,"hash":"0a8f36a8461d50ba42769f4acbd2f9e03f66aac273a291f47271db3fce5b6e0d"},{"id":"func/neutralizedStepRanges","name":"neutralizedStepRanges","line":91,"end_line":102,"hash":"493fe5893eb33c52803bd9482a1da9747ce3e15d72b3ed50f2da58167ae8c05a"},{"id":"func/neutralizedNode","name":"neutralizedNode","line":106,"end_line":114,"hash":"d91b909f0ebedec6ec39d460cbe1da13947879379e099f2db5129b2751c7c4a4"},{"id":"func/maxNodeLine","name":"maxNodeLine","line":116,"end_line":124,"hash":"fdea281d6805d9b44ce5ed9945173efb9e3fce51887dd815c2d115fac44ca1fd"},{"id":"func/blockScalarLineCount","name":"blockScalarLineCount","line":128,"end_line":136,"hash":"de96dbd65508bd3c5c5255117ef8c784ec9baf01c44efcc6e7ce62e0d35c4275"},{"id":"func/removeLineRanges","name":"removeLineRanges","line":138,"end_line":153,"hash":"fafbba62936b5597467da362e54c13ed8debb40936df951e98fbda1401c40221"},{"id":"func/unparsedWorkflowCommandText","name":"unparsedWorkflowCommandText","line":158,"end_line":164,"hash":"9abcaf974500a583b68ace282ab29c229c1a10e21e1e008042336ea4a27ac7ad"},{"id":"func/workflowDocument.bindingStepBlocks","name":"workflowDocument.bindingStepBlocks","line":198,"end_line":204,"hash":"c119e51973f5fff0640c1c7beaad21f12e7f704b61c399683221cd7afc26add4"},{"id":"func/sortedJobNames","name":"sortedJobNames","line":206,"end_line":213,"hash":"1485c946610a8cc6543f2ca5e52653ceaac4c87549e177251ee5ab1b323fa305"},{"id":"func/workflowJob.bindingStepBlocks","name":"workflowJob.bindingStepBlocks","line":215,"end_line":233,"hash":"aa99fa29660e8dc417984f345ff9d106810d0ffce3aba304dcf7e03488730235"},{"id":"func/workflowStep.evidenceBlock","name":"workflowStep.evidenceBlock","line":239,"end_line":251,"hash":"b774bd4b1f8c528f340f5281e8fb9cbd244b9ca0ff43c1d820750d73ddf882b8"},{"id":"func/workflowStep.effectiveDirectory","name":"workflowStep.effectiveDirectory","line":253,"end_line":258,"hash":"79ee383e8deaf86da173035c46c173df362e0e19e5d7eb6aac2b1bb765719d28"},{"id":"func/workflowJob.neutralized","name":"workflowJob.neutralized","line":260,"end_line":262,"hash":"058f7b7355b3f211575b98675f2e4605177d642c878630eccbaa9c182abd3578"},{"id":"func/workflowStep.neutralized","name":"workflowStep.neutralized","line":264,"end_line":266,"hash":"28de9c1442bf33e222ec5445168e69347d0ed3f9a7164639d37200023cc4211b"},{"id":"func/literalFalseCondition","name":"literalFalseCondition","line":271,"end_line":273,"hash":"e5ae2af4c30f4ce79eee5b11581a49f548e921dde95edfca08fff73b58c864fd"},{"id":"func/literalTrueNode","name":"literalTrueNode","line":275,"end_line":280,"hash":"d393af7b6a2e922fbd06877ec89074ac771e5a4dff65daf098557ebaa7bd8ff7"},{"id":"func/literalExpressionValue","name":"literalExpressionValue","line":282,"end_line":288,"hash":"bd419b71028c9dd59806f9f8d895900c6b0ee3b6861ba52d7f3fd327e57c5b31"},{"id":"func/bindingWorkflowTriggers","name":"bindingWorkflowTriggers","line":293,"end_line":304,"hash":"13c47fb47c2f567ddc904393aec7fbb678a1174344484af50063af13f9e97dd2"},{"id":"func/anyBindingTriggerName","name":"anyBindingTriggerName","line":306,"end_line":313,"hash":"a584b820dc873b4ce16d1d20069670de5377a325639568294104ea00b01f6260"},{"id":"func/anyBindingTriggerEntry","name":"anyBindingTriggerEntry","line":315,"end_line":322,"hash":"1f4fafb2a7e75b8f8322d03d4362e45b37ef8e155bb366302abdc5d438427054"},{"id":"func/bindingTriggerEntry","name":"bindingTriggerEntry","line":324,"end_line":333,"hash":"891ea1bf17b93aad8e2288736bf6cfb29901fb0a58eb2b4127dc646fca4f98ee"},{"id":"func/bindingTriggerName","name":"bindingTriggerName","line":343,"end_line":346,"hash":"317966c7ea6e68bfcbdde4f34c6f56c6f8013a5417190796b132517d9d3e91c6"},{"id":"func/bindingPushFilter","name":"bindingPushFilter","line":348,"end_line":356,"hash":"aa5b7cdb80589488573a34d459aa01b8f751304d89473d86891a50970835494d"},{"id":"func/pushFiltersAllowBranches","name":"pushFiltersAllowBranches","line":361,"end_line":366,"hash":"01f835c42b2309002883e5f0acfb4ed01e10fc7be056abf334b825ca54278fd6"},{"id":"func/bindingBranchFilter","name":"bindingBranchFilter","line":368,"end_line":378,"hash":"72c955849c22c3b29f9283794fc8b41ffbce1b2088c6642921335230ec539da9"},{"id":"func/integrationBranchPattern","name":"integrationBranchPattern","line":380,"end_line":390,"hash":"7aecd997ec52266a3279af6b5c2045c417b7d985d7cf76ce53b4f47c6210d57c"}]}
// mutate4go-manifest-end
