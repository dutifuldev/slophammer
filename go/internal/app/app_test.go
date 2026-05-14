package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/dutifuldev/slophammer/go/internal/config"
	"github.com/dutifuldev/slophammer/go/internal/repo"
	"github.com/dutifuldev/slophammer/go/internal/rules"
	"github.com/dutifuldev/slophammer/go/internal/toolchecks"
)

func TestCheckReturnsOKForCleanRepo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", "# Test\n")
	writeFile(t, root, "AGENTS.md", "# Agents\n")
	writeFile(t, root, ".github/workflows/ci.yml", "name: CI\n")

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Check(context.Background(), CheckOptions{Root: root, Format: "json"}, &out, &errOut)

	if code != ExitOK {
		t.Fatalf("code = %d, want %d; stderr=%q", code, ExitOK, errOut.String())
	}
	if !strings.Contains(out.String(), `"ok": true`) {
		t.Fatalf("json output = %q", out.String())
	}
}

func TestCheckMatchesSharedFixtures(t *testing.T) {
	tests := []struct {
		name string
		code int
	}{
		{name: "clean", code: ExitOK},
		{name: "missing-readme", code: ExitFindings},
		{name: "missing-agents", code: ExitFindings},
		{name: "missing-ci", code: ExitFindings},
		{name: "go-clean", code: ExitOK},
		{name: "go-missing-module", code: ExitFindings},
		{name: "go-missing-tests", code: ExitFindings},
		{name: "go-missing-vet", code: ExitFindings},
		{name: "go-missing-lint", code: ExitFindings},
		{name: "go-missing-coverage", code: ExitFindings},
		{name: "go-missing-complexity", code: ExitFindings},
		{name: "go-missing-dry", code: ExitFindings},
		{name: "go-missing-crap", code: ExitFindings},
		{name: "go-missing-mutation", code: ExitFindings},
		{name: "go-bad-dependency", code: ExitFindings},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkSharedFixture(t, tt.name, tt.code)
		})
	}
}

func TestCheckReturnsFindingsForMissingFiles(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Check(context.Background(), CheckOptions{Root: t.TempDir(), Format: "text"}, &out, &errOut)

	if code != ExitFindings {
		t.Fatalf("code = %d, want %d; stderr=%q", code, ExitFindings, errOut.String())
	}
	if !strings.Contains(out.String(), "repo.agents-required") {
		t.Fatalf("text output = %q", out.String())
	}
}

func TestCheckRejectsUnknownFormat(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Check(context.Background(), CheckOptions{Root: t.TempDir(), Format: "xml"}, &out, &errOut)

	if code != ExitError {
		t.Fatalf("code = %d, want %d", code, ExitError)
	}
	if !strings.Contains(errOut.String(), "unsupported format") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestCheckWritesSARIF(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Check(context.Background(), CheckOptions{Root: t.TempDir(), Format: "sarif"}, &out, &errOut)

	if code != ExitFindings {
		t.Fatalf("code = %d, want %d; stderr=%q", code, ExitFindings, errOut.String())
	}
	if !strings.Contains(out.String(), `"version": "2.1.0"`) ||
		!strings.Contains(out.String(), `"ruleId": "repo.agents-required"`) {
		t.Fatalf("SARIF output = %q", out.String())
	}
}

func TestCheckRejectsInvalidConfig(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "slophammer.yml", "rules:\n  repo.readme-required:\n    severity: info\n")

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Check(context.Background(), CheckOptions{Root: root, Format: "text"}, &out, &errOut)

	if code != ExitError {
		t.Fatalf("code = %d, want %d", code, ExitError)
	}
	if !strings.Contains(errOut.String(), "config failed") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestCheckExecuteAddsToolFindings(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", "# Test\n")
	writeFile(t, root, "AGENTS.md", "# Agents\n")
	writeFile(t, root, ".github/workflows/ci.yml", "name: CI\n")
	writeFile(t, root, "slophammer.yml", strings.Join([]string{
		"go:",
		"  dry_max_candidates: 1",
		"  crap_max_score: 8",
		"  mutation_targets:",
		"    - internal/example.go",
		"",
	}, "\n"))

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := check(context.Background(), CheckOptions{Root: root, Format: "json", Execute: true}, &out, &errOut, executeFakeRunner{})

	if code != ExitFindings {
		t.Fatalf("code = %d, want %d; stderr=%q", code, ExitFindings, errOut.String())
	}
	report := unmarshalReport(t, out.Bytes(), "execute")
	assertFinding(t, report, rules.GoDryRequiredRuleID)
	assertFinding(t, report, rules.GoCRAPRequiredRuleID)
	assertFinding(t, report, rules.GoMutationRequiredRuleID)
}

func TestApplyCommandConfigUsesConfiguredDefaults(t *testing.T) {
	cfg := config.Config{Go: config.GoConfig{
		DRYMaxCandidates:    0,
		DRYMaxCandidatesSet: true,
		DRYPaths:            []string{"go/cmd", "go/internal"},
		DRYExclude:          []string{"**/*_test.go"},
		CRAPMaxScore:        8,
		MutationTargets:     []string{"a.go", "b.go"},
	}}

	dry := toolchecks.DryOptions{}
	applyDryConfig(&dry, cfg)
	if dry.MaximumCandidates != 0 || !dry.MaximumSet {
		t.Fatalf("dry = %#v", dry)
	}
	if !reflect.DeepEqual(dry.Paths, []string{"go/cmd", "go/internal"}) || !reflect.DeepEqual(dry.Exclude, []string{"**/*_test.go"}) {
		t.Fatalf("dry paths = %#v excludes = %#v", dry.Paths, dry.Exclude)
	}

	crap := toolchecks.CRAPOptions{}
	applyCRAPConfig(&crap, cfg)
	if crap.MaximumScore != 8 || !crap.MaximumSet {
		t.Fatalf("crap = %#v", crap)
	}

	mutation := toolchecks.MutationOptions{}
	applyMutationConfig(&mutation, cfg)
	if !reflect.DeepEqual(mutation.Targets, []string{"a.go", "b.go"}) {
		t.Fatalf("mutation = %#v", mutation)
	}
}

func TestApplyCommandConfigKeepsExplicitValues(t *testing.T) {
	cfg := config.Config{Go: config.GoConfig{
		DRYMaxCandidates:    7,
		DRYMaxCandidatesSet: true,
		DRYPaths:            []string{"go/internal"},
		CRAPMaxScore:        8,
		MutationTargets:     []string{"configured.go"},
	}}

	dry := toolchecks.DryOptions{MaximumCandidates: 3, MaximumSet: true}
	applyDryConfig(&dry, cfg)
	if dry.MaximumCandidates != 3 {
		t.Fatalf("dry = %#v", dry)
	}

	crap := toolchecks.CRAPOptions{MaximumScore: 4, MaximumSet: true}
	applyCRAPConfig(&crap, cfg)
	if crap.MaximumScore != 4 {
		t.Fatalf("crap = %#v", crap)
	}

	mutation := toolchecks.MutationOptions{Target: "explicit.go"}
	applyMutationConfig(&mutation, cfg)
	if mutation.Target != "explicit.go" || len(mutation.Targets) != 0 {
		t.Fatalf("mutation = %#v", mutation)
	}
}

func TestRunWithCommandConfigLoadsRepoConfig(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "slophammer.yml", "go:\n  crap_max_score: 8\n")

	var errOut bytes.Buffer
	code := runWithCommandConfig(root, &errOut, func(snapshot repo.Snapshot, cfg config.Config) int {
		if snapshot.Root != root {
			t.Fatalf("snapshot.Root = %q, want %q", snapshot.Root, root)
		}
		if cfg.Go.CRAPMaxScore != 8 {
			t.Fatalf("CRAPMaxScore = %v, want 8", cfg.Go.CRAPMaxScore)
		}
		return ExitOK
	})

	if code != ExitOK {
		t.Fatalf("code = %d, want %d; stderr=%q", code, ExitOK, errOut.String())
	}
}

