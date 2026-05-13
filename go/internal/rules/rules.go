package rules

import (
	"context"
	"fmt"
	"sort"

	"github.com/dutifuldev/slophammer/go/internal/config"
	"github.com/dutifuldev/slophammer/go/internal/repo"
)

type Severity string

const (
	SeverityError Severity = "error"
	SeverityWarn  Severity = "warn"
)

type Finding struct {
	RuleID   string   `json:"rule_id"`
	Severity Severity `json:"severity"`
	Path     string   `json:"path"`
	Message  string   `json:"message"`
}

type Report struct {
	OK       bool      `json:"ok"`
	Findings []Finding `json:"findings"`
}

type Definition struct {
	ID          string
	Title       string
	Category    string
	Severity    Severity
	Path        string
	Message     string
	Description string
	Tool        string
	Status      string
}

type Metadata struct {
	ID          string
	Severity    Severity
	Description string
}

type Rule interface {
	Metadata() Metadata
	Check(context.Context, repo.Snapshot) []Finding
}

const (
	ReadmeRequiredRuleID         = "repo.readme-required"
	AgentsRequiredRuleID         = "repo.agents-required"
	CIRequiredRuleID             = "repo.ci-required"
	GoModuleRequiredRuleID       = "go.module-required"
	GoTestsRequiredRuleID        = "go.tests-required"
	GoVetRequiredRuleID          = "go.vet-required"
	GoLintRequiredRuleID         = "go.lint-required"
	GoCoverageRequiredRuleID     = "go.coverage-required"
	GoComplexityRequiredRuleID   = "go.complexity-required"
	GoDryRequiredRuleID          = "go.dry-required"
	GoCRAPRequiredRuleID         = "go.crap-required"
	GoMutationRequiredRuleID     = "go.mutation-required"
	GoDependencyBoundariesRuleID = "go.dependency-boundaries-required"
)

var defaultDefinitions = []Definition{
	{
		ID:          ReadmeRequiredRuleID,
		Title:       "README required",
		Category:    "repo",
		Severity:    SeverityError,
		Path:        "README.md",
		Message:     "README.md is required",
		Description: "The target repo should have a README.md.",
		Status:      "implemented",
	},
	{
		ID:          AgentsRequiredRuleID,
		Title:       "Agent instructions required",
		Category:    "repo",
		Severity:    SeverityError,
		Path:        "AGENTS.md",
		Message:     "AGENTS.md is required",
		Description: "The target repo should have an AGENTS.md.",
		Status:      "implemented",
	},
	{
		ID:          CIRequiredRuleID,
		Title:       "CI workflow required",
		Category:    "repo",
		Severity:    SeverityError,
		Path:        ".github/workflows",
		Message:     ".github/workflows must contain at least one .yml or .yaml workflow",
		Description: "The target repo should have a CI workflow under .github/workflows.",
		Status:      "implemented",
	},
	{
		ID:          GoModuleRequiredRuleID,
		Title:       "Go module required",
		Category:    "go",
		Severity:    SeverityError,
		Path:        "go.mod",
		Message:     "Go projects must include a go.mod file",
		Description: "Go projects should be Go modules.",
		Tool:        "go",
		Status:      "implemented",
	},
	{
		ID:          GoTestsRequiredRuleID,
		Title:       "Go tests required",
		Category:    "go",
		Severity:    SeverityError,
		Path:        ".github/workflows",
		Message:     "Go projects must declare go test ./... in CI or scripts",
		Description: "Go projects should run go test against ./... in an inspectable workflow or script.",
		Tool:        "go test",
		Status:      "implemented",
	},
	{
		ID:          GoVetRequiredRuleID,
		Title:       "Go vet required",
		Category:    "go",
		Severity:    SeverityError,
		Path:        ".github/workflows",
		Message:     "Go projects must declare go vet ./... in CI or scripts",
		Description: "Go projects should run go vet ./... in an inspectable workflow or script.",
		Tool:        "go vet",
		Status:      "implemented",
	},
	{
		ID:          GoLintRequiredRuleID,
		Title:       "Go lint required",
		Category:    "go",
		Severity:    SeverityError,
		Path:        ".golangci.yml",
		Message:     "Go projects must configure and declare golangci-lint",
		Description: "Go projects should configure golangci-lint and declare a lint check in CI or scripts.",
		Tool:        "golangci-lint",
		Status:      "implemented",
	},
	{
		ID:          GoCoverageRequiredRuleID,
		Title:       "Go coverage gate required",
		Category:    "go",
		Severity:    SeverityError,
		Path:        "scripts/check-go-coverage.sh",
		Message:     "Go projects must declare a coverage gate",
		Description: "Go projects should enforce coverage with go test coverage output, go tool cover, and a minimum threshold.",
		Tool:        "go test -coverprofile",
		Status:      "implemented",
	},
	{
		ID:          GoComplexityRequiredRuleID,
		Title:       "Go complexity linting required",
		Category:    "go",
		Severity:    SeverityError,
		Path:        ".golangci.yml",
		Message:     "Go projects must enable a complexity linter",
		Description: "Go projects should enable a complexity linter through golangci-lint.",
		Tool:        "golangci-lint",
		Status:      "implemented",
	},
	{
		ID:          GoDryRequiredRuleID,
		Title:       "Go DRY check required",
		Category:    "go",
		Severity:    SeverityError,
		Path:        ".github/workflows",
		Message:     "Go projects must declare dry4go",
		Description: "Go projects should declare dry4go for structural duplicate detection.",
		Tool:        "dry4go",
		Status:      "implemented",
	},
	{
		ID:          GoCRAPRequiredRuleID,
		Title:       "Go CRAP check required",
		Category:    "go",
		Severity:    SeverityError,
		Path:        ".github/workflows",
		Message:     "Go projects must declare crap4go with a threshold",
		Description: "Go projects should declare crap4go with a threshold for complexity and coverage risk scoring.",
		Tool:        "crap4go",
		Status:      "implemented",
	},
	{
		ID:          GoMutationRequiredRuleID,
		Title:       "Go mutation check required",
		Category:    "go",
		Severity:    SeverityError,
		Path:        ".github/workflows",
		Message:     "Go projects must declare mutate4go",
		Description: "Go projects should declare mutate4go in an inspectable workflow or script.",
		Tool:        "mutate4go",
		Status:      "implemented",
	},
	{
		ID:          GoDependencyBoundariesRuleID,
		Title:       "Go dependency boundaries required",
		Category:    "go",
		Severity:    SeverityError,
		Path:        "slophammer.yml",
		Message:     "Go projects must respect configured dependency boundaries",
		Description: "Go projects should keep imports inside the dependency boundaries declared in slophammer.yml.",
		Status:      "implemented",
	},
}

