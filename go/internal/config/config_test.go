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
  coverage_threshold: 85
  coverage_profile: coverage.out
  targets:
    - go
  exclude:
    - "go/generated/**"
  dry_max_candidates: 0
  dry_paths:
    - go/cmd
    - go/internal
  dry_exclude:
    - "**/*_test.go"
  crap_max_score: 8
  mutation:
    targets:
      - internal/rules
    exclude:
      - "internal/rules/generated/**"
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
	if cfg.Go.CoverageThreshold != 85 || cfg.Go.DRYMaxCandidates != 0 || !cfg.Go.DRYMaxCandidatesSet || cfg.Go.CRAPMaxScore != 8 {
		t.Fatalf("Go config = %#v", cfg.Go)
	}
	if cfg.Go.CoverageProfile != "coverage.out" {
		t.Fatalf("CoverageProfile = %q, want coverage.out", cfg.Go.CoverageProfile)
	}
	assertParsedDryPaths(t, cfg)
	assertParsedTargets(t, cfg)
	assertParsedMutation(t, cfg)
	assertParsedDependencyBoundaries(t, cfg)
}

func assertParsedTargets(t *testing.T, cfg Config) {
	t.Helper()
	if !reflect.DeepEqual(cfg.Go.Targets, []string{"go"}) {
		t.Fatalf("Targets = %#v", cfg.Go.Targets)
	}
	if !reflect.DeepEqual(cfg.Go.Exclude, []string{"go/generated/**"}) {
		t.Fatalf("Exclude = %#v", cfg.Go.Exclude)
	}
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

func assertParsedMutation(t *testing.T, cfg Config) {
	t.Helper()
	if !reflect.DeepEqual(cfg.Go.Mutation.Targets, []string{"internal/rules"}) {
		t.Fatalf("Mutation.Targets = %#v", cfg.Go.Mutation.Targets)
	}
	if !reflect.DeepEqual(cfg.Go.Mutation.Exclude, []string{"internal/rules/generated/**"}) {
		t.Fatalf("Mutation.Exclude = %#v", cfg.Go.Mutation.Exclude)
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
  targets:
    - go/internal/rules
`,
		},
	}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Go.DRYMaxCandidates != 0 || !cfg.Go.DRYMaxCandidatesSet {
		t.Fatalf("DRYMaxCandidates = %d, set=%v, want 0 true", cfg.Go.DRYMaxCandidates, cfg.Go.DRYMaxCandidatesSet)
	}
	if !reflect.DeepEqual(cfg.Go.Targets, []string{"go/internal/rules"}) {
		t.Fatalf("Targets = %#v", cfg.Go.Targets)
	}
}

func TestGoMutationScopeIsRelativeToConfigFile(t *testing.T) {
	cfg, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
		"go/slophammer.yml": {
			Path: "go/slophammer.yml",
			Content: `go:
  targets:
    - internal
  exclude:
    - "generated/**"
`,
		},
	}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	targets, exclude := cfg.GoMutationScope()

	if !reflect.DeepEqual(targets, []string{"go/internal"}) {
		t.Fatalf("targets = %#v", targets)
	}
	if !reflect.DeepEqual(exclude, []string{"generated/**", "go/generated/**"}) {
		t.Fatalf("exclude = %#v", exclude)
	}
}

func TestGoCoverageProfileIsRelativeToConfigFile(t *testing.T) {
	cfg, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
		"go/slophammer.yml": {
			Path: "go/slophammer.yml",
			Content: `go:
  coverage_profile: coverage.out
`,
		},
	}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got := cfg.GoCoverageProfile(); got != "go/coverage.out" {
		t.Fatalf("GoCoverageProfile = %q, want go/coverage.out", got)
	}
}

func TestGoCoverageProfilePreservesAbsoluteWindowsPath(t *testing.T) {
	cfg, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
		"go/slophammer.yml": {
			Path: "go/slophammer.yml",
			Content: `go:
  coverage_profile: 'C:\ci\coverage.out'
`,
		},
	}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got := cfg.GoCoverageProfile(); got != `C:\ci\coverage.out` {
		t.Fatalf("GoCoverageProfile = %q, want Windows absolute path", got)
	}
}

func TestGoMutationScopePreservesBasenameExcludesForConfigFile(t *testing.T) {
	cfg, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
		"go/slophammer.yml": {
			Path: "go/slophammer.yml",
			Content: `go:
  targets:
    - internal
  exclude:
    - "*.pb.go"
`,
		},
	}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	targets, exclude := cfg.GoMutationScope()

	if !reflect.DeepEqual(targets, []string{"go/internal"}) {
		t.Fatalf("targets = %#v", targets)
	}
	if !reflect.DeepEqual(exclude, []string{"*.pb.go"}) {
		t.Fatalf("exclude = %#v", exclude)
	}
}

func TestGoMutationScopeUsesMutationOverride(t *testing.T) {
	cfg, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
		"go/slophammer.yml": {
			Path: "go/slophammer.yml",
			Content: `go:
  targets:
    - internal
  mutation:
    targets:
      - cmd
    exclude:
      - "cmd/generated/**"
`,
		},
	}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	targets, exclude := cfg.GoMutationScope()

	if !reflect.DeepEqual(targets, []string{"go/cmd"}) {
		t.Fatalf("targets = %#v", targets)
	}
	if !reflect.DeepEqual(exclude, []string{"cmd/generated/**", "go/cmd/generated/**"}) {
		t.Fatalf("exclude = %#v", exclude)
	}
}

func TestGoMutationScopeDoesNotInheritSharedExcludeForMutationTargets(t *testing.T) {
	cfg, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  targets:
    - internal
  exclude:
    - "generated/**"
  mutation:
    targets:
      - internal/generated
`,
		},
	}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	targets, exclude := cfg.GoMutationScope()

	if !reflect.DeepEqual(targets, []string{"internal/generated"}) {
		t.Fatalf("targets = %#v", targets)
	}
	if len(exclude) != 0 {
		t.Fatalf("exclude = %#v, want empty", exclude)
	}
}

