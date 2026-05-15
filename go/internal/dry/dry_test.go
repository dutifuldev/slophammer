package dry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindReportsStructuralFunctionSimilarity(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "left.go", structuralSource("Left", "item%2 == 0"))
	writeFile(t, root, "right.go", structuralSource("Right", "value%2 == 1"))

	report, err := Find(Options{
		Root:               root,
		StructuralEnabled:  true,
		CopiedBlockEnabled: false,
	})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}

	assertFindingKind(t, report, "structural-function")
}

func TestFindReportsCopiedBlockInsideDifferentFunctions(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "handlers.go", copiedBlockSource())

	report, err := Find(Options{
		Root:               root,
		StructuralEnabled:  false,
		CopiedBlockEnabled: true,
		CopiedBlockTokens:  40,
	})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}

	assertFindingKind(t, report, "copied-block")
}

func TestFindGroupsOverlappingStructuralAndCopiedBlockFindings(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "left.go", structuralSource("Left", "item%2 == 0"))
	writeFile(t, root, "right.go", structuralSource("Right", "item%2 == 0"))

	report, err := Find(Options{
		Root:              root,
		CopiedBlockTokens: 20,
	})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}

	if len(report.Groups) == 0 {
		t.Fatal("Groups empty, want grouped findings")
	}
	foundBoth := false
	for _, group := range report.Groups {
		if hasKind(group.Kinds, "structural-function") && hasKind(group.Kinds, "copied-block") {
			foundBoth = true
		}
	}
	if !foundBoth {
		t.Fatalf("groups = %#v, want group with both finding kinds", report.Groups)
	}
}

func TestFindHonorsExplicitDisabledEngines(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "left.go", structuralSource("Left", "item%2 == 0"))
	writeFile(t, root, "right.go", structuralSource("Right", "item%2 == 0"))

	report, err := Find(Options{
		Root:               root,
		StructuralEnabled:  false,
		StructuralSet:      true,
		CopiedBlockEnabled: false,
		CopiedBlockSet:     true,
	})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}

	if len(report.Findings) != 0 {
		t.Fatalf("findings = %#v, want none", report.Findings)
	}
}

func TestFindHonorsExplicitPaths(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "included/left.go", structuralSource("Left", "item%2 == 0"))
	writeFile(t, root, "included/right.go", structuralSource("Right", "item%2 == 0"))
	writeFile(t, root, "ignored/left.go", structuralSource("IgnoredLeft", "item%3 == 0"))
	writeFile(t, root, "ignored/right.go", structuralSource("IgnoredRight", "item%3 == 0"))

	report, err := Find(Options{
		Root:               root,
		Paths:              []string{"included"},
		StructuralEnabled:  true,
		CopiedBlockEnabled: false,
	})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}

	for _, finding := range report.Findings {
		if strings.HasPrefix(finding.Left.Path, "ignored/") || strings.HasPrefix(finding.Right.Path, "ignored/") {
			t.Fatalf("finding includes ignored path: %#v", finding)
		}
	}
	assertFindingKind(t, report, "structural-function")
}

func assertFindingKind(t *testing.T, report Report, kind string) {
	t.Helper()
	for _, finding := range report.Findings {
		if finding.Kind == kind {
			return
		}
	}
	t.Fatalf("missing finding kind %q in %#v", kind, report.Findings)
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

func structuralSource(name string, condition string) string {
	return `package sample

func ` + name + `(items []int) []int {
	var kept []int
	for _, item := range items {
		if ` + condition + ` {
			kept = append(kept, item+1)
		}
	}
	return kept
}
`
}

func copiedBlockSource() string {
	return `package sample

func Resolve(value int) int {
	total := 0
	for i := 0; i < 20; i++ {
		total += i
	}
	if value > 10 {
		total += value
	}
	if value%2 == 0 {
		total += 2
	}
	if value%3 == 0 {
		total += 3
	}
	return total + 1
}

func Abandon(value int) int {
	total := 0
	for i := 0; i < 20; i++ {
		total += i
	}
	if value > 10 {
		total += value
	}
	if value%2 == 0 {
		total += 2
	}
	if value%3 == 0 {
		total += 3
	}
	return total - 1
}
`
}
