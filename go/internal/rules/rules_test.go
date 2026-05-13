package rules

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/dutifuldev/slophammer/go/internal/gotools"
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
		AgentsRequiredRuleID,
		CIRequiredRuleID,
		ReadmeRequiredRuleID,
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

func TestDefaultRulesReportMissingGoGuardrails(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md": {Path: "README.md"},
		"AGENTS.md": {Path: "AGENTS.md"},
		"main.go":   {Path: "main.go"},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	wantRuleIDs := []string{
		GoComplexityRequiredRuleID,
		GoCoverageRequiredRuleID,
		GoCRAPRequiredRuleID,
		GoDryRequiredRuleID,
		GoLintRequiredRuleID,
		GoModuleRequiredRuleID,
		GoMutationRequiredRuleID,
		GoTestsRequiredRuleID,
		GoVetRequiredRuleID,
		CIRequiredRuleID,
	}
	assertRuleIDs(t, report.Findings, wantRuleIDs)
}

func TestDefaultRulesPassForGoRepoWithDeclaredGuardrails(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", cleanGoGuardrailFiles(nil))

	report := Run(context.Background(), snapshot, DefaultRules())
	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoTestsRuleAcceptsFlagsBeforePackagePattern(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		".github/workflows/ci.yml": {
			Path:    ".github/workflows/ci.yml",
			Content: strings.ReplaceAll(goCleanWorkflow, "go test ./...", "go test -race -count=1 ./..."),
		},
		"go/scripts/check-go-coverage.sh": {
			Path:    "go/scripts/check-go-coverage.sh",
			Content: strings.ReplaceAll(cleanCoverageScript, "go test -coverprofile=coverage.out ./...", "go test -coverprofile=coverage.out ./internal/..."),
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoCommandRulesAcceptShellContinuations(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
defaults:
  run:
    working-directory: go
jobs:
  test:
    steps:
      - run: |
          go test \
            -race \
            ./...
      - run: |
          go vet \
            ./...
      - run: golangci-lint run
      - run: ./scripts/check-go-coverage.sh
`,
		},
		"go/scripts/check-go-coverage.sh": {
			Path:    "go/scripts/check-go-coverage.sh",
			Content: strings.ReplaceAll(cleanCoverageScript, "go test -coverprofile=coverage.out ./...", "go test -coverprofile=coverage.out ./internal/..."),
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoCommandRulesIgnoreCommentedCommands(t *testing.T) {
	tests := []struct {
		name      string
		workflow  string
		overrides map[string]repo.File
		want      string
	}{
		{
			name: "commented go test",
			workflow: `name: CI
defaults:
  run:
    working-directory: go
jobs:
  test:
    steps:
      - run: "# go test ./..."
      - run: go vet ./...
      - run: golangci-lint run
      - run: ./scripts/check-go-coverage.sh
`,
			overrides: map[string]repo.File{
				"go/scripts/check-go-coverage.sh": {
					Path:    "go/scripts/check-go-coverage.sh",
					Content: strings.ReplaceAll(cleanCoverageScript, "go test -coverprofile=coverage.out ./...", "go test -coverprofile=coverage.out ./internal/..."),
				},
			},
			want: GoTestsRequiredRuleID,
		},
		{
			name: "commented go vet",
			workflow: `name: CI
defaults:
  run:
    working-directory: go
jobs:
  test:
    steps:
      - run: go test ./...
      - run: "# go vet ./..."
      - run: golangci-lint run
      - run: ./scripts/check-go-coverage.sh
`,
			want: GoVetRequiredRuleID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrides := map[string]repo.File{
				".github/workflows/ci.yml": {Path: ".github/workflows/ci.yml", Content: tt.workflow},
			}
			for path, file := range tt.overrides {
				overrides[path] = file
			}
			files := cleanGoGuardrailFiles(overrides)

			report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

			assertRuleIDs(t, report.Findings, []string{tt.want})
		})
	}
}

func TestGoRulesInspectNestedModuleScripts(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                              {Path: "README.md"},
		"AGENTS.md":                              {Path: "AGENTS.md"},
		"services/api/go.mod":                    {Path: "services/api/go.mod"},
		"services/api/main.go":                   {Path: "services/api/main.go"},
		"services/api/.golangci.yml":             {Path: "services/api/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		".github/workflows/ci.yml":               {Path: ".github/workflows/ci.yml", Content: nestedGoWorkflow},
		"services/api/scripts/check-coverage.sh": {Path: "services/api/scripts/check-coverage.sh", Content: cleanCoverageScript},
		"services/api/scripts/check-dry.sh":      {Path: "services/api/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/api/scripts/check-crap.sh":     {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/api/scripts/check-mutation.sh": {Path: "services/api/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest internal/rules/rules.go --scan\n"},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoRulesIgnoreEmbeddedFixtureEvidence(t *testing.T) {
	files := cleanGoGuardrailFiles(nil)
	delete(files, "go/.golangci.yml")
	delete(files, "go/scripts/check-mutation.sh")
	files["fixtures/repos/go-clean/go/.golangci.yml"] = repo.File{
		Path:    "fixtures/repos/go-clean/go/.golangci.yml",
		Content: "linters:\n  enable:\n    - cyclop\n",
	}
	files["fixtures/repos/go-clean/go/scripts/check-mutation.sh"] = repo.File{
		Path:    "fixtures/repos/go-clean/go/scripts/check-mutation.sh",
		Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest internal/rules/rules.go --scan\n",
	}

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	assertRuleIDs(t, report.Findings, []string{
		GoComplexityRequiredRuleID,
		GoLintRequiredRuleID,
		GoMutationRequiredRuleID,
	})
}

func TestGoRulesDoNotTreatEmbeddedFixturesAsTargetProjects(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                         {Path: "README.md"},
		"AGENTS.md":                         {Path: "AGENTS.md"},
		".github/workflows/ci.yml":          {Path: ".github/workflows/ci.yml"},
		"fixtures/repos/go-missing/go.mod":  {Path: "fixtures/repos/go-missing/go.mod"},
		"fixtures/repos/go-missing/main.go": {Path: "fixtures/repos/go-missing/main.go"},
		"templates/go/go.mod":               {Path: "templates/go/go.mod"},
		"templates/go/main.go":              {Path: "templates/go/main.go"},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoRulesIgnoreVendoredModules(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		"go/vendor/example.com/dep/go.mod": {Path: "go/vendor/example.com/dep/go.mod"},
		"go/vendor/example.com/dep/dep.go": {Path: "go/vendor/example.com/dep/dep.go"},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoRulesIgnoreNonGoCommandSubstrings(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md": {Path: "README.md"},
		"AGENTS.md": {Path: "AGENTS.md"},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
defaults:
  run:
    working-directory: go
jobs:
  test:
    steps:
      - run: cargo test
      - run: python manage.py django test
      - run: echo "go test ./..."
`,
		},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoCommandRulesAcceptEnvPrefixedCommands(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
defaults:
  run:
    working-directory: go
jobs:
  test:
    steps:
      - run: CGO_ENABLED=0 go test ./...
      - run: GOFLAGS=-mod=readonly go vet ./...
      - run: env GOLANGCI_LINT_CACHE=/tmp golangci-lint run
      - run: ./scripts/check-go-coverage.sh
`,
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoRulesScopeWorkflowEvidenceToModuleRoots(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                                    {Path: "README.md"},
		"AGENTS.md":                                    {Path: "AGENTS.md"},
		"services/api/go.mod":                          {Path: "services/api/go.mod"},
		"services/api/main.go":                         {Path: "services/api/main.go"},
		"services/api/.golangci.yml":                   {Path: "services/api/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/api/scripts/check-go-coverage.sh":    {Path: "services/api/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/api/scripts/check-dry.sh":            {Path: "services/api/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/api/scripts/check-crap.sh":           {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/api/scripts/check-mutation.sh":       {Path: "services/api/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		"services/worker/go.mod":                       {Path: "services/worker/go.mod"},
		"services/worker/main.go":                      {Path: "services/worker/main.go"},
		"services/worker/.golangci.yml":                {Path: "services/worker/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/worker/scripts/check-go-coverage.sh": {Path: "services/worker/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/worker/scripts/check-dry.sh":         {Path: "services/worker/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/worker/scripts/check-crap.sh":        {Path: "services/worker/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/worker/scripts/check-mutation.sh":    {Path: "services/worker/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  api:
    steps:
      - run: cd services/api && go test ./...
      - run: cd services/api && go vet ./...
      - run: cd services/api && golangci-lint run
`,
		},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	assertRuleIDs(t, report.Findings, []string{
		GoLintRequiredRuleID,
		GoVetRequiredRuleID,
	})
}

func TestGoRulesAcceptDotSlashWorkflowModuleRoots(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                                    {Path: "README.md"},
		"AGENTS.md":                                    {Path: "AGENTS.md"},
		"services/api/go.mod":                          {Path: "services/api/go.mod"},
		"services/api/main.go":                         {Path: "services/api/main.go"},
		"services/api/.golangci.yml":                   {Path: "services/api/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/api/scripts/check-go-coverage.sh":    {Path: "services/api/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/api/scripts/check-dry.sh":            {Path: "services/api/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/api/scripts/check-crap.sh":           {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/api/scripts/check-mutation.sh":       {Path: "services/api/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		"services/worker/go.mod":                       {Path: "services/worker/go.mod"},
		"services/worker/main.go":                      {Path: "services/worker/main.go"},
		"services/worker/.golangci.yml":                {Path: "services/worker/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/worker/scripts/check-go-coverage.sh": {Path: "services/worker/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/worker/scripts/check-dry.sh":         {Path: "services/worker/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/worker/scripts/check-crap.sh":        {Path: "services/worker/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/worker/scripts/check-mutation.sh":    {Path: "services/worker/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  api:
    steps:
      - run: cd ./services/api && go test ./...
      - run: cd ./services/api && go vet ./...
      - run: cd ./services/api && golangci-lint run
  worker:
    steps:
      - run: cd ./services/worker && go test ./...
      - run: cd ./services/worker && go vet ./...
      - run: cd ./services/worker && golangci-lint run
`,
		},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoRulesFilterWorkflowCommandsPerModule(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                                    {Path: "README.md"},
		"AGENTS.md":                                    {Path: "AGENTS.md"},
		"services/api/go.mod":                          {Path: "services/api/go.mod"},
		"services/api/main.go":                         {Path: "services/api/main.go"},
		"services/api/.golangci.yml":                   {Path: "services/api/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/api/scripts/check-go-coverage.sh":    {Path: "services/api/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/api/scripts/check-dry.sh":            {Path: "services/api/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/api/scripts/check-crap.sh":           {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/api/scripts/check-mutation.sh":       {Path: "services/api/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		"services/worker/go.mod":                       {Path: "services/worker/go.mod"},
		"services/worker/main.go":                      {Path: "services/worker/main.go"},
		"services/worker/.golangci.yml":                {Path: "services/worker/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/worker/scripts/check-go-coverage.sh": {Path: "services/worker/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/worker/scripts/check-dry.sh":         {Path: "services/worker/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/worker/scripts/check-crap.sh":        {Path: "services/worker/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/worker/scripts/check-mutation.sh":    {Path: "services/worker/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  api:
    steps:
      - run: cd services/api && go test ./...
      - run: cd services/api && go vet ./...
      - run: cd services/api && golangci-lint run
  worker:
    steps:
      - run: cd services/worker && golangci-lint run
      - run: echo services/worker
`,
		},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	assertRuleIDs(t, report.Findings, []string{GoVetRequiredRuleID})
}

func TestScopedWorkflowStepBlockFiltersMixedModuleCommands(t *testing.T) {
	block := `      - run: |
          cd services/worker && go test ./...
          cd services/api
          go vet ./...
          golangci-lint run
          cd ../worker
          go vet ./...
`

	scoped, ok := scopedWorkflowStepBlock(block, "services/api", []string{"services/api", "services/worker"})

	if !ok {
		t.Fatal("scopedWorkflowStepBlock returned ok=false")
	}
	if strings.Contains(scoped, "go test ./...") {
		t.Fatalf("scoped block leaked worker command: %q", scoped)
	}
	if strings.Contains(scoped, "cd ../worker") {
		t.Fatalf("scoped block kept command after leaving api: %q", scoped)
	}
	if !strings.Contains(scoped, "- run: |") ||
		!strings.Contains(scoped, "cd services/api") ||
		!strings.Contains(scoped, "go vet ./...") ||
		!strings.Contains(scoped, "golangci-lint run") {
		t.Fatalf("scoped block lost api run content: %q", scoped)
	}
}

func TestGoRulesKeepRootWorkflowStepsWithNestedModules(t *testing.T) {
	files := map[string]repo.File{
		"README.md":                                 {Path: "README.md"},
		"AGENTS.md":                                 {Path: "AGENTS.md"},
		"go.mod":                                    {Path: "go.mod"},
		"main.go":                                   {Path: "main.go"},
		".golangci.yml":                             {Path: ".golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"scripts/check-go-coverage.sh":              {Path: "scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"scripts/check-dry.sh":                      {Path: "scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"scripts/check-crap.sh":                     {Path: "scripts/check-crap.sh", Content: cleanCRAPScript},
		"scripts/check-mutation.sh":                 {Path: "scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		"services/api/go.mod":                       {Path: "services/api/go.mod"},
		"services/api/main.go":                      {Path: "services/api/main.go"},
		"services/api/.golangci.yml":                {Path: "services/api/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/api/scripts/check-go-coverage.sh": {Path: "services/api/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/api/scripts/check-dry.sh":         {Path: "services/api/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/api/scripts/check-crap.sh":        {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/api/scripts/check-mutation.sh":    {Path: "services/api/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  root:
    steps:
      - run: go test ./...
      - run: go vet ./...
      - run: golangci-lint run
  api:
    steps:
      - run: cd services/api && go test ./...
      - run: cd services/api && go vet ./...
      - run: cd services/api && golangci-lint run
`,
		},
	}

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoRulesKeepWholeWorkflowStepForModule(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                                    {Path: "README.md"},
		"AGENTS.md":                                    {Path: "AGENTS.md"},
		"services/api/go.mod":                          {Path: "services/api/go.mod"},
		"services/api/main.go":                         {Path: "services/api/main.go"},
		"services/api/.golangci.yml":                   {Path: "services/api/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/api/scripts/check-go-coverage.sh":    {Path: "services/api/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/api/scripts/check-dry.sh":            {Path: "services/api/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/api/scripts/check-crap.sh":           {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/api/scripts/check-mutation.sh":       {Path: "services/api/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		"services/worker/go.mod":                       {Path: "services/worker/go.mod"},
		"services/worker/main.go":                      {Path: "services/worker/main.go"},
		"services/worker/.golangci.yml":                {Path: "services/worker/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/worker/scripts/check-go-coverage.sh": {Path: "services/worker/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/worker/scripts/check-dry.sh":         {Path: "services/worker/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/worker/scripts/check-crap.sh":        {Path: "services/worker/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/worker/scripts/check-mutation.sh":    {Path: "services/worker/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  go:
    steps:
      - run: go test ./...
        working-directory: services/api
      - run: go vet ./...
        working-directory: services/api
      - uses: golangci/golangci-lint-action@v8
        with:
          working-directory: services/api
      - run: go test ./...
        working-directory: services/worker
      - run: go vet ./...
        working-directory: services/worker
      - uses: golangci/golangci-lint-action@v8
        with:
          working-directory: services/worker
`,
		},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoRulesMatchNestedModuleWorkflowRootsOnBoundaries(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                              {Path: "README.md"},
		"AGENTS.md":                              {Path: "AGENTS.md"},
		"services/go.mod":                        {Path: "services/go.mod"},
		"services/main.go":                       {Path: "services/main.go"},
		"services/.golangci.yml":                 {Path: "services/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/scripts/check-go-coverage.sh":  {Path: "services/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/scripts/check-dry.sh":          {Path: "services/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/scripts/check-crap.sh":         {Path: "services/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/scripts/check-mutation.sh":     {Path: "services/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		"services/api/go.mod":                    {Path: "services/api/go.mod"},
		"services/api/main.go":                   {Path: "services/api/main.go"},
		"services/api/.golangci.yml":             {Path: "services/api/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/api/scripts/check-coverage.sh": {Path: "services/api/scripts/check-coverage.sh", Content: cleanCoverageScript},
		"services/api/scripts/check-dry.sh":      {Path: "services/api/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/api/scripts/check-crap.sh":     {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/api/scripts/check-mutation.sh": {Path: "services/api/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  api:
    steps:
      - run: cd services/api && go test ./...
      - run: cd services/api && go vet ./...
      - run: cd services/api && golangci-lint run
`,
		},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	assertRuleIDs(t, report.Findings, []string{
		GoLintRequiredRuleID,
		GoVetRequiredRuleID,
	})
}

func TestGoRulesKeepJobDefaultWorkingDirectoryWithWorkflowSteps(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                                    {Path: "README.md"},
		"AGENTS.md":                                    {Path: "AGENTS.md"},
		"services/api/go.mod":                          {Path: "services/api/go.mod"},
		"services/api/main.go":                         {Path: "services/api/main.go"},
		"services/api/.golangci.yml":                   {Path: "services/api/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/api/scripts/check-go-coverage.sh":    {Path: "services/api/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/api/scripts/check-dry.sh":            {Path: "services/api/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/api/scripts/check-crap.sh":           {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/api/scripts/check-mutation.sh":       {Path: "services/api/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		"services/worker/go.mod":                       {Path: "services/worker/go.mod"},
		"services/worker/main.go":                      {Path: "services/worker/main.go"},
		"services/worker/.golangci.yml":                {Path: "services/worker/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/worker/scripts/check-go-coverage.sh": {Path: "services/worker/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/worker/scripts/check-dry.sh":         {Path: "services/worker/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/worker/scripts/check-crap.sh":        {Path: "services/worker/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/worker/scripts/check-mutation.sh":    {Path: "services/worker/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  api:
    defaults:
      run:
        working-directory: services/api
    steps:
      - run: go test ./...
      - run: go vet ./...
      - run: golangci-lint run
  worker:
    defaults:
      run:
        working-directory: services/worker
    steps:
      - run: go test ./...
      - run: go vet ./...
      - run: golangci-lint run
`,
		},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoRulesIgnoreWorkflowListsBeforeSteps(t *testing.T) {
	blocks := workflowStepBlocks(`name: CI
jobs:
  go:
    strategy:
      matrix:
        include:
          - go: "1.23"
    defaults:
      run:
        working-directory: go
    steps:
      - run: go test ./...
      - run: go vet ./...
      - run: golangci-lint run
`)

	if len(blocks) != 3 {
		t.Fatalf("len(blocks) = %d, want 3: %#v", len(blocks), blocks)
	}
	for _, block := range blocks {
		if !strings.Contains(block, "working-directory: go") {
			t.Fatalf("block missing working directory context: %q", block)
		}
		if strings.Contains(block, `- go: "1.23"`) {
			t.Fatalf("block included matrix list item before steps: %q", block)
		}
	}
}

func TestGoRulesKeepTopLevelDefaultWorkingDirectoryWithWorkflowSteps(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                       {Path: "README.md"},
		"AGENTS.md":                       {Path: "AGENTS.md"},
		"go/go.mod":                       {Path: "go/go.mod"},
		"go/main.go":                      {Path: "go/main.go"},
		"go/.golangci.yml":                {Path: "go/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"go/scripts/check-go-coverage.sh": {Path: "go/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"go/scripts/check-dry.sh":         {Path: "go/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"go/scripts/check-crap.sh":        {Path: "go/scripts/check-crap.sh", Content: cleanCRAPScript},
		"go/scripts/check-mutation.sh":    {Path: "go/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
defaults:
  run:
    working-directory: go
jobs:
  test:
    steps:
      - run: go test ./...
      - run: go vet ./...
      - run: golangci-lint run
`,
		},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoRulesIgnoreWorkflowListsBeforeJobs(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                       {Path: "README.md"},
		"AGENTS.md":                       {Path: "AGENTS.md"},
		"go/go.mod":                       {Path: "go/go.mod"},
		"go/main.go":                      {Path: "go/main.go"},
		"go/.golangci.yml":                {Path: "go/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"go/scripts/check-go-coverage.sh": {Path: "go/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"go/scripts/check-dry.sh":         {Path: "go/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"go/scripts/check-crap.sh":        {Path: "go/scripts/check-crap.sh", Content: cleanCRAPScript},
		"go/scripts/check-mutation.sh":    {Path: "go/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
on:
  push:
    branches:
      - main
defaults:
  run:
    working-directory: go
jobs:
  test:
    steps:
      - run: go test ./...
      - run: go vet ./...
      - run: golangci-lint run
`,
		},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoRulesScopeSingleNestedModuleWorkflowEvidence(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: go test ./...
      - run: go vet ./...
      - run: golangci-lint run
`,
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	assertRuleIDs(t, report.Findings, []string{
		GoLintRequiredRuleID,
		GoVetRequiredRuleID,
	})
}

func TestGoRulesAcceptRootGuardrailsForNestedModule(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":     {Path: "README.md"},
		"AGENTS.md":     {Path: "AGENTS.md"},
		"go/go.mod":     {Path: "go/go.mod"},
		"go/main.go":    {Path: "go/main.go"},
		".golangci.yml": {Path: ".golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"Makefile": {
			Path: "Makefile",
			Content: `test:
	cd go && go test ./...
vet:
	cd go && go vet ./...
lint:
	cd go && golangci-lint run
`,
		},
		"scripts/check-go-coverage.sh": {
			Path: "scripts/check-go-coverage.sh",
			Content: `cd go
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
awk -v total="80" -v minimum="80" 'BEGIN { exit !(total + 0 >= minimum + 0) }'
`,
		},
		"scripts/check-dry.sh": {
			Path:    "scripts/check-dry.sh",
			Content: "cd go && go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n",
		},
		"scripts/check-crap.sh": {
			Path: "scripts/check-crap.sh",
			Content: `cd go
maximum_crap_score="30"
go run github.com/unclebob/crap4go/cmd/crap4go@latest
awk -v score="0" -v maximum="$maximum_crap_score" 'BEGIN { exit !(score + 0 <= maximum + 0) }'
`,
		},
		"scripts/check-mutation.sh": {
			Path:    "scripts/check-mutation.sh",
			Content: "cd go && go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n",
		},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: make test
      - run: make vet
      - run: make lint
      - run: scripts/check-go-coverage.sh
      - run: scripts/check-dry.sh
      - run: scripts/check-crap.sh
      - run: scripts/check-mutation.sh
`,
		},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoRulesPreferModuleLocalConfigOverRepoRootConfig(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		".golangci.yml": {
			Path:    ".golangci.yml",
			Content: "linters:\n  enable:\n    - errcheck\n",
		},
		"go/.golangci.yml": {
			Path:    "go/.golangci.yml",
			Content: "linters:\n  enable:\n    - cyclop\n",
		},
	})

	for range 100 {
		report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())
		if !report.OK {
			t.Fatalf("report.OK = false, findings = %#v", report.Findings)
		}
	}
}

func TestGoRulesDoNotUseRepoRootConfigWhenModuleLocalConfigExists(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		".golangci.yml": {
			Path:    ".golangci.yml",
			Content: "linters:\n  enable:\n    - cyclop\n",
		},
		"go/.golangci.yaml": {
			Path:    "go/.golangci.yaml",
			Content: "linters:\n  enable:\n    - errcheck\n",
		},
	})
	delete(files, "go/.golangci.yml")

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	assertRuleIDs(t, report.Findings, []string{GoComplexityRequiredRuleID})
}

func TestGoRulesDoNotCarryMakefileScopeAcrossTargets(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":     {Path: "README.md"},
		"AGENTS.md":     {Path: "AGENTS.md"},
		"go/go.mod":     {Path: "go/go.mod"},
		"go/main.go":    {Path: "go/main.go"},
		".golangci.yml": {Path: ".golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"Makefile": {
			Path: "Makefile",
			Content: `test:
	cd go && go test ./...
vet:
	go vet ./...
lint:
	cd go && golangci-lint run
`,
		},
		"scripts/check-go-coverage.sh": {
			Path: "scripts/check-go-coverage.sh",
			Content: `cd go
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
awk -v total="80" -v minimum="80" 'BEGIN { exit !(total + 0 >= minimum + 0) }'
`,
		},
		"scripts/check-dry.sh": {
			Path:    "scripts/check-dry.sh",
			Content: "cd go && go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n",
		},
		"scripts/check-crap.sh": {
			Path: "scripts/check-crap.sh",
			Content: `cd go
maximum_crap_score="30"
go run github.com/unclebob/crap4go/cmd/crap4go@latest
awk -v score="0" -v maximum="$maximum_crap_score" 'BEGIN { exit !(score + 0 <= maximum + 0) }'
`,
		},
		"scripts/check-mutation.sh": {
			Path:    "scripts/check-mutation.sh",
			Content: "cd go && go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n",
		},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: make test
      - run: make vet
      - run: make lint
`,
		},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	assertRuleIDs(t, report.Findings, []string{GoVetRequiredRuleID})
}

func TestGoRulesScopeRootCommandFilesPerModule(t *testing.T) {
	coverageScript := strings.ReplaceAll(cleanCoverageScript, "go test -coverprofile=coverage.out ./...", "go test -coverprofile=coverage.out ./internal/...")
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                              {Path: "README.md"},
		"AGENTS.md":                              {Path: "AGENTS.md"},
		"services/api/go.mod":                    {Path: "services/api/go.mod"},
		"services/api/main.go":                   {Path: "services/api/main.go"},
		"services/api/.golangci.yml":             {Path: "services/api/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/api/scripts/check-coverage.sh": {Path: "services/api/scripts/check-coverage.sh", Content: coverageScript},
		"services/api/scripts/check-dry.sh":      {Path: "services/api/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/api/scripts/check-crap.sh":     {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/api/scripts/check-mutation.sh": {Path: "services/api/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		"services/worker/go.mod":                 {Path: "services/worker/go.mod"},
		"services/worker/main.go":                {Path: "services/worker/main.go"},
		"services/worker/.golangci.yml":          {Path: "services/worker/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/worker/scripts/check-coverage.sh": {
			Path:    "services/worker/scripts/check-coverage.sh",
			Content: coverageScript,
		},
		"services/worker/scripts/check-dry.sh": {
			Path:    "services/worker/scripts/check-dry.sh",
			Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n",
		},
		"services/worker/scripts/check-crap.sh": {
			Path:    "services/worker/scripts/check-crap.sh",
			Content: cleanCRAPScript,
		},
		"services/worker/scripts/check-mutation.sh": {
			Path:    "services/worker/scripts/check-mutation.sh",
			Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n",
		},
		"Makefile": {
			Path: "Makefile",
			Content: `api-test:
	cd services/api && go test ./...
api-lint:
	cd services/api && golangci-lint run
worker-vet:
	cd services/worker && go vet ./...
worker-lint:
	cd services/worker && golangci-lint run
`,
		},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: make api-test
      - run: make api-lint
      - run: make worker-vet
      - run: make worker-lint
`,
		},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	assertRuleIDs(t, report.Findings, []string{
		GoTestsRequiredRuleID,
		GoVetRequiredRuleID,
	})
}

func TestScopedRootCommandContentStopsAfterLeavingModule(t *testing.T) {
	content := `cd go
go test ./...
cd ..
go vet ./...
dry4go .
`

	scoped, ok := scopedRootCommandContent("scripts/check.sh", content, "go", []string{"go"})

	if !ok {
		t.Fatal("scopedRootCommandContent returned ok=false")
	}
	if strings.Contains(scoped, "go vet ./...") || strings.Contains(scoped, "dry4go .") {
		t.Fatalf("scoped content kept commands after leaving module: %q", scoped)
	}
	if !strings.Contains(scoped, "go test ./...") {
		t.Fatalf("scoped content lost in-module command: %q", scoped)
	}
}

func TestGoLintRuleRequiresRunCommand(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
defaults:
  run:
    working-directory: go
jobs:
  test:
    steps:
      - run: go test ./...
      - run: go vet ./...
      - run: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.0
      - run: ./scripts/check-go-coverage.sh
`,
		},
		"go/scripts/check-go-coverage.sh": {
			Path:    "go/scripts/check-go-coverage.sh",
			Content: strings.ReplaceAll(cleanCoverageScript, "go test -coverprofile=coverage.out ./...", "go test -coverprofile=coverage.out ./internal/..."),
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	assertRuleIDs(t, report.Findings, []string{GoLintRequiredRuleID})
}

func TestGoRulesKeepNamedWorkflowStepsForModule(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                                    {Path: "README.md"},
		"AGENTS.md":                                    {Path: "AGENTS.md"},
		"services/api/go.mod":                          {Path: "services/api/go.mod"},
		"services/api/main.go":                         {Path: "services/api/main.go"},
		"services/api/.golangci.yml":                   {Path: "services/api/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/api/scripts/check-go-coverage.sh":    {Path: "services/api/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/api/scripts/check-dry.sh":            {Path: "services/api/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/api/scripts/check-crap.sh":           {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/api/scripts/check-mutation.sh":       {Path: "services/api/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		"services/worker/go.mod":                       {Path: "services/worker/go.mod"},
		"services/worker/main.go":                      {Path: "services/worker/main.go"},
		"services/worker/.golangci.yml":                {Path: "services/worker/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/worker/scripts/check-go-coverage.sh": {Path: "services/worker/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/worker/scripts/check-dry.sh":         {Path: "services/worker/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/worker/scripts/check-crap.sh":        {Path: "services/worker/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/worker/scripts/check-mutation.sh":    {Path: "services/worker/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  api:
    steps:
      - name: api tests
        run: cd services/api && go test ./...
      - name: api vet
        run: cd services/api && go vet ./...
      - name: api lint
        run: cd services/api && golangci-lint run
  worker:
    steps:
      - name: worker lint
        run: cd services/worker && golangci-lint run
      - name: marker
        run: echo services/worker
`,
		},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	assertRuleIDs(t, report.Findings, []string{GoVetRequiredRuleID})
}

func TestGoRulesDoNotTreatGoCommandAsGoModuleRoot(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                        {Path: "README.md"},
		"AGENTS.md":                        {Path: "AGENTS.md"},
		"go/go.mod":                        {Path: "go/go.mod"},
		"go/main.go":                       {Path: "go/main.go"},
		"go/.golangci.yml":                 {Path: "go/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"go/scripts/check-go-coverage.sh":  {Path: "go/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"go/scripts/check-dry.sh":          {Path: "go/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"go/scripts/check-crap.sh":         {Path: "go/scripts/check-crap.sh", Content: cleanCRAPScript},
		"go/scripts/check-mutation.sh":     {Path: "go/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		"api/go.mod":                       {Path: "api/go.mod"},
		"api/main.go":                      {Path: "api/main.go"},
		"api/.golangci.yml":                {Path: "api/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"api/scripts/check-go-coverage.sh": {Path: "api/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"api/scripts/check-dry.sh":         {Path: "api/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"api/scripts/check-crap.sh":        {Path: "api/scripts/check-crap.sh", Content: cleanCRAPScript},
		"api/scripts/check-mutation.sh":    {Path: "api/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  api:
    steps:
      - run: cd api && go test ./...
      - run: cd api && go vet ./...
      - run: cd api && golangci-lint run
`,
		},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

	assertRuleIDs(t, report.Findings, []string{
		GoLintRequiredRuleID,
		GoVetRequiredRuleID,
	})
}

func TestGoRulesDetectRootGoSourceWithNestedModule(t *testing.T) {
	files := map[string]repo.File{
		"README.md":                  {Path: "README.md"},
		"AGENTS.md":                  {Path: "AGENTS.md"},
		"main.go":                    {Path: "main.go"},
		"services/api/go.mod":        {Path: "services/api/go.mod"},
		"services/api/main.go":       {Path: "services/api/main.go"},
		"services/api/.golangci.yml": {Path: "services/api/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/api/scripts/check-go-coverage.sh": {Path: "services/api/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/api/scripts/check-dry.sh":         {Path: "services/api/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"services/api/scripts/check-crap.sh":        {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript},
		"services/api/scripts/check-mutation.sh":    {Path: "services/api/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan\n"},
		".github/workflows/ci.yml":                  {Path: ".github/workflows/ci.yml", Content: nestedGoWorkflow},
	}

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	assertRuleIDs(t, report.Findings, []string{
		GoComplexityRequiredRuleID,
		GoCoverageRequiredRuleID,
		GoCRAPRequiredRuleID,
		GoDryRequiredRuleID,
		GoLintRequiredRuleID,
		GoModuleRequiredRuleID,
		GoMutationRequiredRuleID,
		GoTestsRequiredRuleID,
		GoVetRequiredRuleID,
	})
}

func TestGoCRAPRuleRequiresThreshold(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		"go/scripts/check-crap.sh": {
			Path:    "go/scripts/check-crap.sh",
			Content: "go run github.com/unclebob/crap4go/cmd/crap4go@latest\n",
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	assertRuleIDs(t, report.Findings, []string{GoCRAPRequiredRuleID})
}

func TestGoCRAPRuleRejectsReportRedirectionWithoutThreshold(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		"go/scripts/check-crap.sh": {
			Path:    "go/scripts/check-crap.sh",
			Content: "go run github.com/unclebob/crap4go/cmd/crap4go@latest > crap-report.txt\n",
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	assertRuleIDs(t, report.Findings, []string{GoCRAPRequiredRuleID})
}

func TestGoCRAPRuleRequiresThresholdInSameCheck(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
defaults:
  run:
    working-directory: go
jobs:
  test:
    steps:
      - run: go test ./...
      - run: go vet ./...
      - run: go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.0 run
      - run: ./scripts/check-go-coverage.sh
      - run: crap4go .
      - run: echo "minimum coverage >= 80"
`,
		},
		"go/scripts/check-crap.sh": {
			Path:    "go/scripts/check-crap.sh",
			Content: "go run github.com/unclebob/crap4go/cmd/crap4go@latest\n",
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	assertRuleIDs(t, report.Findings, []string{GoCRAPRequiredRuleID})
}

func TestGoToolRulesAcceptSlophammerGoCommands(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
defaults:
  run:
    working-directory: go
jobs:
  test:
    steps:
      - run: go test ./...
      - run: go vet ./...
      - run: golangci-lint run
      - run: ./scripts/check-go-coverage.sh
      - run: go run ./cmd/slophammer go dry . --max-candidates 40
      - run: go run ./cmd/slophammer go crap . --max-score 30
      - run: go run ./cmd/slophammer go mutate . --target internal/rules/rules.go --scan
`,
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoMutationRuleRequiresTargetForSlophammerCommand(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: go run ./cmd/slophammer go mutate . --scan
`,
		},
	})

	if hasMutate4GoCommand(snapshot) {
		t.Fatal("hasMutate4GoCommand = true, want false without --target")
	}
}

func TestWorkflowCommandSectionsIgnoreStepMetadata(t *testing.T) {
	sections := commandSections(repo.File{
		Path: ".github/workflows/ci.yml",
		Content: `name: CI
jobs:
  test:
    steps:
      - name: go test ./...
        run: echo ok
      - name: go vet ./...
        run: |
          echo still ok
`,
	})

	joined := strings.Join(sections, "\n")
	if strings.Contains(joined, "go test ./...") || strings.Contains(joined, "go vet ./...") {
		t.Fatalf("workflow metadata leaked into command sections: %q", joined)
	}
	if !strings.Contains(joined, "echo ok") || !strings.Contains(joined, "echo still ok") {
		t.Fatalf("workflow run content missing from command sections: %q", joined)
	}
}

func TestWorkflowCommandSectionsFoldRunBlocks(t *testing.T) {
	sections := commandSections(repo.File{
		Path: ".github/workflows/ci.yml",
		Content: `name: CI
jobs:
  test:
    steps:
      - run: >
          go vet
          ./...
`,
	})

	if len(sections) != 1 || sections[0] != "go vet ./..." {
		t.Fatalf("workflow folded run sections = %#v", sections)
	}
}

func TestGoCommandRulesIgnoreWorkflowStepMetadata(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - name: go test ./...
        run: echo no tests
      - name: go vet ./...
        run: echo no vet
      - name: golangci-lint run
        run: echo no lint
`,
		},
	})

	checks := []struct {
		name  string
		check func(repo.Snapshot) bool
	}{
		{name: "go test", check: hasGoTestCommand},
		{name: "go vet", check: hasGoVetCommand},
		{name: "golangci-lint", check: hasGolangCICommand},
	}
	for _, check := range checks {
		if check.check(snapshot) {
			t.Fatalf("%s command accepted workflow step metadata", check.name)
		}
	}
}

func TestGoCommandRulesIgnoreEchoedCommands(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
defaults:
  run:
    working-directory: go
jobs:
  test:
    steps:
      - run: echo "go test ./..."
      - run: echo "go vet ./..."
      - run: echo "golangci-lint run"
      - run: ./scripts/check-go-coverage.sh
`,
		},
		"go/scripts/check-go-coverage.sh": {
			Path:    "go/scripts/check-go-coverage.sh",
			Content: strings.ReplaceAll(cleanCoverageScript, "go test -coverprofile=coverage.out ./...", "go test -coverprofile=coverage.out ./internal/..."),
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	assertRuleIDs(t, report.Findings, []string{
		GoLintRequiredRuleID,
		GoTestsRequiredRuleID,
		GoVetRequiredRuleID,
	})
}

func TestGoToolCommandDetectionRequiresRunnableCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		tool    gotools.Tool
		want    bool
	}{
		{
			name:    "dry package run",
			command: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .",
			tool:    gotools.Dry4Go,
			want:    true,
		},
		{
			name:    "dry package run with flag",
			command: "go run -mod=readonly github.com/unclebob/dry4go/cmd/dry4go@latest --format json .",
			tool:    gotools.Dry4Go,
			want:    true,
		},
		{
			name:    "dry package run with flag value",
			command: "go run -mod readonly github.com/unclebob/dry4go/cmd/dry4go@latest --format json .",
			tool:    gotools.Dry4Go,
			want:    true,
		},
		{
			name:    "dry install only",
			command: "go install github.com/unclebob/dry4go/cmd/dry4go@latest",
			tool:    gotools.Dry4Go,
		},
		{
			name:    "dry install then run",
			command: "go install github.com/unclebob/dry4go/cmd/dry4go@latest && dry4go .",
			tool:    gotools.Dry4Go,
			want:    true,
		},
		{
			name:    "dry after semicolon",
			command: "cd go; dry4go .",
			tool:    gotools.Dry4Go,
			want:    true,
		},
		{
			name:    "dry after env assignment",
			command: "DRY_CACHE=/tmp dry4go .",
			tool:    gotools.Dry4Go,
			want:    true,
		},
		{
			name:    "dry echo only",
			command: "echo go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .",
			tool:    gotools.Dry4Go,
		},
		{
			name:    "crap binary run",
			command: "crap4go .",
			tool:    gotools.CRAP4Go,
			want:    true,
		},
		{
			name:    "crap package run with flag",
			command: "go run -mod=readonly github.com/unclebob/crap4go/cmd/crap4go@latest",
			tool:    gotools.CRAP4Go,
			want:    true,
		},
		{
			name:    "crap echo only",
			command: "echo crap4go",
			tool:    gotools.CRAP4Go,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contentHasGoToolCommand(tt.command, tt.tool); got != tt.want {
				t.Fatalf("contentHasGoToolCommand(%q) = %t, want %t", tt.command, got, tt.want)
			}
		})
	}
}

func TestGoMutationRuleRequiresTargetForDirectMutate4Go(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{name: "package target", command: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan", want: true},
		{name: "binary target", command: "mutate4go main.go --scan", want: true},
		{name: "flag before target", command: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest --scan internal/rules/rules.go", want: true},
		{name: "go run flag before package", command: "go run -mod=readonly github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan", want: true},
		{name: "install then run", command: "go install github.com/unclebob/mutate4go/cmd/mutate4go@latest && mutate4go main.go --scan", want: true},
		{name: "package after semicolon", command: "cd go; go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan", want: true},
		{name: "binary after semicolon", command: "cd go; mutate4go main.go --scan", want: true},
		{name: "binary after env assignment", command: "MUTATE_CACHE=/tmp mutate4go main.go --scan", want: true},
		{name: "package missing target", command: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest --scan"},
		{name: "binary missing target", command: "mutate4go --scan"},
		{name: "install only", command: "go install github.com/unclebob/mutate4go/cmd/mutate4go@latest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contentHasDirectMutate4GoCommand(tt.command); got != tt.want {
				t.Fatalf("contentHasDirectMutate4GoCommand(%q) = %t, want %t", tt.command, got, tt.want)
			}
		})
	}
}

func TestSlophammerGoCommandRequiredFlagsNeedValues(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		subcommand  string
		required    string
		expectMatch bool
	}{
		{name: "crap value", command: "slophammer go crap . --max-score 30", subcommand: "crap", required: "--max-score", expectMatch: true},
		{name: "crap after semicolon", command: "cd go; slophammer go crap . --max-score 30", subcommand: "crap", required: "--max-score", expectMatch: true},
		{name: "crap missing value", command: "slophammer go crap . --max-score", subcommand: "crap", required: "--max-score"},
		{name: "crap flag after separator", command: "slophammer go crap . && echo --max-score 30", subcommand: "crap", required: "--max-score"},
		{name: "mutate value", command: "slophammer go mutate . --target main.go --scan", subcommand: "mutate", required: "--target", expectMatch: true},
		{name: "mutate after semicolon", command: "cd go; slophammer go mutate . --target main.go --scan", subcommand: "mutate", required: "--target", expectMatch: true},
		{name: "mutate missing value", command: "slophammer go mutate . --target --scan", subcommand: "mutate", required: "--target"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contentHasSlophammerGoCommand(tt.command, tt.subcommand, tt.required)
			if got != tt.expectMatch {
				t.Fatalf("contentHasSlophammerGoCommand(%q) = %t, want %t", tt.command, got, tt.expectMatch)
			}
		})
	}
}

func TestGoCoverageRuleRequiresCoverageOutputAndCoverTool(t *testing.T) {
	tests := []struct {
		name          string
		coverageCheck string
	}{
		{name: "missing cover tool", coverageCheck: "go test -coverprofile=coverage.out ./...\n"},
		{name: "missing cover profile", coverageCheck: "go tool cover -func=coverage.out\n"},
		{name: "missing threshold", coverageCheck: "go test -coverprofile=coverage.out ./...\ngo tool cover -func=coverage.out\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := cleanGoGuardrailFiles(map[string]repo.File{
				"go/scripts/check-go-coverage.sh": {
					Path:    "go/scripts/check-go-coverage.sh",
					Content: tt.coverageCheck,
				},
			})

			report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

			assertRuleIDs(t, report.Findings, []string{GoCoverageRequiredRuleID})
		})
	}
}

func TestGoCoverageRuleRejectsReportRedirectionWithoutThreshold(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		"go/scripts/check-go-coverage.sh": {
			Path: "go/scripts/check-go-coverage.sh",
			Content: `go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out > coverage.html
`,
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	assertRuleIDs(t, report.Findings, []string{GoCoverageRequiredRuleID})
}

func TestGoCoverageRuleRequiresThresholdInSameCheck(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
defaults:
  run:
    working-directory: go
jobs:
  test:
    steps:
      - run: go test ./...
      - run: go vet ./...
      - run: go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.0 run
      - run: |
          go test -coverprofile=coverage.out ./...
          go tool cover -html=coverage.out > coverage.html
      - run: echo "minimum node version >= 20"
`,
		},
		"go/scripts/check-go-coverage.sh": {
			Path: "go/scripts/check-go-coverage.sh",
			Content: `go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out > coverage.html
`,
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	assertRuleIDs(t, report.Findings, []string{GoCoverageRequiredRuleID})
}

func TestGoCoverageRuleRequiresEvidenceInSameCheck(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
defaults:
  run:
    working-directory: go
jobs:
  test:
    steps:
      - run: go test ./...
      - run: go vet ./...
      - run: golangci-lint run
      - run: ./scripts/check-go-coverage.sh
      - run: echo "minimum node version >= 20"
`,
		},
		"go/scripts/check-go-coverage.sh": {
			Path: "go/scripts/check-go-coverage.sh",
			Content: `go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out > coverage.html
`,
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	assertRuleIDs(t, report.Findings, []string{GoCoverageRequiredRuleID})
}

func TestGoThresholdPatternsAcceptStrictComparisons(t *testing.T) {
	if !hasCoverageThreshold(`awk -v total="$total" -v minimum="80" 'BEGIN { exit (total + 0 < minimum + 0) }'`) {
		t.Fatal("coverage threshold pattern rejected strict minimum comparison")
	}

	if !hasCRAPThreshold(`awk -v score="0" -v maximum="30" 'BEGIN { exit (score + 0 > maximum + 0) }'`) {
		t.Fatal("CRAP threshold pattern rejected strict maximum comparison")
	}
}

func TestGoCoverageRuleAcceptsCommonThresholdNames(t *testing.T) {
	tests := []struct {
		name          string
		coverageCheck string
	}{
		{
			name: "threshold variable",
			coverageCheck: `go test -coverprofile=coverage.out ./...
total="$(go tool cover -func=coverage.out | awk '/^total:/ {print substr($3, 1, length($3)-1)}')"
awk -v total="$total" -v threshold="80" 'BEGIN { exit !(total + 0 >= threshold + 0) }'
`,
		},
		{
			name: "literal threshold",
			coverageCheck: `go test -coverprofile=coverage.out ./...
total="$(go tool cover -func=coverage.out | awk '/^total:/ {print substr($3, 1, length($3)-1)}')"
awk -v total="$total" 'BEGIN { exit !(total + 0 >= 80) }'
`,
		},
		{
			name: "strict threshold",
			coverageCheck: `go test -coverprofile=coverage.out ./...
total="$(go tool cover -func=coverage.out | awk '/^total:/ {print substr($3, 1, length($3)-1)}')"
awk -v total="$total" -v minimum="80" 'BEGIN { exit (total + 0 < minimum + 0) }'
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := cleanGoGuardrailFiles(map[string]repo.File{
				"go/scripts/check-go-coverage.sh": {
					Path:    "go/scripts/check-go-coverage.sh",
					Content: tt.coverageCheck,
				},
			})

			report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

			if !report.OK {
				t.Fatalf("report.OK = false, findings = %#v", report.Findings)
			}
		})
	}
}

func TestGoComplexityRuleRequiresEnabledLinter(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{
			name: "disabled",
			config: `linters:
  disable:
    - cyclop
`,
		},
		{
			name: "settings only",
			config: `linters:
  settings:
    cyclop:
      max-complexity: 10
`,
		},
		{
			name: "comment only",
			config: `linters:
  enable:
    - errcheck
# cyclop belongs in enable, not comments.
`,
		},
		{
			name: "enabled and disabled",
			config: `linters:
  enable:
    - cyclop
  disable:
    - cyclop
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := cleanGoGuardrailFiles(map[string]repo.File{
				"go/.golangci.yml": {Path: "go/.golangci.yml", Content: tt.config},
			})

			report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

			assertRuleIDs(t, report.Findings, []string{GoComplexityRequiredRuleID})
		})
	}
}

func TestGoComplexityRuleAcceptsDefaultAll(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		"go/.golangci.yml": {
			Path:    "go/.golangci.yml",
			Content: "linters:\n  default: all\n",
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoComplexityRuleRejectsDefaultAllWhenAllComplexityLintersDisabled(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		"go/.golangci.yml": {
			Path: "go/.golangci.yml",
			Content: `linters:
  default: all
  disable:
    - cyclop
    - gocognit
    - gocyclo
`,
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	assertRuleIDs(t, report.Findings, []string{GoComplexityRequiredRuleID})
}

func TestExplainKnownRule(t *testing.T) {
	got, ok := Explain(DefaultRules(), AgentsRequiredRuleID)
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

func TestDefaultDefinitionsAreStable(t *testing.T) {
	definitions := DefaultDefinitions()
	wantIDs := []string{
		ReadmeRequiredRuleID,
		AgentsRequiredRuleID,
		CIRequiredRuleID,
		GoModuleRequiredRuleID,
		GoTestsRequiredRuleID,
		GoVetRequiredRuleID,
		GoLintRequiredRuleID,
		GoCoverageRequiredRuleID,
		GoComplexityRequiredRuleID,
		GoDryRequiredRuleID,
		GoCRAPRequiredRuleID,
		GoMutationRequiredRuleID,
	}
	if len(definitions) != len(wantIDs) {
		t.Fatalf("len(definitions) = %d, want %d", len(definitions), len(wantIDs))
	}
	for i, want := range wantIDs {
		if definitions[i].ID != want {
			t.Fatalf("definition[%d].ID = %q, want %q", i, definitions[i].ID, want)
		}
		if definitions[i].Severity == "" {
			t.Fatalf("definition[%d].Severity is empty", i)
		}
		if definitions[i].Path == "" {
			t.Fatalf("definition[%d].Path is empty", i)
		}
		if definitions[i].Message == "" {
			t.Fatalf("definition[%d].Message is empty", i)
		}
		if definitions[i].Description == "" {
			t.Fatalf("definition[%d].Description is empty", i)
		}
	}
}

func TestDefaultDefinitionsMatchRulesSpec(t *testing.T) {
	specContent, err := os.ReadFile(filepath.Join(repoRoot(t), "specs", "rules.json"))
	if err != nil {
		t.Fatalf("read rules spec: %v", err)
	}

	var spec ruleSpecFile
	if err := json.Unmarshal(specContent, &spec); err != nil {
		t.Fatalf("unmarshal rules spec: %v", err)
	}

	definitions := DefaultDefinitions()
	if len(spec.Rules) != len(definitions) {
		t.Fatalf("len(spec.Rules) = %d, want %d", len(spec.Rules), len(definitions))
	}
	for i, definition := range definitions {
		got := spec.Rules[i]
		want := ruleSpec(definition)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("spec rule[%d] mismatch\ngot:  %#v\nwant: %#v", i, got, want)
		}
	}
}

func assertRuleIDs(t *testing.T, findings []Finding, want []string) {
	t.Helper()
	if len(findings) != len(want) {
		t.Fatalf("len(findings) = %d, want %d; findings = %#v", len(findings), len(want), findings)
	}
	for i, wantID := range want {
		if findings[i].RuleID != wantID {
			t.Fatalf("finding[%d].RuleID = %q, want %q", i, findings[i].RuleID, wantID)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller returned ok=false")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

type ruleSpecFile struct {
	Rules []ruleSpec `json:"rules"`
}

type ruleSpec struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Category    string   `json:"category"`
	Severity    Severity `json:"severity"`
	Path        string   `json:"path"`
	Message     string   `json:"message"`
	Description string   `json:"description"`
	Tool        string   `json:"tool,omitempty"`
	Status      string   `json:"status"`
}

func cleanGoGuardrailFiles(overrides map[string]repo.File) map[string]repo.File {
	files := map[string]repo.File{
		"README.md":                       {Path: "README.md"},
		"AGENTS.md":                       {Path: "AGENTS.md"},
		"go/go.mod":                       {Path: "go/go.mod"},
		"go/main.go":                      {Path: "go/main.go"},
		"go/.golangci.yml":                {Path: "go/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		".github/workflows/ci.yml":        {Path: ".github/workflows/ci.yml", Content: goCleanWorkflow},
		"go/scripts/check-go-coverage.sh": {Path: "go/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"go/scripts/check-dry.sh":         {Path: "go/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"go/scripts/check-crap.sh":        {Path: "go/scripts/check-crap.sh", Content: cleanCRAPScript},
		"go/scripts/check-mutation.sh":    {Path: "go/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest internal/rules/rules.go --scan\n"},
	}
	for path, file := range overrides {
		files[path] = file
	}
	return files
}

const cleanCoverageScript = `minimum_coverage="80"
go test -coverprofile=coverage.out ./...
total="$(go tool cover -func=coverage.out | awk '/^total:/ {print substr($3, 1, length($3)-1)}')"
awk -v total="$total" -v minimum="$minimum_coverage" 'BEGIN { exit !(total + 0 >= minimum + 0) }'
`

const cleanCRAPScript = `maximum_crap_score="30"
go run github.com/unclebob/crap4go/cmd/crap4go@latest
awk -v score="0" -v maximum="$maximum_crap_score" 'BEGIN { exit !(score + 0 <= maximum + 0) }'
`

const goCleanWorkflow = `name: CI

defaults:
  run:
    working-directory: go

jobs:
  test:
    steps:
      - run: go test ./...
      - run: go vet ./...
      - run: go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.0 run
      - run: ./scripts/check-go-coverage.sh
`

const nestedGoWorkflow = `name: CI

jobs:
  test:
    steps:
      - run: cd services/api && go test ./...
      - run: cd services/api && go vet ./...
      - run: cd services/api && golangci-lint run
      - run: services/api/scripts/check-coverage.sh
      - run: services/api/scripts/check-dry.sh
      - run: services/api/scripts/check-crap.sh
      - run: services/api/scripts/check-mutation.sh
`
