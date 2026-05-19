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
  crap_max_score: 8
  targets:
    - .
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
		"main.go": {Path: "main.go"},
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  targets:
    - .
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
		"go/main.go": {Path: "go/main.go"},
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  crap_max_score: 8
  targets:
    - .
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

func TestGoMutationRuleResolvesRepoRootTargetsForNestedModule(t *testing.T) {
	files := cleanGoGuardrailFiles(map[string]repo.File{
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  targets:
    - go
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
      - run: ./scripts/check-dry.sh
      - run: ./scripts/check-crap.sh
      - run: go run ./cmd/slophammer go mutate .. --scan
`,
		},
	})

	report := Run(context.Background(), repo.NewSnapshot("/repo", files), DefaultRules())

	if !report.OK {
		t.Fatalf("report.OK = false, findings = %#v", report.Findings)
	}
}

func TestGoMutationRuleAcceptsRootConfigCommandForMultipleModules(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"a/go.mod":  {Path: "a/go.mod"},
		"a/main.go": {Path: "a/main.go"},
		"b/go.mod":  {Path: "b/go.mod"},
		"b/main.go": {Path: "b/main.go"},
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  targets:
    - .
`,
		},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: slophammer-go mutate . --scan
`,
		},
	})
	roots := goProjectRoots(snapshot)

	for _, root := range roots {
		scoped := goProjectSnapshot(snapshot, root, roots)
		if !hasMutate4GoCommandForRoot(snapshot, scoped, root, roots) {
			t.Fatalf("hasMutate4GoCommandForRoot(%q) = false, want true", root)
		}
	}
}

func TestGoMutationRuleSkipsModulesOutsideConfiguredTargets(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"a/go.mod":  {Path: "a/go.mod"},
		"a/main.go": {Path: "a/main.go"},
		"b/go.mod":  {Path: "b/go.mod"},
		"b/main.go": {Path: "b/main.go"},
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  targets:
    - a
`,
		},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: slophammer-go mutate . --scan
`,
		},
	})
	roots := goProjectRoots(snapshot)

	for _, root := range roots {
		scoped := goProjectSnapshot(snapshot, root, roots)
		if !hasMutate4GoCommandForRoot(snapshot, scoped, root, roots) {
			t.Fatalf("hasMutate4GoCommandForRoot(%q) = false, want true", root)
		}
	}
}

func TestGoMutationRuleDoesNotAcceptUnrelatedModuleLocalConfig(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"a/go.mod":  {Path: "a/go.mod"},
		"a/main.go": {Path: "a/main.go"},
		"a/slophammer.yml": {
			Path: "a/slophammer.yml",
			Content: `go:
  targets:
    - .
`,
		},
		"b/go.mod":  {Path: "b/go.mod"},
		"b/main.go": {Path: "b/main.go"},
		"b/slophammer.yml": {
			Path: "b/slophammer.yml",
			Content: `go:
  targets:
    - .
`,
		},
		"scripts/check-mutation.sh": {
			Path:    "scripts/check-mutation.sh",
			Content: "cd a && slophammer-go mutate --scan\n",
		},
	})
	roots := goProjectRoots(snapshot)

	aScoped := goProjectSnapshot(snapshot, "a", roots)
	if !hasMutate4GoCommandForRoot(snapshot, aScoped, "a", roots) {
		t.Fatal("hasMutate4GoCommandForRoot(a) = false, want true")
	}
	bScoped := goProjectSnapshot(snapshot, "b", roots)
	if hasMutate4GoCommandForRoot(snapshot, bScoped, "b", roots) {
		t.Fatal("hasMutate4GoCommandForRoot(b) = true, want false")
	}
}

func TestGoMutationRuleDoesNotUseRootScopeToBypassLocalConfig(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  targets:
    - b
`,
		},
		"a/go.mod":  {Path: "a/go.mod"},
		"a/main.go": {Path: "a/main.go"},
		"a/slophammer.yml": {
			Path: "a/slophammer.yml",
			Content: `go:
  targets:
    - .
`,
		},
		"b/go.mod":  {Path: "b/go.mod"},
		"b/main.go": {Path: "b/main.go"},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: slophammer-go mutate . --scan
