package rules

import (
	"strings"
	"testing"
)

func TestBindingFilteredWorkflowRemovesNeutralizedJobs(t *testing.T) {
	content := `name: CI
on: [push]
jobs:
  skipped:
    if: false
    steps:
      - run: go test ./...
  live:
    steps:
      - run: go vet ./...
`
	filtered, ok := bindingFilteredWorkflow(content)

	if !ok {
		t.Fatal("ok = false, want structural filtering")
	}
	if strings.Contains(filtered, "go test ./...") {
		t.Fatalf("neutralized job survived: %q", filtered)
	}
	if !strings.Contains(filtered, "go vet ./...") {
		t.Fatalf("live job lost: %q", filtered)
	}
}

func TestBindingFilteredWorkflowRemovesNeutralizedBlockScalarSteps(t *testing.T) {
	content := `name: CI
on: [push]
jobs:
  test:
    steps:
      - run: |
          go test ./...
          go vet ./...
        continue-on-error: true
      - run: >
          golangci-lint
          run
`
	filtered, ok := bindingFilteredWorkflow(content)

	if !ok {
		t.Fatal("ok = false, want structural filtering")
	}
	if strings.Contains(filtered, "go test ./...") || strings.Contains(filtered, "go vet ./...") {
		t.Fatalf("neutralized block scalar content survived: %q", filtered)
	}
	if !strings.Contains(filtered, "golangci-lint") {
		t.Fatalf("folded live step lost: %q", filtered)
	}
}

func TestBindingFilteredWorkflowRemovesExpressionLiteralNeutralization(t *testing.T) {
	content := `name: CI
on: [push]
jobs:
  test:
    steps:
      - run: go test ./...
        continue-on-error: ${{ true }}
      - run: go vet ./...
        if: ${{ false }}
      - run: golangci-lint run
`
	filtered, ok := bindingFilteredWorkflow(content)

	if !ok {
		t.Fatal("ok = false, want structural filtering")
	}
	if strings.Contains(filtered, "go test ./...") || strings.Contains(filtered, "go vet ./...") {
		t.Fatalf("expression-literal neutralized steps survived: %q", filtered)
	}
	if !strings.Contains(filtered, "golangci-lint run") {
		t.Fatalf("live step lost: %q", filtered)
	}
}

func TestBindingFilteredWorkflowDropsNonIntegrationTriggers(t *testing.T) {
	content := `name: CI
on:
  push:
    branches: [branch-that-never-existed]
jobs:
  test:
    steps:
      - run: go test ./...
`
	filtered, ok := bindingFilteredWorkflow(content)

	if !ok || strings.TrimSpace(filtered) != "" {
		t.Fatalf("filtered = %q ok = %v, want empty binding content", filtered, ok)
	}
}

func TestBindingFilteredWorkflowKeepsUntouchedWorkflows(t *testing.T) {
	content := `name: CI
on: [push]
jobs:
  test:
    steps:
      - run: go test ./...
`
	filtered, ok := bindingFilteredWorkflow(content)

	if !ok || filtered != content {
		t.Fatalf("filtered = %q ok = %v, want original content", filtered, ok)
	}
}

func TestBindingFilteredWorkflowFallsBackWithoutStructure(t *testing.T) {
	for _, content := range []string{"go test ./...", ": [", "name: CI\non: [push]\n"} {
		if _, ok := bindingFilteredWorkflow(content); ok {
			t.Fatalf("ok = true for unfilterable content %q", content)
		}
	}
}

func TestWorkflowTriggerNodeResolvesYAMLBooleanOnKey(t *testing.T) {
	content := `"true": [push]
jobs:
  test:
    steps:
      - run: go test ./...
`
	filtered, ok := bindingFilteredWorkflow(content)

	if !ok || !strings.Contains(filtered, "go test ./...") {
		t.Fatalf("filtered = %q ok = %v, want boolean on key honored", filtered, ok)
	}
}
