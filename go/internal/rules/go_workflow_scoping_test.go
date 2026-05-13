package rules

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/dutifuldev/slophammer/go/internal/repo"
)

func TestGoRulesScopeGoCFlagCommandsToNestedModule(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                       {Path: "README.md"},
		"AGENTS.md":                       {Path: "AGENTS.md"},
		"go/go.mod":                       {Path: "go/go.mod"},
		"go/main.go":                      {Path: "go/main.go"},
		"go/.golangci.yml":                {Path: "go/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"go/scripts/check-go-coverage.sh": {Path: "go/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"go/scripts/check-dry.sh":         {Path: "go/scripts/check-dry.sh", Content: cleanDryScript()},
		"go/scripts/check-crap.sh":        {Path: "go/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"go/scripts/check-mutation.sh":    {Path: "go/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: go -C go test ./...
      - run: go -C=go vet ./...
      - run: go -C go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.0 run
`,
		},
	})

	report := Run(context.Background(), snapshot, DefaultRules())

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
		"services/api/scripts/check-dry.sh":            {Path: "services/api/scripts/check-dry.sh", Content: cleanDryScript()},
		"services/api/scripts/check-crap.sh":           {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"services/api/scripts/check-mutation.sh":       {Path: "services/api/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
		"services/worker/go.mod":                       {Path: "services/worker/go.mod"},
		"services/worker/main.go":                      {Path: "services/worker/main.go"},
		"services/worker/.golangci.yml":                {Path: "services/worker/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/worker/scripts/check-go-coverage.sh": {Path: "services/worker/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/worker/scripts/check-dry.sh":         {Path: "services/worker/scripts/check-dry.sh", Content: cleanDryScript()},
		"services/worker/scripts/check-crap.sh":        {Path: "services/worker/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"services/worker/scripts/check-mutation.sh":    {Path: "services/worker/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
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
		"services/api/scripts/check-dry.sh":            {Path: "services/api/scripts/check-dry.sh", Content: cleanDryScript()},
		"services/api/scripts/check-crap.sh":           {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"services/api/scripts/check-mutation.sh":       {Path: "services/api/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
		"services/worker/go.mod":                       {Path: "services/worker/go.mod"},
		"services/worker/main.go":                      {Path: "services/worker/main.go"},
		"services/worker/.golangci.yml":                {Path: "services/worker/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/worker/scripts/check-go-coverage.sh": {Path: "services/worker/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/worker/scripts/check-dry.sh":         {Path: "services/worker/scripts/check-dry.sh", Content: cleanDryScript()},
		"services/worker/scripts/check-crap.sh":        {Path: "services/worker/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"services/worker/scripts/check-mutation.sh":    {Path: "services/worker/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
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
		"services/api/scripts/check-dry.sh":            {Path: "services/api/scripts/check-dry.sh", Content: cleanDryScript()},
		"services/api/scripts/check-crap.sh":           {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"services/api/scripts/check-mutation.sh":       {Path: "services/api/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
		"services/worker/go.mod":                       {Path: "services/worker/go.mod"},
		"services/worker/main.go":                      {Path: "services/worker/main.go"},
		"services/worker/.golangci.yml":                {Path: "services/worker/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/worker/scripts/check-go-coverage.sh": {Path: "services/worker/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/worker/scripts/check-dry.sh":         {Path: "services/worker/scripts/check-dry.sh", Content: cleanDryScript()},
		"services/worker/scripts/check-crap.sh":        {Path: "services/worker/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"services/worker/scripts/check-mutation.sh":    {Path: "services/worker/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
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
	block := `      - run: |-
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
	if !strings.Contains(scoped, "- run: |-") ||
		!strings.Contains(scoped, "cd services/api") ||
		!strings.Contains(scoped, "go vet ./...") ||
		!strings.Contains(scoped, "golangci-lint run") {
		t.Fatalf("scoped block lost api run content: %q", scoped)
	}
}

func TestGoRulesAcceptChompedMixedModuleWorkflowRunBlocks(t *testing.T) {
	workflow := `name: CI
jobs:
  all:
    steps:
      - run: |-
          cd services/api
          go test ./...
          go vet ./...
          golangci-lint run
          cd services/worker
          go test ./...
          go vet ./...
          golangci-lint run
`
	snapshot := repo.NewSnapshot("/repo", cleanTwoModuleGoGuardrailFiles(workflow))

	report := Run(context.Background(), snapshot, DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
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
		"scripts/check-dry.sh":                      {Path: "scripts/check-dry.sh", Content: cleanDryScript()},
		"scripts/check-crap.sh":                     {Path: "scripts/check-crap.sh", Content: cleanCRAPScript()},
		"scripts/check-mutation.sh":                 {Path: "scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
		"services/api/go.mod":                       {Path: "services/api/go.mod"},
		"services/api/main.go":                      {Path: "services/api/main.go"},
		"services/api/.golangci.yml":                {Path: "services/api/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/api/scripts/check-go-coverage.sh": {Path: "services/api/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/api/scripts/check-dry.sh":         {Path: "services/api/scripts/check-dry.sh", Content: cleanDryScript()},
		"services/api/scripts/check-crap.sh":        {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"services/api/scripts/check-mutation.sh":    {Path: "services/api/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
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
		"services/api/scripts/check-dry.sh":            {Path: "services/api/scripts/check-dry.sh", Content: cleanDryScript()},
		"services/api/scripts/check-crap.sh":           {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"services/api/scripts/check-mutation.sh":       {Path: "services/api/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
		"services/worker/go.mod":                       {Path: "services/worker/go.mod"},
		"services/worker/main.go":                      {Path: "services/worker/main.go"},
		"services/worker/.golangci.yml":                {Path: "services/worker/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/worker/scripts/check-go-coverage.sh": {Path: "services/worker/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/worker/scripts/check-dry.sh":         {Path: "services/worker/scripts/check-dry.sh", Content: cleanDryScript()},
		"services/worker/scripts/check-crap.sh":        {Path: "services/worker/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"services/worker/scripts/check-mutation.sh":    {Path: "services/worker/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
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
		"services/scripts/check-dry.sh":          {Path: "services/scripts/check-dry.sh", Content: cleanDryScript()},
		"services/scripts/check-crap.sh":         {Path: "services/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"services/scripts/check-mutation.sh":     {Path: "services/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
		"services/api/go.mod":                    {Path: "services/api/go.mod"},
		"services/api/main.go":                   {Path: "services/api/main.go"},
		"services/api/.golangci.yml":             {Path: "services/api/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/api/scripts/check-coverage.sh": {Path: "services/api/scripts/check-coverage.sh", Content: cleanCoverageScript},
		"services/api/scripts/check-dry.sh":      {Path: "services/api/scripts/check-dry.sh", Content: cleanDryScript()},
		"services/api/scripts/check-crap.sh":     {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"services/api/scripts/check-mutation.sh": {Path: "services/api/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
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
		"services/api/scripts/check-dry.sh":            {Path: "services/api/scripts/check-dry.sh", Content: cleanDryScript()},
		"services/api/scripts/check-crap.sh":           {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"services/api/scripts/check-mutation.sh":       {Path: "services/api/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
		"services/worker/go.mod":                       {Path: "services/worker/go.mod"},
		"services/worker/main.go":                      {Path: "services/worker/main.go"},
		"services/worker/.golangci.yml":                {Path: "services/worker/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/worker/scripts/check-go-coverage.sh": {Path: "services/worker/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/worker/scripts/check-dry.sh":         {Path: "services/worker/scripts/check-dry.sh", Content: cleanDryScript()},
		"services/worker/scripts/check-crap.sh":        {Path: "services/worker/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"services/worker/scripts/check-mutation.sh":    {Path: "services/worker/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
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
		"go/scripts/check-dry.sh":         {Path: "go/scripts/check-dry.sh", Content: cleanDryScript()},
		"go/scripts/check-crap.sh":        {Path: "go/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"go/scripts/check-mutation.sh":    {Path: "go/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
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

func TestGoRulesDoNotApplyRunDefaultsToActionSteps(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    defaults:
      run:
        working-directory: go
    steps:
      - run: go test ./...
      - run: go vet ./...
      - uses: golangci/golangci-lint-action@v8
`,
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	assertRuleIDs(t, report.Findings, []string{GoLintRequiredRuleID})
}

func TestGoRulesIgnoreWorkflowListsBeforeJobs(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                       {Path: "README.md"},
		"AGENTS.md":                       {Path: "AGENTS.md"},
		"go/go.mod":                       {Path: "go/go.mod"},
		"go/main.go":                      {Path: "go/main.go"},
		"go/.golangci.yml":                {Path: "go/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"go/scripts/check-go-coverage.sh": {Path: "go/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"go/scripts/check-dry.sh":         {Path: "go/scripts/check-dry.sh", Content: cleanDryScript()},
		"go/scripts/check-crap.sh":        {Path: "go/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"go/scripts/check-mutation.sh":    {Path: "go/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
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

func TestGoRulesKeepNamedWorkflowStepsForModule(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"README.md":                                    {Path: "README.md"},
		"AGENTS.md":                                    {Path: "AGENTS.md"},
		"services/api/go.mod":                          {Path: "services/api/go.mod"},
		"services/api/main.go":                         {Path: "services/api/main.go"},
		"services/api/.golangci.yml":                   {Path: "services/api/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/api/scripts/check-go-coverage.sh":    {Path: "services/api/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/api/scripts/check-dry.sh":            {Path: "services/api/scripts/check-dry.sh", Content: cleanDryScript()},
		"services/api/scripts/check-crap.sh":           {Path: "services/api/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"services/api/scripts/check-mutation.sh":       {Path: "services/api/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
		"services/worker/go.mod":                       {Path: "services/worker/go.mod"},
		"services/worker/main.go":                      {Path: "services/worker/main.go"},
		"services/worker/.golangci.yml":                {Path: "services/worker/.golangci.yml", Content: "linters:\n  enable:\n    - cyclop\n"},
		"services/worker/scripts/check-go-coverage.sh": {Path: "services/worker/scripts/check-go-coverage.sh", Content: cleanCoverageScript},
		"services/worker/scripts/check-dry.sh":         {Path: "services/worker/scripts/check-dry.sh", Content: cleanDryScript()},
		"services/worker/scripts/check-crap.sh":        {Path: "services/worker/scripts/check-crap.sh", Content: cleanCRAPScript()},
		"services/worker/scripts/check-mutation.sh":    {Path: "services/worker/scripts/check-mutation.sh", Content: cleanMutationScript("main.go")},
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

func TestWorkflowCommandSectionsAcceptChompedRunBlocks(t *testing.T) {
	sections := commandSections(repo.File{
		Path: ".github/workflows/ci.yml",
		Content: `name: CI
jobs:
  test:
    steps:
      - run: |-
          go test ./...
      - run: >-
          go vet
          ./...
`,
	})

	want := []string{"go test ./...", "go vet ./..."}
	if !reflect.DeepEqual(sections, want) {
		t.Fatalf("workflow chomped run sections = %#v, want %#v", sections, want)
	}
}

func TestWorkflowCommandSectionsIgnorePostRunYamlBlocks(t *testing.T) {
	sections := commandSections(repo.File{
		Path: ".github/workflows/ci.yml",
		Content: `name: CI
jobs:
  test:
    steps:
      - run: |
          echo noop
        env:
          SCRIPT: |
            go test ./...
            go vet ./...
`,
	})

	want := []string{"echo noop"}
	if !reflect.DeepEqual(sections, want) {
		t.Fatalf("workflow run sections = %#v, want %#v", sections, want)
	}
}
