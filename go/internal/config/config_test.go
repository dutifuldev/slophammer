package config

import (
	"strings"
	"testing"

	"github.com/dutifuldev/slophammer/go/internal/repo"
)

func TestLoadReturnsZeroConfigWhenMissing(t *testing.T) {
	cfg, err := Load(repo.NewSnapshot("/repo", nil))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(cfg.Go.DependencyBoundaries) != 0 {
		t.Fatalf("DependencyBoundaries = %#v, want empty", cfg.Go.DependencyBoundaries)
	}
}

func TestLoadParsesPolicyConfig(t *testing.T) {
	cfg, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `rules:
  go.crap-required:
    severity: warn
go:
  coverage_threshold: 80
  dry_max_candidates: 40
  crap_max_score: 30
  mutation_targets:
    - internal/rules/rules.go
  dependency_boundaries:
    - from: internal/repo
      allow: []
`,
		},
	}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got := cfg.RuleSeverity("go.crap-required", "error"); got != "warn" {
		t.Fatalf("RuleSeverity = %q, want warn", got)
	}
	if cfg.Go.CoverageThreshold != 80 || cfg.Go.DRYMaxCandidates != 40 || cfg.Go.CRAPMaxScore != 30 {
		t.Fatalf("Go config = %#v", cfg.Go)
	}
	if len(cfg.Go.MutationTargets) != 1 || cfg.Go.MutationTargets[0] != "internal/rules/rules.go" {
		t.Fatalf("MutationTargets = %#v", cfg.Go.MutationTargets)
	}
	if len(cfg.Go.DependencyBoundaries) != 1 || cfg.Go.DependencyBoundaries[0].From != "internal/repo" {
		t.Fatalf("DependencyBoundaries = %#v", cfg.Go.DependencyBoundaries)
	}
}

func TestLoadRejectsInvalidConfig(t *testing.T) {
	_, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
		"slophammer.yml": {
			Path:    "slophammer.yml",
			Content: "rules:\n  go.crap-required:\n    severity: info\n",
		},
	}))
	if err == nil {
		t.Fatal("Load returned nil error")
	}
	if !strings.Contains(err.Error(), "severity") {
		t.Fatalf("error = %v", err)
	}
}
