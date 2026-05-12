package rules

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
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

func TestGoCoverageRuleRequiresCoverageOutputAndCoverTool(t *testing.T) {
	tests := []struct {
		name          string
		coverageCheck string
	}{
		{name: "missing cover tool", coverageCheck: "go test -coverprofile=coverage.out ./...\n"},
		{name: "missing cover profile", coverageCheck: "go tool cover -func=coverage.out\n"},
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
		"go/scripts/check-go-coverage.sh": {Path: "go/scripts/check-go-coverage.sh", Content: "go test -coverprofile=coverage.out ./...\ngo tool cover -func=coverage.out\n"},
		"go/scripts/check-dry.sh":         {Path: "go/scripts/check-dry.sh", Content: "go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json .\n"},
		"go/scripts/check-crap.sh":        {Path: "go/scripts/check-crap.sh", Content: "go run github.com/unclebob/crap4go/cmd/crap4go@latest\n"},
		"go/scripts/check-mutation.sh":    {Path: "go/scripts/check-mutation.sh", Content: "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest internal/rules/rules.go --scan\n"},
	}
	for path, file := range overrides {
		files[path] = file
	}
	return files
}

const goCleanWorkflow = `name: CI

jobs:
  test:
    steps:
      - run: go test ./...
      - run: go vet ./...
      - run: go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.0 run
      - run: ./scripts/check-go-coverage.sh
`
