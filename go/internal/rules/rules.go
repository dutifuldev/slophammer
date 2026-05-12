package rules

import (
	"context"
	"fmt"
	"sort"

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
	Severity    Severity
	Path        string
	Message     string
	Description string
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
	ReadmeRequiredRuleID = "repo.readme-required"
	AgentsRequiredRuleID = "repo.agents-required"
	CIRequiredRuleID     = "repo.ci-required"
)

var defaultDefinitions = []Definition{
	{
		ID:          ReadmeRequiredRuleID,
		Severity:    SeverityError,
		Path:        "README.md",
		Message:     "README.md is required",
		Description: "The target repo should have a README.md.",
	},
	{
		ID:          AgentsRequiredRuleID,
		Severity:    SeverityError,
		Path:        "AGENTS.md",
		Message:     "AGENTS.md is required",
		Description: "The target repo should have an AGENTS.md.",
	},
	{
		ID:          CIRequiredRuleID,
		Severity:    SeverityError,
		Path:        ".github/workflows",
		Message:     ".github/workflows must contain at least one .yml or .yaml workflow",
		Description: "The target repo should have a CI workflow under .github/workflows.",
	},
}

func DefaultDefinitions() []Definition {
	return append([]Definition(nil), defaultDefinitions...)
}

func DefaultRules() []Rule {
	return []Rule{
		requiredFileRule{definition: mustDefaultDefinition(ReadmeRequiredRuleID)},
		requiredFileRule{definition: mustDefaultDefinition(AgentsRequiredRuleID)},
		ciRequiredRule{definition: mustDefaultDefinition(CIRequiredRuleID)},
	}
}

func Run(ctx context.Context, snapshot repo.Snapshot, ruleSet []Rule) Report {
	findings := make([]Finding, 0)
	for _, rule := range ruleSet {
		findings = append(findings, rule.Check(ctx, snapshot)...)
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].RuleID == findings[j].RuleID {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].RuleID < findings[j].RuleID
	})
	return Report{OK: len(findings) == 0, Findings: findings}
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

func mustDefaultDefinition(id string) Definition {
	for _, definition := range defaultDefinitions {
		if definition.ID == id {
			return definition
		}
	}
	panic("missing default rule definition: " + id)
}

func (d Definition) Metadata() Metadata {
	return Metadata{
		ID:          d.ID,
		Severity:    d.Severity,
		Description: d.Description,
	}
}

type requiredFileRule struct {
	definition Definition
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
