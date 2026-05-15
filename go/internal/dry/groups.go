package dry

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func groupFindings(findings []Finding) []Group {
	var groups []Group
	for i, finding := range findings {
		added := false
		for groupIndex := range groups {
			if overlapsGroup(finding, groups[groupIndex]) {
				groups[groupIndex].Findings = append(groups[groupIndex].Findings, i)
				groups[groupIndex].Kinds = appendKind(groups[groupIndex].Kinds, finding.Kind)
				groups[groupIndex].Left = mergeRange(groups[groupIndex].Left, finding.Left)
				groups[groupIndex].Right = mergeRange(groups[groupIndex].Right, finding.Right)
				added = true
				break
			}
		}
		if added {
			continue
		}
		groups = append(groups, Group{
			ID:       fmt.Sprintf("dry-%d", len(groups)+1),
			Findings: []int{i},
			Kinds:    []string{finding.Kind},
			Left:     finding.Left,
			Right:    finding.Right,
		})
	}
	return groups
}

func overlapsGroup(finding Finding, group Group) bool {
	return rangesOverlapSamePair(finding.Left, finding.Right, group.Left, group.Right) ||
		rangesOverlapSamePair(finding.Left, finding.Right, group.Right, group.Left)
}

func rangesOverlapSamePair(leftA, rightA, leftB, rightB Range) bool {
	return rangesOverlapRange(leftA, leftB) && rangesOverlapRange(rightA, rightB)
}

func rangesOverlapRange(left, right Range) bool {
	return left.Path == right.Path && max(left.StartLine, right.StartLine) <= min(left.EndLine, right.EndLine)
}

func appendKind(kinds []string, kind string) []string {
	for _, existing := range kinds {
		if existing == kind {
			return kinds
		}
	}
	kinds = append(kinds, kind)
	sort.Strings(kinds)
	return kinds
}

func mergeRange(left, right Range) Range {
	if left.Path != right.Path {
		return left
	}
	return Range{
		Path:      left.Path,
		StartLine: min(left.StartLine, right.StartLine),
		EndLine:   max(left.EndLine, right.EndLine),
	}
}

func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		left := findings[i]
		right := findings[j]
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if left.Score != right.Score {
			return left.Score > right.Score
		}
		if left.Tokens != right.Tokens {
			return left.Tokens > right.Tokens
		}
		return rangeKey(left.Left, left.Right) < rangeKey(right.Left, right.Right)
	})
}

func rangeKey(left, right Range) string {
	return strings.Join([]string{
		left.Path,
		strconv.Itoa(left.StartLine),
		strconv.Itoa(left.EndLine),
		right.Path,
		strconv.Itoa(right.StartLine),
		strconv.Itoa(right.EndLine),
	}, "|")
}
