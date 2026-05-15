package toolchecks

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dutifuldev/slophammer/go/internal/gotools"
)

func TestCheckDryRunsNativeEngineAndEnforcesCandidateBudget(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "left.go", duplicateSource("Left"))
	writeFile(t, root, "right.go", duplicateSource("Right"))
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckDry(context.Background(), DryOptions{Root: root, MaximumCandidates: 0, MaximumSet: true}, &out, &errOut, &fakeRunner{})

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "DRY candidates:") || !strings.Contains(out.String(), "maximum: 0") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestCheckDryHonorsExplicitZeroBudget(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "left.go", duplicateSource("Left"))
	writeFile(t, root, "right.go", duplicateSource("Right"))
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckDry(context.Background(), DryOptions{Root: root, MaximumCandidates: 0, MaximumSet: true}, &out, &errOut, &fakeRunner{})

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "maximum: 0") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestCheckDryPassesConfiguredPaths(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "cmd/main.go", duplicateSource("Left"))
	writeFile(t, root, "internal/app/app.go", duplicateSource("Right"))
	writeFile(t, root, "ignored/ignored.go", duplicateSource("Ignored"))
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckDry(context.Background(), DryOptions{
		Root:              root,
		MaximumCandidates: 0,
		MaximumSet:        true,
		Paths:             []string{"cmd/main.go", "internal/app/app.go"},
	}, &out, &errOut, &fakeRunner{})

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "maximum: 0") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestCheckDryCanRenderJSONReport(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "left.go", duplicateSource("Left"))
	writeFile(t, root, "right.go", duplicateSource("Right"))
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckDry(context.Background(), DryOptions{Root: root, MaximumCandidates: 999, MaximumSet: true, Format: "json"}, &out, &errOut, &fakeRunner{})

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), `"findings"`) || !strings.Contains(out.String(), `"structural-function"`) {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestCheckDryCanRenderTextReport(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "left.go", duplicateSource("Left"))
	writeFile(t, root, "right.go", duplicateSource("Right"))
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckDry(context.Background(), DryOptions{Root: root, MaximumCandidates: 999, MaximumSet: true, Format: "text"}, &out, &errOut, &fakeRunner{})

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "Structural function findings:") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestCheckDryReportsScanErrors(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckDry(context.Background(), DryOptions{Root: filepath.Join(t.TempDir(), "missing")}, &out, &errOut, &fakeRunner{})

	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "dry check failed") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestCountDRYCandidatesAcceptsNativeAndLegacyReports(t *testing.T) {
	for _, tc := range []struct {
		name   string
		report string
		want   int
	}{
		{name: "native", report: `{"findings":[{},{}]}`, want: 2},
		{name: "legacy", report: `{"candidates":[{}]}`, want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CountDRYCandidates([]byte(tc.report))
			if err != nil {
				t.Fatalf("CountDRYCandidates returned error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("CountDRYCandidates = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestCountDRYCandidatesRejectsUnknownShape(t *testing.T) {
	if _, err := CountDRYCandidates([]byte(`{"groups":[]}`)); err == nil {
		t.Fatal("CountDRYCandidates returned nil error")
	}
}

func TestCheckCRAPRunsCRAP4GoAndReportsViolations(t *testing.T) {
	runner := &fakeRunner{output: []byte("pkg.Func 1 2 3 30.1\npkg.OK 1 2 3 30.0\n")}
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckCRAP(context.Background(), CRAPOptions{Root: "/repo", MaximumScore: 30, MaximumSet: true}, &out, &errOut, runner)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	wantArgs := gotools.CRAP4Go.GoRunArgs(gotools.Latest)
	if runnerCall := runner.call; runnerCall.dir != "/repo" || runnerCall.name != "go" || !reflect.DeepEqual(runnerCall.args, wantArgs) {
		t.Fatalf("call = %#v, want dir=/repo name=go args=%#v", runnerCall, wantArgs)
	}
	if !strings.Contains(errOut.String(), "CRAP score 30.1 exceeds maximum 30.0 for pkg.Func") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestCheckCRAPHonorsExplicitZeroLimit(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckCRAP(context.Background(), CRAPOptions{MaximumScore: 0, MaximumSet: true}, &out, &errOut, &fakeRunner{
		output: []byte("pkg.Func 1 2 3 0.1\n"),
	})

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "CRAP score 0.1 exceeds maximum 0.0 for pkg.Func") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestCheckMutationRunsMutate4GoScan(t *testing.T) {
	runner := &fakeRunner{}
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckMutation(context.Background(), MutationOptions{Root: "/repo", Target: "main.go", Scan: true}, &out, &errOut, runner)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	wantArgs := gotools.Mutate4Go.GoRunArgs(gotools.Latest, "main.go", "--scan")
	if runnerCall := runner.call; runnerCall.dir != "/repo" || runnerCall.name != "go" || !reflect.DeepEqual(runnerCall.args, wantArgs) {
		t.Fatalf("call = %#v, want dir=/repo name=go args=%#v", runnerCall, wantArgs)
	}
}

func TestCheckMutationRunsConfiguredTargets(t *testing.T) {
	runner := &fakeRunner{}
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckMutation(context.Background(), MutationOptions{Root: "/repo", Targets: []string{"a.go", "b.go"}, Scan: true}, &out, &errOut, runner)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if len(runner.calls) != 2 {
		t.Fatalf("calls = %#v, want 2 calls", runner.calls)
	}
	wantFirst := gotools.Mutate4Go.GoRunArgs(gotools.Latest, "a.go", "--scan")
	wantSecond := gotools.Mutate4Go.GoRunArgs(gotools.Latest, "b.go", "--scan")
	if !reflect.DeepEqual(runner.calls[0].args, wantFirst) || !reflect.DeepEqual(runner.calls[1].args, wantSecond) {
		t.Fatalf("calls = %#v, want args %#v and %#v", runner.calls, wantFirst, wantSecond)
	}
}

func TestCheckMutationRequiresTarget(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckMutation(context.Background(), MutationOptions{}, &out, &errOut, &fakeRunner{})

	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "--target is required") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestToolFailureReturnsInfrastructureError(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckMutation(context.Background(), MutationOptions{Target: "main.go"}, &out, &errOut, &fakeRunner{err: errors.New("boom")})

	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "mutate4go failed") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

type fakeRunner struct {
	call   fakeCall
	calls  []fakeCall
	output []byte
	stderr []byte
	err    error
}

func (runner *fakeRunner) Run(_ context.Context, dir string, name string, args ...string) (CommandResult, error) {
	runner.call = fakeCall{dir: dir, name: name, args: args}
	runner.calls = append(runner.calls, runner.call)
	return CommandResult{Stdout: runner.output, Stderr: runner.stderr}, runner.err
}

type fakeCall struct {
	dir  string
	name string
	args []string
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

func duplicateSource(name string) string {
	return `package sample

func ` + name + `(items []int) []int {
	var kept []int
	for _, item := range items {
		if item%2 == 0 {
			kept = append(kept, item+1)
		}
	}
	return kept
}
`
}
