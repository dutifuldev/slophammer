package app

import (
	"reflect"
	"testing"

	"github.com/dutifuldev/slophammer/go/internal/repo"
	"github.com/dutifuldev/slophammer/go/internal/toolchecks"
)

func TestGoToolRootsUseNestedModule(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"fixtures/repos/go-clean/go.mod": {Path: "fixtures/repos/go-clean/go.mod"},
		"go/go.mod":                      {Path: "go/go.mod"},
		"templates/go/go.mod":            {Path: "templates/go/go.mod"},
	})

	got := goToolRoots("..", snapshot)
	want := []string{"../go"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("goToolRoots = %#v, want %#v", got, want)
	}
}

func TestMutationOptionsForModulesTrimsConfiguredTargets(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"go/go.mod": {Path: "go/go.mod"},
	})

	got := mutationOptionsForModules(toolchecks.MutationOptions{
		Root:    "..",
		Targets: []string{"go/internal/rules/rules.go"},
		Scan:    true,
	}, snapshot)
	want := []toolchecks.MutationOptions{
		{Root: "../go", Targets: []string{"internal/rules/rules.go"}, Scan: true},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mutationOptionsForModules = %#v, want %#v", got, want)
	}
}

func TestMutationOptionsForModulesKeepsRootModuleTargets(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo/go", map[string]repo.File{
		"go.mod": {Path: "go.mod"},
	})

	got := mutationOptionsForModules(toolchecks.MutationOptions{
		Root:   ".",
		Target: "internal/rules/rules.go",
		Scan:   true,
	}, snapshot)
	want := []toolchecks.MutationOptions{
		{Root: ".", Targets: []string{"internal/rules/rules.go"}, Scan: true},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mutationOptionsForModules = %#v, want %#v", got, want)
	}
}
