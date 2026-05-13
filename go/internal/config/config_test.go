package config

import (
	"reflect"
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
  dry_max_candidates: 0
  dry_paths:
    - go/cmd
    - go/internal
  dry_exclude:
    - "**/*_test.go"
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
	assertParsedGoPolicyConfig(t, cfg)
}

func assertParsedGoPolicyConfig(t *testing.T, cfg Config) {
	t.Helper()
	if cfg.Go.CoverageThreshold != 80 || cfg.Go.DRYMaxCandidates != 0 || !cfg.Go.DRYMaxCandidatesSet || cfg.Go.CRAPMaxScore != 30 {
		t.Fatalf("Go config = %#v", cfg.Go)
	}
	assertParsedDryPaths(t, cfg)
	assertParsedMutationTargets(t, cfg)
	assertParsedDependencyBoundaries(t, cfg)
}

func assertParsedDryPaths(t *testing.T, cfg Config) {
	t.Helper()
	if !reflect.DeepEqual(cfg.Go.DRYPaths, []string{"go/cmd", "go/internal"}) {
		t.Fatalf("DRYPaths = %#v", cfg.Go.DRYPaths)
	}
	if !reflect.DeepEqual(cfg.Go.DRYExclude, []string{"**/*_test.go"}) {
		t.Fatalf("DRYExclude = %#v", cfg.Go.DRYExclude)
	}
}

func assertParsedMutationTargets(t *testing.T, cfg Config) {
	t.Helper()
	if len(cfg.Go.MutationTargets) != 1 || cfg.Go.MutationTargets[0] != "internal/rules/rules.go" {
		t.Fatalf("MutationTargets = %#v", cfg.Go.MutationTargets)
	}
}

func assertParsedDependencyBoundaries(t *testing.T, cfg Config) {
	t.Helper()
	if len(cfg.Go.DependencyBoundaries) != 1 || cfg.Go.DependencyBoundaries[0].From != "internal/repo" {
		t.Fatalf("DependencyBoundaries = %#v", cfg.Go.DependencyBoundaries)
	}
}

func TestLoadPrefersRootConfig(t *testing.T) {
	cfg, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
		"fixtures/repos/go-bad-dependency/slophammer.yml": {
			Path: "fixtures/repos/go-bad-dependency/slophammer.yml",
			Content: `go:
  dry_max_candidates: 1
`,
		},
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  dry_max_candidates: 0
  mutation_targets:
    - go/internal/rules/rules.go
`,
		},
	}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Go.DRYMaxCandidates != 0 || !cfg.Go.DRYMaxCandidatesSet {
		t.Fatalf("DRYMaxCandidates = %d, set=%v, want 0 true", cfg.Go.DRYMaxCandidates, cfg.Go.DRYMaxCandidatesSet)
	}
	if len(cfg.Go.MutationTargets) != 1 || cfg.Go.MutationTargets[0] != "go/internal/rules/rules.go" {
		t.Fatalf("MutationTargets = %#v", cfg.Go.MutationTargets)
	}
}

func TestLoadRejectsNegativeDryBudget(t *testing.T) {
	_, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
		"slophammer.yml": {
			Path:    "slophammer.yml",
			Content: "go:\n  dry_max_candidates: -1\n",
		},
	}))
	if err == nil {
		t.Fatal("Load returned nil error")
	}
	if !strings.Contains(err.Error(), "dry_max_candidates") {
		t.Fatalf("error = %v", err)
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
