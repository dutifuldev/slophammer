package rules

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/dutifuldev/slophammer/go/internal/repo"
)

// nolintDirective is assembled from two literals so this production file does
// not itself contain the marker it scans for.
const nolintDirective = "//" + "nolint"

var generatedGoFilePattern = regexp.MustCompile(`(?m)^// Code generated .* DO NOT EDIT\.$`)

var suppressionExemptSegments = map[string]struct{}{
	"fixtures":  {},
	"templates": {},
	"testdata":  {},
	"vendor":    {},
	"scripts":   {},
	"generated": {},
}

// goSuppressionsRule flags bare nolint directives in production Go code:
// suppressions must carry a trailing explanation comment in the nolintlint
// style or sit under a preceding comment line.
type goSuppressionsRule struct {
	definition Definition
}

func newGoSuppressionsRule(definition Definition) Rule {
	return goSuppressionsRule{definition: definition}
}

func (r goSuppressionsRule) Metadata() Metadata {
	return r.definition.Metadata()
}

func (r goSuppressionsRule) Check(_ context.Context, snapshot repo.Snapshot) []Finding {
	findings := make([]Finding, 0)
	for _, file := range snapshot.FilesWithSuffix(".go") {
		if !productionGoSuppressionPath(file.Path) || generatedGoFile(file.Content) {
			continue
		}
		if line, ok := bareSuppressionLine(file.Content); ok {
			findings = append(findings, r.suppressionFinding(file.Path, line))
		}
	}
	return findings
}

func (r goSuppressionsRule) suppressionFinding(filePath string, line int) Finding {
	return Finding{
		RuleID:   r.definition.ID,
		Severity: r.definition.Severity,
		Path:     filePath,
		Message:  fmt.Sprintf("%s (line %d)", r.definition.Message, line),
	}
}

func productionGoSuppressionPath(filePath string) bool {
	return !strings.HasSuffix(filePath, "_test.go") &&
		!pathHasAnySegment(filePath, suppressionExemptSegments)
}

func generatedGoFile(content string) bool {
	return generatedGoFilePattern.MatchString(content)
}

// bareSuppressionLine returns the first line carrying a nolint directive
// with neither a trailing explanation nor a preceding comment line.
func bareSuppressionLine(content string) (int, bool) {
	previousLineIsComment := false
	for index, line := range strings.Split(content, "\n") {
		if bareNolintLine(line, previousLineIsComment) {
			return index + 1, true
		}
		previousLineIsComment = strings.HasPrefix(strings.TrimSpace(line), "//")
	}
	return 0, false
}

func bareNolintLine(line string, previousLineIsComment bool) bool {
	directive := strings.Index(line, nolintDirective)
	if directive < 0 || previousLineIsComment {
		return false
	}
	return !nolintExplanation(line[directive+len(nolintDirective):])
}

// nolintExplanation reports whether the text after a nolint directive carries
// a non-empty trailing comment, the nolintlint reason convention.
func nolintExplanation(rest string) bool {
	comment := strings.Index(rest, "//")
	if comment < 0 {
		return false
	}
	return strings.TrimSpace(rest[comment+2:]) != ""
}
