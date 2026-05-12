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

type Metadata struct {
	ID          string
	Severity    Severity
	Description string
}

type Rule interface {
	Metadata() Metadata
	Check(context.Context, repo.Snapshot) []Finding
}

func DefaultRules() []Rule {
	return []Rule{
		requiredFileRule{
			metadata: Metadata{
				ID:          "repo.readme-required",
				Severity:    SeverityError,
				Description: "The target repo should have a README.md.",
			},
			path:    "README.md",
			message: "README.md is required",
		},
		requiredFileRule{
			metadata: Metadata{
				ID:          "repo.agents-required",
				Severity:    SeverityError,
				Description: "The target repo should have an AGENTS.md.",
			},
			path:    "AGENTS.md",
			message: "AGENTS.md is required",
		},
		ciRequiredRule{},
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

type requiredFileRule struct {
	metadata Metadata
	path     string
	message  string
}

func (r requiredFileRule) Metadata() Metadata {
	return r.metadata
}

func (r requiredFileRule) Check(_ context.Context, snapshot repo.Snapshot) []Finding {
	if snapshot.HasFileFold(r.path) {
		return nil
	}
	return []Finding{{
		RuleID:   r.metadata.ID,
		Severity: r.metadata.Severity,
		Path:     r.path,
		Message:  r.message,
	}}
}

type ciRequiredRule struct{}

func (ciRequiredRule) Metadata() Metadata {
	return Metadata{
		ID:          "repo.ci-required",
		Severity:    SeverityError,
		Description: "The target repo should have a CI workflow under .github/workflows.",
	}
}

func (r ciRequiredRule) Check(_ context.Context, snapshot repo.Snapshot) []Finding {
	if len(snapshot.WorkflowFiles()) > 0 {
		return nil
	}
	metadata := r.Metadata()
	return []Finding{{
		RuleID:   metadata.ID,
		Severity: metadata.Severity,
		Path:     ".github/workflows",
		Message:  ".github/workflows must contain at least one .yml or .yaml workflow",
	}}
}
