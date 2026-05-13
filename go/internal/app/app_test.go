package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/dutifuldev/slophammer/go/internal/rules"
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
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
