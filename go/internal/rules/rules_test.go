package rules

import (
	"context"
	"testing"

	"github.com/dutifuldev/slophammer/go/internal/repo"
)

func TestDefaultRulesPassForMinimalRepo(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                 {Path: "README.md"},
		"AGENTS.md":                 {Path: "AGENTS.md"},
		".github/workflows/ci.yaml": {Path: ".github/workflows/ci.yaml"},
	})

	report := Run(context.Background(), snapshot, DefaultRules())
	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestDefaultRulesReportMissingFiles(t *testing.T) {
	report := Run(context.Background(), repo.NewSnapshot("/repo", nil), DefaultRules())

	if report.OK {
		t.Fatal("report.OK = true, want false")
	}
	wantRuleIDs := []string{
		"repo.agents-required",
		"repo.ci-required",
		"repo.readme-required",
	}
	if len(report.Findings) != len(wantRuleIDs) {
		t.Fatalf("len(findings) = %d, want %d", len(report.Findings), len(wantRuleIDs))
	}
	for i, want := range wantRuleIDs {
		if report.Findings[i].RuleID != want {
			t.Fatalf("finding[%d].RuleID = %q, want %q", i, report.Findings[i].RuleID, want)
		}
	}
}

func TestExplainKnownRule(t *testing.T) {
	got, ok := Explain(DefaultRules(), "repo.agents-required")
	if !ok {
		t.Fatal("Explain returned ok=false")
	}
	if got == "" {
		t.Fatal("Explain returned empty output")
	}
}

func TestExplainUnknownRule(t *testing.T) {
	_, ok := Explain(DefaultRules(), "missing")
	if ok {
		t.Fatal("Explain returned ok=true for missing rule")
	}
}
