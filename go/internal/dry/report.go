package dry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func WriteJSON(report Report) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

func FormatText(report Report) string {
	if len(report.Findings) == 0 {
		return "No DRY findings found.\n"
	}
	var out bytes.Buffer
	_, _ = fmt.Fprintf(&out, "DRY findings: %d\n", len(report.Findings))
	_, _ = fmt.Fprintf(&out, "Groups: %d\n", len(report.Groups))
	_, _ = fmt.Fprintf(&out, "Structural function findings: %d\n", countKind(report.Findings, "structural-function"))
	_, _ = fmt.Fprintf(&out, "Copied block findings: %d\n", countKind(report.Findings, "copied-block"))
	_, _ = fmt.Fprintf(&out, "Found by both: %d\n", countGroupsWithBoth(report.Groups))
	for _, group := range report.Groups {
		_, _ = fmt.Fprintf(&out, "\nDRY %s [%s]\n", group.ID, strings.Join(group.Kinds, ", "))
		_, _ = fmt.Fprintf(&out, "  %s:%d-%d\n", group.Left.Path, group.Left.StartLine, group.Left.EndLine)
		_, _ = fmt.Fprintf(&out, "  %s:%d-%d\n", group.Right.Path, group.Right.StartLine, group.Right.EndLine)
	}
	return out.String()
}

func countKind(findings []Finding, kind string) int {
	count := 0
	for _, finding := range findings {
		if finding.Kind == kind {
			count++
		}
	}
	return count
}

func countGroupsWithBoth(groups []Group) int {
	count := 0
	for _, group := range groups {
		if hasKind(group.Kinds, "structural-function") && hasKind(group.Kinds, "copied-block") {
			count++
		}
	}
	return count
}

func hasKind(kinds []string, want string) bool {
	for _, kind := range kinds {
		if kind == want {
			return true
		}
	}
	return false
}
