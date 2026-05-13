package toolchecks

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestCheckDryRunsDry4GoAndEnforcesCandidateBudget(t *testing.T) {
	runner := &fakeRunner{output: []byte(`{"candidates":[{},{}]}`)}
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckDry(context.Background(), DryOptions{Root: "/repo", MaximumCandidates: 1, MaximumSet: true}, &out, &errOut, runner)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	wantArgs := []string{"run", "github.com/unclebob/dry4go/cmd/dry4go@latest", "--format", "json", "."}
	if runnerCall := runner.call; runnerCall.dir != "/repo" || runnerCall.name != "go" || !reflect.DeepEqual(runnerCall.args, wantArgs) {
		t.Fatalf("call = %#v, want dir=/repo name=go args=%#v", runnerCall, wantArgs)
	}
	if !strings.Contains(out.String(), "DRY candidates: 2; maximum: 1") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestCheckDryHonorsExplicitZeroBudget(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckDry(context.Background(), DryOptions{MaximumCandidates: 0, MaximumSet: true}, &out, &errOut, &fakeRunner{
		output: []byte(`{"candidates":[{}]}`),
	})

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "DRY candidates: 1; maximum: 0") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestCheckDryParsesStdoutWhenGoRunWritesToStderr(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckDry(context.Background(), DryOptions{MaximumCandidates: 1, MaximumSet: true}, &out, &errOut, &fakeRunner{
		output: []byte(`{"candidates":[]}`),
		stderr: []byte("go: downloading github.com/unclebob/dry4go v0.0.0\n"),
	})

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "DRY candidates: 0; maximum: 1") {
		t.Fatalf("stdout = %q", out.String())
	}
	if !strings.Contains(errOut.String(), "go: downloading") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestCheckDryAcceptsNullCandidates(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckDry(context.Background(), DryOptions{MaximumCandidates: 0, MaximumSet: true}, &out, &errOut, &fakeRunner{
		output: []byte(`{"candidates":null}`),
	})

	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "DRY candidates: 0; maximum: 0") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestCheckDryRejectsInvalidReport(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := CheckDry(context.Background(), DryOptions{}, &out, &errOut, &fakeRunner{output: []byte(`{}`)})

	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "missing candidates") {
		t.Fatalf("stderr = %q", errOut.String())
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
	wantArgs := []string{"run", "github.com/unclebob/crap4go/cmd/crap4go@latest"}
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
	wantArgs := []string{"run", "github.com/unclebob/mutate4go/cmd/mutate4go@latest", "main.go", "--scan"}
	if runnerCall := runner.call; runnerCall.dir != "/repo" || runnerCall.name != "go" || !reflect.DeepEqual(runnerCall.args, wantArgs) {
		t.Fatalf("call = %#v, want dir=/repo name=go args=%#v", runnerCall, wantArgs)
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
	output []byte
	stderr []byte
	err    error
}

func (runner *fakeRunner) Run(_ context.Context, dir string, name string, args ...string) (CommandResult, error) {
	runner.call = fakeCall{dir: dir, name: name, args: args}
	return CommandResult{Stdout: runner.output, Stderr: runner.stderr}, runner.err
}

type fakeCall struct {
	dir  string
	name string
	args []string
}
