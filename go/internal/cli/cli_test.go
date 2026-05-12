package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dutifuldev/slophammer/go/internal/app"
)

func TestRunHelp(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"help"}, &out, &errOut)

	if code != app.ExitOK {
		t.Fatalf("code = %d, want %d", code, app.ExitOK)
	}
	if !strings.Contains(out.String(), "slophammer check") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestRunCheckParsesFormatAfterPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", "# Test\n")
	writeFile(t, root, "AGENTS.md", "# Agents\n")
	writeFile(t, root, ".github/workflows/ci.yml", "name: CI\n")

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"check", root, "--format", "json"}, &out, &errOut)

	if code != app.ExitOK {
		t.Fatalf("code = %d, want %d; stderr=%q", code, app.ExitOK, errOut.String())
	}
	if !strings.Contains(out.String(), `"ok": true`) {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestRunExplain(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"explain", "repo.ci-required"}, &out, &errOut)

	if code != app.ExitOK {
		t.Fatalf("code = %d, want %d; stderr=%q", code, app.ExitOK, errOut.String())
	}
	if !strings.Contains(out.String(), "repo.ci-required") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestRunExplainRejectsWrongArity(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"explain"}, &out, &errOut)

	if code != app.ExitError {
		t.Fatalf("code = %d, want %d", code, app.ExitError)
	}
	if !strings.Contains(errOut.String(), "usage: slophammer explain") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestRunCheckRejectsMissingFormatValue(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"check", ".", "--format"}, &out, &errOut)

	if code != app.ExitError {
		t.Fatalf("code = %d, want %d", code, app.ExitError)
	}
	if !strings.Contains(errOut.String(), "--format requires a value") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestRunCheckRejectsUnknownOption(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"check", "--wat", "."}, &out, &errOut)

	if code != app.ExitError {
		t.Fatalf("code = %d, want %d", code, app.ExitError)
	}
	if !strings.Contains(errOut.String(), "unknown check option") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestRunCheckRejectsDuplicatePath(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"check", ".", ".."}, &out, &errOut)

	if code != app.ExitError {
		t.Fatalf("code = %d, want %d", code, app.ExitError)
	}
	if !strings.Contains(errOut.String(), "exactly one path") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestRunCheckRejectsMissingPath(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"check"}, &out, &errOut)

	if code != app.ExitError {
		t.Fatalf("code = %d, want %d", code, app.ExitError)
	}
	if !strings.Contains(errOut.String(), "usage: slophammer check") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"wat"}, &out, &errOut)

	if code != app.ExitError {
		t.Fatalf("code = %d, want %d", code, app.ExitError)
	}
	if !strings.Contains(errOut.String(), "unknown command") {
		t.Fatalf("stderr = %q", errOut.String())
	}
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
