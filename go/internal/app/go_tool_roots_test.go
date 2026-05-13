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

func TestDryOptionsForModuleExpandsProductionFiles(t *testing.T) {
	snapshot := repo.NewSnapshot("/repo", map[string]repo.File{
		"go/go.mod":                          {Path: "go/go.mod"},
		"go/cmd/slophammer/main.go":          {Path: "go/cmd/slophammer/main.go"},
		"go/internal/app/app.go":             {Path: "go/internal/app/app.go"},
		"go/internal/app/app_test.go":        {Path: "go/internal/app/app_test.go"},
		"go/internal/app/generated/model.go": {Path: "go/internal/app/generated/model.go"},
		"go/internal/app/testdata/input.go":  {Path: "go/internal/app/testdata/input.go"},
		"fixtures/repos/go-clean/main.go":    {Path: "fixtures/repos/go-clean/main.go"},
		"templates/go/main.go":               {Path: "templates/go/main.go"},
	})

	got, ok := dryOptionsForModule(toolchecks.DryOptions{
		Root:    "..",
		Paths:   []string{"go/cmd", "go/internal"},
		Exclude: []string{"**/*_test.go", "**/generated/**", "**/testdata/**", "fixtures/**", "templates/**"},
	}, snapshot, "go")
	want := toolchecks.DryOptions{
		Root:    "../go",
		Paths:   []string{"cmd/slophammer/main.go", "internal/app/app.go"},
		Exclude: []string{"**/*_test.go", "**/generated/**", "**/testdata/**", "fixtures/**", "templates/**"},
	}

	if !ok {
		t.Fatal("ok = false, want true")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dryOptionsForModule = %#v, want %#v", got, want)
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
