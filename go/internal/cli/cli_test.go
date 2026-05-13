package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dutifuldev/slophammer/go/internal/app"
)

func TestRunHelp(t *testing.T) {
	result := runCLI(t, "help")

	if result.code != app.ExitOK {
		t.Fatalf("code = %d, want %d", result.code, app.ExitOK)
	}
	if !strings.Contains(result.stdout, "slophammer check") {
		t.Fatalf("stdout = %q", result.stdout)
	}
}

func TestRunCheckParsesFormatAfterPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", "# Test\n")
	writeFile(t, root, "AGENTS.md", "# Agents\n")
	writeFile(t, root, ".github/workflows/ci.yml", "name: CI\n")

	result := runCLI(t, "check", root, "--format", "json")

	if result.code != app.ExitOK {
		t.Fatalf("code = %d, want %d; stderr=%q", result.code, app.ExitOK, result.stderr)
	}
	if !strings.Contains(result.stdout, `"ok": true`) {
		t.Fatalf("stdout = %q", result.stdout)
	}
}

func TestRunExplain(t *testing.T) {
	result := runCLI(t, "explain", "repo.ci-required")

	if result.code != app.ExitOK {
		t.Fatalf("code = %d, want %d; stderr=%q", result.code, app.ExitOK, result.stderr)
	}
	if !strings.Contains(result.stdout, "repo.ci-required") {
		t.Fatalf("stdout = %q", result.stdout)
	}
}

func TestRunExplainRejectsWrongArity(t *testing.T) {
	assertCLIError(t, []string{"explain"}, "usage: slophammer explain")
}

func TestRunGoRejectsMissingSubcommand(t *testing.T) {
	assertCLIError(t, []string{"go"}, "slophammer go dry")
}

func TestParseGoDryArgs(t *testing.T) {
	var errOut bytes.Buffer
	options, ok := parseGoDryArgs([]string{"/repo", "--max-candidates", "12", "--show-report"}, &errOut)

	if !ok {
		t.Fatalf("ok = false; stderr=%q", errOut.String())
	}
	if options.Root != "/repo" || options.MaximumCandidates != 12 || !options.MaximumSet || !options.ShowReport {
		t.Fatalf("options = %#v", options)
	}
}

func TestParseGoCRAPArgs(t *testing.T) {
	var errOut bytes.Buffer
	options, ok := parseGoCRAPArgs([]string{"/repo", "--max-score", "25.5"}, &errOut)

	if !ok {
		t.Fatalf("ok = false; stderr=%q", errOut.String())
	}
	if options.Root != "/repo" || options.MaximumScore != 25.5 || !options.MaximumSet {
		t.Fatalf("options = %#v", options)
	}
}

func TestParseGoMutationArgs(t *testing.T) {
	var errOut bytes.Buffer
	options, ok := parseGoMutationArgs([]string{"/repo", "--target", "main.go", "--scan"}, &errOut)

	if !ok {
		t.Fatalf("ok = false; stderr=%q", errOut.String())
	}
	if options.Root != "/repo" || options.Target != "main.go" || !options.Scan {
		t.Fatalf("options = %#v", options)
	}
}

func TestParseGoMutationArgsRequiresTarget(t *testing.T) {
	var errOut bytes.Buffer
	_, ok := parseGoMutationArgs([]string{"/repo", "--scan"}, &errOut)

	if ok {
		t.Fatal("ok = true, want false")
	}
	if !strings.Contains(errOut.String(), "--target cannot be empty") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestParseGoMutationArgsRejectsFlagTarget(t *testing.T) {
	var errOut bytes.Buffer
	_, ok := parseGoMutationArgs([]string{"/repo", "--target", "--scan"}, &errOut)

	if ok {
		t.Fatal("ok = true, want false")
	}
	if !strings.Contains(errOut.String(), "--target requires a file value") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestParseGoToolArgsRejectInvalidNumbers(t *testing.T) {
	tests := []struct {
		name string
		run  func(io.Writer) bool
		want string
	}{
		{
			name: "dry",
			run: func(errOut io.Writer) bool {
				_, ok := parseGoDryArgs([]string{"--max-candidates", "x"}, errOut)
				return ok
			},
			want: "--max-candidates must be a non-negative integer",
		},
		{
			name: "crap",
			run: func(errOut io.Writer) bool {
				_, ok := parseGoCRAPArgs([]string{"--max-score", "-1"}, errOut)
				return ok
			},
			want: "--max-score must be a non-negative number",
		},
		{
			name: "crap NaN",
			run: func(errOut io.Writer) bool {
				_, ok := parseGoCRAPArgs([]string{"--max-score", "NaN"}, errOut)
				return ok
			},
			want: "--max-score must be a non-negative number",
		},
		{
			name: "crap infinity",
			run: func(errOut io.Writer) bool {
				_, ok := parseGoCRAPArgs([]string{"--max-score", "+Inf"}, errOut)
				return ok
			},
			want: "--max-score must be a non-negative number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errOut bytes.Buffer
			if tt.run(&errOut) {
				t.Fatal("ok = true, want false")
			}
			if !strings.Contains(errOut.String(), tt.want) {
				t.Fatalf("stderr = %q", errOut.String())
			}
		})
	}
}

func TestRunCheckRejectsMissingFormatValue(t *testing.T) {
	assertCLIError(t, []string{"check", ".", "--format"}, "--format requires a value")
}

func TestRunCheckRejectsUnknownOption(t *testing.T) {
	assertCLIError(t, []string{"check", "--wat", "."}, "unknown check option")
}

func TestRunCheckRejectsDuplicatePath(t *testing.T) {
	assertCLIError(t, []string{"check", ".", ".."}, "exactly one path")
}

func TestRunCheckRejectsMissingPath(t *testing.T) {
	assertCLIError(t, []string{"check"}, "usage: slophammer check")
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	assertCLIError(t, []string{"wat"}, "unknown command")
}

func assertCLIError(t *testing.T, args []string, stderr string) {
	t.Helper()
	result := runCLI(t, args...)
	if result.code != app.ExitError {
		t.Fatalf("code = %d, want %d", result.code, app.ExitError)
	}
	if !strings.Contains(result.stderr, stderr) {
		t.Fatalf("stderr = %q", result.stderr)
	}
}

func runCLI(t *testing.T, args ...string) cliResult {
	t.Helper()
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), args, &out, &errOut)
	return cliResult{code: code, stdout: out.String(), stderr: errOut.String()}
}

type cliResult struct {
	code   int
	stdout string
	stderr string
}

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