`,
		},
	})
	roots := goProjectRoots(snapshot)
	scoped := goProjectSnapshot(snapshot, "a", roots)

	if hasMutate4GoCommandForRoot(snapshot, scoped, "a", roots) {
		t.Fatal("hasMutate4GoCommandForRoot(a) = true, want false")
	}
}

func TestGoMutationRuleRejectsRepoRootCommandForLocalConfigWithoutModuleRoot(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"go/go.mod":  {Path: "go/go.mod"},
		"go/main.go": {Path: "go/main.go"},
		"go/slophammer.yml": {
			Path: "go/slophammer.yml",
			Content: `go:
  targets:
    - .
`,
		},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: go test ./go/... && slophammer-go mutate --scan
`,
		},
	})
	roots := goProjectRoots(snapshot)
	scoped := goProjectSnapshot(snapshot, "go", roots)

	if hasMutate4GoCommandForRoot(snapshot, scoped, "go", roots) {
		t.Fatal("hasMutate4GoCommandForRoot(go) = true, want false")
	}
}

func TestGoMutationRuleUsesSingleModuleFallbackForRootConfig(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"go/go.mod":              {Path: "go/go.mod"},
		"go/internal/example.go": {Path: "go/internal/example.go"},
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  targets:
    - internal
`,
		},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: slophammer-go mutate . --scan
`,
		},
	})
	roots := goProjectRoots(snapshot)
	scoped := goProjectSnapshot(snapshot, "go", roots)

	if !hasMutate4GoCommandForRoot(snapshot, scoped, "go", roots) {
		t.Fatal("hasMutate4GoCommandForRoot = false, want true")
	}
}

func TestGoMutationRuleAllowsRootCoverageWithUnrelatedLocalConfig(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"go/go.mod":  {Path: "go/go.mod"},
		"go/main.go": {Path: "go/main.go"},
		"go/slophammer.yml": {
			Path: "go/slophammer.yml",
			Content: `rules:
  repo.readme-required:
    severity: warn
`,
		},
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  targets:
    - .
`,
		},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: slophammer-go mutate . --scan
`,
		},
	})
	roots := goProjectRoots(snapshot)
	scoped := goProjectSnapshot(snapshot, "go", roots)

	if !hasMutate4GoCommandForRoot(snapshot, scoped, "go", roots) {
		t.Fatal("hasMutate4GoCommandForRoot = false, want true")
	}
}

func TestGoMutationRuleResolvesModuleLocalConfigTargets(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  crap_max_score: 8
`,
		},
		"go/go.mod":              {Path: "go/go.mod"},
		"go/internal/example.go": {Path: "go/internal/example.go"},
		"go/slophammer.yml": {
			Path: "go/slophammer.yml",
			Content: `go:
  targets:
    - internal
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
      - run: slophammer-go mutate --scan
`,
		},
	})
	roots := goProjectRoots(snapshot)
	scoped := goProjectSnapshot(snapshot, "go", roots)

	if !hasMutate4GoCommandForRoot(snapshot, scoped, "go", roots) {
		t.Fatal("hasMutate4GoCommandForRoot = false, want true")
	}
}

func TestGoMutationRuleAcceptsRepoRootCommandForModuleLocalConfig(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"go/go.mod":  {Path: "go/go.mod"},
		"go/main.go": {Path: "go/main.go"},
		"go/slophammer.yml": {
			Path: "go/slophammer.yml",
			Content: `go:
  targets:
    - .
`,
		},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: slophammer-go mutate go --scan
`,
		},
	})
	roots := goProjectRoots(snapshot)
	scoped := goProjectSnapshot(snapshot, "go", roots)

	if !hasMutate4GoCommandForRoot(snapshot, scoped, "go", roots) {
		t.Fatal("hasMutate4GoCommandForRoot = false, want true")
	}
}