func TestGoMutationScopeUsesMutationExcludeWithSharedTargets(t *testing.T) {
	cfg, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  targets:
    - internal
  exclude:
    - "internal/generated/**"
  mutation:
    exclude:
      - "internal/mutation_generated/**"
`,
		},
	}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	targets, exclude := cfg.GoMutationScope()

	if !reflect.DeepEqual(targets, []string{"internal"}) {
		t.Fatalf("targets = %#v", targets)
	}
	if !reflect.DeepEqual(exclude, []string{"internal/mutation_generated/**"}) {
		t.Fatalf("exclude = %#v", exclude)
	}
}

func TestLoadParsesNestedDryConfig(t *testing.T) {
	cfg, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  dry:
    max_findings: 0
    paths:
      - go/internal
    exclude:
      - "**/*_test.go"
    structural:
      enabled: true
      threshold: 0.82
      min_lines: 4
      min_nodes: 20
    copied_blocks:
      enabled: true
      min_tokens: 100
`,
		},
	}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	assertNestedDryConfig(t, cfg)
}

func assertNestedDryConfig(t *testing.T, cfg Config) {
	t.Helper()
	if cfg.Go.DRYMaxCandidates != 0 || !cfg.Go.DRYMaxCandidatesSet || !cfg.Go.DRY.MaxFindingsSet {
		t.Fatalf("DRY budget = %#v", cfg.Go)
	}
	if !reflect.DeepEqual(cfg.Go.DRYPaths, []string{"go/internal"}) || !reflect.DeepEqual(cfg.Go.DRY.Paths, []string{"go/internal"}) {
		t.Fatalf("DRY paths = %#v nested=%#v", cfg.Go.DRYPaths, cfg.Go.DRY.Paths)
	}
	assertNestedDryEngines(t, cfg.Go.DRY)
}

func assertNestedDryEngines(t *testing.T, cfg DryConfig) {
	t.Helper()
	if !cfg.Structural.EnabledSet || !cfg.Structural.Enabled || cfg.Structural.Threshold != 0.82 {
		t.Fatalf("DRY structural = %#v", cfg.Structural)
	}
	if !cfg.CopiedBlocks.EnabledSet || !cfg.CopiedBlocks.Enabled || cfg.CopiedBlocks.MinTokens != 100 {
		t.Fatalf("DRY copied blocks = %#v", cfg.CopiedBlocks)
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

func TestLoadRejectsInvalidNestedDryTargets(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		want    string
	}{
		{name: "threshold", content: "go:\n  dry:\n    structural:\n      threshold: 1.2\n", want: "threshold"},
		{name: "min lines", content: "go:\n  dry:\n    structural:\n      min_lines: -1\n", want: "min_lines"},
		{name: "min nodes", content: "go:\n  dry:\n    structural:\n      min_nodes: -1\n", want: "min_nodes"},
		{name: "copied tokens", content: "go:\n  dry:\n    copied_blocks:\n      min_tokens: -1\n", want: "min_tokens"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
				"slophammer.yml": {Path: "slophammer.yml", Content: tc.content},
			}))
			if err == nil {
				t.Fatal("Load returned nil error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestLoadRejectsWeakerRecommendedGoTargets(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "coverage",
			content: "go:\n  coverage_threshold: 84\n",
			want:    "coverage_threshold",
		},
		{
			name:    "crap",
			content: "go:\n  crap_max_score: 9\n",
			want:    "crap_max_score",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
				"slophammer.yml": {Path: "slophammer.yml", Content: tt.content},
			}))
			if err == nil {
				t.Fatal("Load returned nil error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v", err)
			}
		})
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

func TestLoadRejectsUnknownConfigKeys(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "root", content: "made_up: true\n", want: "root.made_up"},
		{name: "rules", content: "rules:\n  repo.readme-required:\n    made_up: true\n", want: "rules.repo.readme-required.made_up"},
		{name: "go", content: "go:\n  made_up: true\n", want: "go.made_up"},
		{name: "go mutation", content: "go:\n  mutation:\n    made_up: true\n", want: "go.mutation.made_up"},
		{name: "removed go mutation targets", content: "go:\n  mutation_targets:\n    - main.go\n", want: "go.mutation_targets"},
		{name: "go dry", content: "go:\n  dry:\n    made_up: true\n", want: "go.dry.made_up"},
		{name: "go structural", content: "go:\n  dry:\n    structural:\n      made_up: true\n", want: "go.dry.structural.made_up"},
		{name: "go boundary", content: "go:\n  dependency_boundaries:\n    - from: internal/app\n      made_up: true\n", want: "go.dependency_boundaries[0].made_up"},
		{name: "typescript", content: "typescript:\n  made_up: true\n", want: "typescript.made_up"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
				"slophammer.yml": {Path: "slophammer.yml", Content: tt.content},
			}))
			if err == nil {
				t.Fatal("Load returned nil error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestLoadAllowsSharedGoAndTypeScriptConfig(t *testing.T) {
	_, err := Load(repo.NewSnapshot("/repo", map[string]repo.File{
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  coverage_threshold: 85
typescript:
  coverage_threshold: 85
  complexity_max: 8
  dry:
    copied_blocks:
      enabled: true
      min_tokens: 100
`,
		},
	}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
}