func TestRunWithCommandConfigRejectsInvalidRoot(t *testing.T) {
	var errOut bytes.Buffer
	code := runWithCommandConfig(filepath.Join(t.TempDir(), "missing"), &errOut, func(repo.Snapshot, config.Config) int {
		t.Fatal("run should not be called")
		return ExitOK
	})

	if code != ExitError {
		t.Fatalf("code = %d, want %d", code, ExitError)
	}
	if !strings.Contains(errOut.String(), "config failed") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestExplain(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Explain("repo.readme-required", &out, &errOut)

	if code != ExitOK {
		t.Fatalf("code = %d, want %d; stderr=%q", code, ExitOK, errOut.String())
	}
	if !strings.Contains(out.String(), "repo.readme-required") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestExplainRejectsUnknownRule(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Explain("missing", &out, &errOut)

	if code != ExitError {
		t.Fatalf("code = %d, want %d", code, ExitError)
	}
	if !strings.Contains(errOut.String(), "unknown rule") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
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

func checkSharedFixture(t *testing.T, name string, wantCode int) {
	t.Helper()
	root := repoRoot(t)
	fixtureRoot := filepath.Join(root, "fixtures", "repos", name)
	expectedPath := filepath.Join(root, "fixtures", "expected", name+".json")

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Check(context.Background(), CheckOptions{Root: fixtureRoot, Format: "json"}, &out, &errOut)

	if code != wantCode {
		t.Fatalf("code = %d, want %d; stderr=%q", code, wantCode, errOut.String())
	}

	got := unmarshalReport(t, out.Bytes(), "actual")
	// #nosec G304 -- test fixtures are read from a path derived from the test name table.
	expectedContent, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read expected report: %v", err)
	}
	want := unmarshalReport(t, expectedContent, "expected")

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("report mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}

func unmarshalReport(t *testing.T, content []byte, label string) rules.Report {
	t.Helper()
	var report rules.Report
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatalf("unmarshal %s report: %v\n%s", label, err, string(content))
	}
	return report
}

func assertFinding(t *testing.T, report rules.Report, ruleID string) {
	t.Helper()
	for _, finding := range report.Findings {
		if finding.RuleID == ruleID {
			return
		}
	}
	t.Fatalf("missing finding %s in %#v", ruleID, report.Findings)
}

type executeFakeRunner struct{}

func (executeFakeRunner) Run(_ context.Context, _ string, _ string, args ...string) (toolchecks.CommandResult, error) {
	command := strings.Join(args, " ")
	switch {
	case strings.Contains(command, "dry4go"):
		return toolchecks.CommandResult{Stdout: []byte(`{"candidates":[{},{}]}`)}, nil
	case strings.Contains(command, "crap4go"):
		return toolchecks.CommandResult{Stdout: []byte("pkg.Func 1 2 3 10.1\n")}, nil
	case strings.Contains(command, "mutate4go"):
		return toolchecks.CommandResult{}, errors.New("boom")
	default:
		return toolchecks.CommandResult{}, errors.New("unexpected command")
	}
}