func TestGoMutationRuleAcceptsCdCommandForModuleLocalConfig(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"go/go.mod":  {Path: "go/go.mod"},
		"go/main.go": {Path: "go/main.go"},
		"go/slophammer.yml": {
			Path: "go/slophammer.yml",
			Content: `go:
  targets:
    - .
`,
		},
		"scripts/check-mutation.sh": {
			Path:    "scripts/check-mutation.sh",
			Content: "cd go && slophammer-go mutate --scan\n",
		},
	})
	roots := goProjectRoots(snapshot)
	scoped := goProjectSnapshot(snapshot, "go", roots)

	if !hasMutate4GoCommandForRoot(snapshot, scoped, "go", roots) {
		t.Fatal("hasMutate4GoCommandForRoot = false, want true")
	}
}

func TestGoToolRulesRejectConfigBackedNonRootParentPath(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"go/main.go": {Path: "go/main.go"},
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  crap_max_score: 8
  targets:
    - .
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
      - run: go run ./cmd/slophammer go crap ../tmp
      - run: go run ./cmd/slophammer go mutate ../tmp --scan
`,
		},
	})

	if hasCRAP4GoGate(snapshot) {
		t.Fatal("hasCRAP4GoGate = true, want false for non-root parent path")
	}
	if hasMutate4GoCommand(snapshot) {
		t.Fatal("hasMutate4GoCommand = true, want false for non-root parent path")
	}
}

func TestGoToolRulesRequireConfigRootForDeepWorkingDirectory(t *testing.T) {
	files := map[string]repo.File{
		"services/api/main.go": {Path: "services/api/main.go"},
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  crap_max_score: 8
  targets:
    - .
`,
		},
	}
	for _, tt := range []struct {
		name string
		path string
		want bool
	}{
		{name: "one parent", path: ".."},
		{name: "repo root", path: "../..", want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			snapshotFiles := map[string]repo.File{}
			for path, file := range files {
				snapshotFiles[path] = file
			}
			snapshotFiles[".github/workflows/ci.yml"] = repo.File{
				Path: ".github/workflows/ci.yml",
				Content: `name: CI
defaults:
  run:
    working-directory: services/api
jobs:
  test:
    steps:
      - run: go run ./cmd/slophammer go crap ` + tt.path + `
      - run: go run ./cmd/slophammer go mutate ` + tt.path + ` --scan
`,
			}
			snapshot := repo.NewSnapshot("/repo", snapshotFiles)

			if got := hasCRAP4GoGate(snapshot); got != tt.want {
				t.Fatalf("hasCRAP4GoGate = %t, want %t", got, tt.want)
			}
			if got := hasMutate4GoCommand(snapshot); got != tt.want {
				t.Fatalf("hasMutate4GoCommand = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestGoToolRulesAcceptConfigBackedRootSlophammerCommands(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"main.go": {Path: "main.go"},
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  crap_max_score: 8
  targets:
    - .
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

func TestGoToolRulesAcceptConfigBackedRootSlophammerCommandsWithDefaultPath(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"main.go": {Path: "main.go"},
		"slophammer.yml": {
			Path: "slophammer.yml",
			Content: `go:
  crap_max_score: 8
  targets:
    - .
`,
		},
		".github/workflows/ci.yml": {
			Path: ".github/workflows/ci.yml",
			Content: `name: CI
jobs:
  test:
    steps:
      - run: go run ./cmd/slophammer go crap
      - run: go run ./cmd/slophammer go mutate --scan
`,
		},
	})

	if !hasCRAP4GoGate(snapshot) {
		t.Fatal("hasCRAP4GoGate = false, want true with default root path")
	}
	if !hasMutate4GoCommand(snapshot) {
		t.Fatal("hasMutate4GoCommand = false, want true with default root path")
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
