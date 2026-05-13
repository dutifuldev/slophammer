package rules

import (
	"context"
	"testing"

	"github.com/dutifuldev/slophammer/go/internal/repo"
)

func TestGoCRAPRuleRequiresMetricThreshold(t *testing.T) {
	tests := []struct {
		name      string
		workflow  string
		crapCheck string
	}{
		{
			name:      "missing threshold",
			crapCheck: "go run github.com/unclebob/crap4go/cmd/crap4go@latest\n",
		},
		{
			name:      "report redirection",
			crapCheck: "go run github.com/unclebob/crap4go/cmd/crap4go@latest > crap-report.txt\n",
		},
		{
			name: "threshold in different workflow step",
			workflow: `name: CI
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
			crapCheck: "go run github.com/unclebob/crap4go/cmd/crap4go@latest\n",
		},
		{
			name: "unrelated coverage threshold",
			crapCheck: `go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
minimum_coverage="80"
awk -v total="90" -v minimum="$minimum_coverage" 'BEGIN { exit !(total + 0 >= minimum + 0) }'
go run github.com/unclebob/crap4go/cmd/crap4go@latest
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrides := map[string]repo.File{
				"go/scripts/check-crap.sh": {Path: "go/scripts/check-crap.sh", Content: tt.crapCheck},
			}
			if tt.workflow != "" {
				overrides[".github/workflows/ci.yml"] = repo.File{Path: ".github/workflows/ci.yml", Content: tt.workflow}
			}
			files := cleanGoGuardrailFiles(overrides)
			report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

			assertRuleIDs(t, report.Findings, []string{GoCRAPRequiredRuleID})
		})
	}
}

func TestGoToolRulesAcceptSlophammerGoCommands(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  crap_max_score: 30
  mutation_targets:
    - go/internal/rules/rules.go
`,
		},
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
      - run: go run ./cmd/slophammer go dry ..
      - run: go run ./cmd/slophammer go crap ..
      - run: go run ./cmd/slophammer go mutate .. --scan
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
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  mutation_targets:
    - go/internal/rules/rules.go
`,
		},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
defaults:
  run:
    working-directory: go
jobs:
  test:
    steps:
      - run: go run ./cmd/slophammer go mutate . --scan
`,
		},
	})

	if hasMutate4GoCommand(snapshot) {
		t.Fatal("hasMutate4GoCommand = true, want false without --target or config-root path")
	}
}

func TestGoToolRulesAcceptConfigBackedSlophammerCommands(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  crap_max_score: 30
  mutation_targets:
    - go/internal/rules/rules.go
`,
		},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: go run ./cmd/slophammer go crap ..
      - run: go run ./cmd/slophammer go mutate .. --scan
`,
		},
	})

	if !hasCRAP4GoGate(snapshot) {
		t.Fatal("hasCRAP4GoGate = false, want true with configured threshold")
	}
	if !hasMutate4GoCommand(snapshot) {
		t.Fatal("hasMutate4GoCommand = false, want true with configured target")
	}
}

func TestGoToolRulesAcceptConfigBackedRootSlophammerCommands(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  crap_max_score: 30
  mutation_targets:
    - internal/rules/rules.go
`,
		},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: go run ./cmd/slophammer go crap .
      - run: go run ./cmd/slophammer go mutate . --scan
`,
		},
	})

	if !hasCRAP4GoGate(snapshot) {
		t.Fatal("hasCRAP4GoGate = false, want true with root config path")
	}
	if !hasMutate4GoCommand(snapshot) {
		t.Fatal("hasMutate4GoCommand = false, want true with root config path")
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
		{name: "go global flag before run", command: "go -C tools run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan", want: true},
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