func DefaultDefinitions() []Definition {
	return append([]Definition(nil), defaultDefinitions...)
}

func DefaultRules() []Rule {
	ruleSet := make([]Rule, 0, len(defaultDefinitions))
	for _, definition := range defaultDefinitions {
		ruleSet = append(ruleSet, ruleFromDefinition(definition))
	}
	return ruleSet
}

func Run(ctx context.Context, snapshot repo.Snapshot, ruleSet []Rule) Report {
	return RunWithConfig(ctx, snapshot, ruleSet, config.Config{})
}

func RunWithConfig(ctx context.Context, snapshot repo.Snapshot, ruleSet []Rule, cfg config.Config) Report {
	findings := make([]Finding, 0)
	for _, rule := range ruleSet {
		findings = append(findings, checkRule(ctx, rule, snapshot, cfg)...)
	}
	applyConfig(findings, cfg)
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].RuleID == findings[j].RuleID {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].RuleID < findings[j].RuleID
	})
	return Report{OK: len(findings) == 0, Findings: findings}
}

type configuredRule interface {
	CheckWithConfig(context.Context, repo.Snapshot, config.Config) []Finding
}

func checkRule(ctx context.Context, rule Rule, snapshot repo.Snapshot, cfg config.Config) []Finding {
	if configured, ok := rule.(configuredRule); ok {
		return configured.CheckWithConfig(ctx, snapshot, cfg)
	}
	return rule.Check(ctx, snapshot)
}

func applyConfig(findings []Finding, cfg config.Config) {
	for i := range findings {
		findings[i].Severity = Severity(cfg.RuleSeverity(findings[i].RuleID, string(findings[i].Severity)))
	}
}

func Find(ruleSet []Rule, id string) (Metadata, bool) {
	for _, rule := range ruleSet {
		metadata := rule.Metadata()
		if metadata.ID == id {
			return metadata, true
		}
	}
	return Metadata{}, false
}

func Explain(ruleSet []Rule, id string) (string, bool) {
	metadata, ok := Find(ruleSet, id)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%s\nseverity: %s\n\n%s\n", metadata.ID, metadata.Severity, metadata.Description), true
}

func (d Definition) Metadata() Metadata {
	return Metadata{
		ID:          d.ID,
		Severity:    d.Severity,
		Description: d.Description,
	}
}

func ruleFromDefinition(definition Definition) Rule {
	factory, ok := ruleFactories[definition.ID]
	if !ok {
		panic("missing rule implementation: " + definition.ID)
	}
	return factory(definition)
}

type ruleFactory func(Definition) Rule

var ruleFactories = map[string]ruleFactory{
	ReadmeRequiredRuleID:         newRequiredFileRule,
	AgentsRequiredRuleID:         newRequiredFileRule,
	CIRequiredRuleID:             newCIRequiredRule,
	GoModuleRequiredRuleID:       newGoModuleRule,
	GoTestsRequiredRuleID:        newGoTestsRule,
	GoVetRequiredRuleID:          newGoVetRule,
	GoLintRequiredRuleID:         newGoLintRule,
	GoCoverageRequiredRuleID:     newGoCoverageRule,
	GoComplexityRequiredRuleID:   newGoComplexityRule,
	GoDryRequiredRuleID:          newGoDryRule,
	GoCRAPRequiredRuleID:         newGoCRAPRule,
	GoMutationRequiredRuleID:     newGoMutationRule,
	GoDependencyBoundariesRuleID: newGoDependencyBoundariesRule,
}

type requiredFileRule struct {
	definition Definition
}

func newRequiredFileRule(definition Definition) Rule {
	return requiredFileRule{definition: definition}
}

func (r requiredFileRule) Metadata() Metadata {
	return r.definition.Metadata()
}

func (r requiredFileRule) Check(_ context.Context, snapshot repo.Snapshot) []Finding {
	if snapshot.HasFileFold(r.definition.Path) {
		return nil
	}
	return []Finding{{
		RuleID:   r.definition.ID,
		Severity: r.definition.Severity,
		Path:     r.definition.Path,
		Message:  r.definition.Message,
	}}
}

type ciRequiredRule struct {
	definition Definition
}

func newCIRequiredRule(definition Definition) Rule {
	return ciRequiredRule{definition: definition}
}

func (r ciRequiredRule) Metadata() Metadata {
	return r.definition.Metadata()
}

func (r ciRequiredRule) Check(_ context.Context, snapshot repo.Snapshot) []Finding {
	if len(snapshot.WorkflowFiles()) > 0 {
		return nil
	}
	return []Finding{{
		RuleID:   r.definition.ID,
		Severity: r.definition.Severity,
		Path:     r.definition.Path,
		Message:  r.definition.Message,
	}}
}
