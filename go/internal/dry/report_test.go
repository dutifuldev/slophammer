package dry

import (
	"strings"
	"testing"
)

func TestFormatTextSummarizesGroupedFindings(t *testing.T) {
	report := Report{
		Findings: []Finding{
			{
				Kind:  "structural-function",
				Left:  Range{Path: "a.go", StartLine: 1, EndLine: 10},
				Right: Range{Path: "b.go", StartLine: 1, EndLine: 10},
			},
			{
				Kind:  "copied-block",
				Left:  Range{Path: "a.go", StartLine: 2, EndLine: 9},
				Right: Range{Path: "b.go", StartLine: 2, EndLine: 9},
			},
		},
	}
	report.Groups = groupFindings(report.Findings)

	text := FormatText(report)

	for _, want := range []string{
		"DRY findings: 2",
		"Structural function findings: 1",
		"Copied block findings: 1",
		"Found by both: 1",
		"DRY dry-1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("FormatText() missing %q in:\n%s", want, text)
		}
	}
}

func TestCopiedFindingOverlapHandlesSwappedRanges(t *testing.T) {
	existing := []Finding{{
		Kind:  "copied-block",
		Left:  Range{Path: "a.go", StartLine: 10, EndLine: 20},
		Right: Range{Path: "b.go", StartLine: 30, EndLine: 40},
	}}
	candidate := Finding{
		Kind:  "copied-block",
		Left:  Range{Path: "b.go", StartLine: 35, EndLine: 45},
		Right: Range{Path: "a.go", StartLine: 15, EndLine: 25},
	}

	if !overlapsCopiedFinding(candidate, existing) {
		t.Fatal("expected swapped copied ranges to overlap")
	}
}
