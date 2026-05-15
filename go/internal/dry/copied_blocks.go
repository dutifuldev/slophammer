package dry

import (
	"go/scanner"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

type tokenEntry struct {
	tag  string
	file string
	line int
}

type tokenSequence struct {
	file   sourceFile
	tokens []tokenEntry
}

type tokenOccurrence struct {
	sequence int
	index    int
}

func findCopiedBlocks(files []sourceFile, options Options) ([]Finding, error) {
	sequences, err := tokenSequences(files)
	if err != nil {
		return nil, err
	}
	matches := copiedBlockMatches(sequences, options.CopiedBlockTokens)
	sortCopiedMatches(matches)
	return copiedBlockFindings(matches), nil
}

func copiedBlockMatches(sequences []tokenSequence, windowSize int) []Finding {
	windows := map[string][]tokenOccurrence{}
	for sequenceIndex, sequence := range sequences {
		for i := 0; i+windowSize <= len(sequence.tokens); i++ {
			key := tokenWindowKey(sequence.tokens[i : i+windowSize])
			windows[key] = append(windows[key], tokenOccurrence{sequence: sequenceIndex, index: i})
		}
	}

	seen := map[string]bool{}
	var matches []Finding
	for _, occurrences := range windows {
		matches = append(matches, matchesForOccurrences(sequences, occurrences, windowSize, seen)...)
	}
	return matches
}

func matchesForOccurrences(
	sequences []tokenSequence,
	occurrences []tokenOccurrence,
	windowSize int,
	seen map[string]bool,
) []Finding {
	var matches []Finding
	for i := 0; i < len(occurrences); i++ {
		for j := i + 1; j < len(occurrences); j++ {
			match, ok := copiedBlockMatch(sequences, occurrences[i], occurrences[j], windowSize, seen)
			if ok {
				matches = append(matches, match)
			}
		}
	}
	return matches
}

func copiedBlockMatch(
	sequences []tokenSequence,
	left tokenOccurrence,
	right tokenOccurrence,
	windowSize int,
	seen map[string]bool,
) (Finding, bool) {
	if occurrenceLess(right, left, sequences) {
		left, right = right, left
	}
	match := expandTokenMatch(sequences, left, right, windowSize)
	if match.Tokens < windowSize {
		return Finding{}, false
	}
	key := copiedFindingKey(match)
	if seen[key] {
		return Finding{}, false
	}
	seen[key] = true
	return match, true
}

func sortCopiedMatches(matches []Finding) {
	sort.Slice(matches, func(i, j int) bool {
		left := matches[i]
		right := matches[j]
		if left.Tokens != right.Tokens {
			return left.Tokens > right.Tokens
		}
		return rangeKey(left.Left, left.Right) < rangeKey(right.Left, right.Right)
	})
}

func copiedBlockFindings(matches []Finding) []Finding {
	var findings []Finding
	for _, match := range matches {
		if overlapsCopiedFinding(match, findings) {
			continue
		}
		findings = append(findings, match)
	}
	sort.Slice(findings, func(i, j int) bool {
		return rangeKey(findings[i].Left, findings[i].Right) < rangeKey(findings[j].Left, findings[j].Right)
	})
	return findings
}

func tokenSequences(files []sourceFile) ([]tokenSequence, error) {
	sequences := make([]tokenSequence, 0, len(files))
	for _, file := range files {
		fileSet := token.NewFileSet()
		scanFile := fileSet.AddFile(file.Path, fileSet.Base(), len(file.Content))
		var lexer scanner.Scanner
		lexer.Init(scanFile, file.Content, nil, 0)
		var tokens []tokenEntry
		for {
			pos, tok, lit := lexer.Scan()
			if tok == token.EOF {
				break
			}
			tag, ok := normalizedToken(tok, lit)
			if !ok {
				continue
			}
			position := fileSet.Position(pos)
			tokens = append(tokens, tokenEntry{tag: tag, file: file.Path, line: position.Line})
		}
		sequences = append(sequences, tokenSequence{file: file, tokens: tokens})
	}
	return sequences, nil
}

func normalizedToken(tok token.Token, lit string) (string, bool) {
	switch tok {
	case token.COMMENT, token.SEMICOLON:
		return "", false
	case token.IDENT:
		return "ident/" + lit, true
	case token.INT, token.FLOAT, token.IMAG, token.CHAR, token.STRING:
		return "literal/" + tok.String() + "/" + lit, true
	default:
		return tok.String(), true
	}
}

func tokenWindowKey(tokens []tokenEntry) string {
	var builder strings.Builder
	for _, token := range tokens {
		builder.WriteString(token.tag)
		builder.WriteByte(0)
	}
	return builder.String()
}

func occurrenceLess(left, right tokenOccurrence, sequences []tokenSequence) bool {
	leftFile := sequences[left.sequence].file.Path
	rightFile := sequences[right.sequence].file.Path
	if leftFile != rightFile {
		return leftFile < rightFile
	}
	return left.index < right.index
}

func expandTokenMatch(sequences []tokenSequence, left, right tokenOccurrence, windowSize int) Finding {
	leftTokens := sequences[left.sequence].tokens
	rightTokens := sequences[right.sequence].tokens
	startLeft, startRight := expandTokenStart(leftTokens, rightTokens, left, right, windowSize)
	endLeft, endRight := expandTokenEnd(leftTokens, rightTokens, left, right, startLeft, startRight, windowSize)

	return Finding{
		Kind:   "copied-block",
		Left:   tokenRange(leftTokens[startLeft:endLeft]),
		Right:  tokenRange(rightTokens[startRight:endRight]),
		Tokens: endLeft - startLeft,
		Engine: "token-window",
	}
}

func expandTokenStart(
	leftTokens []tokenEntry,
	rightTokens []tokenEntry,
	left tokenOccurrence,
	right tokenOccurrence,
	windowSize int,
) (int, int) {
	startLeft := left.index
	startRight := right.index
	for startLeft > 0 && startRight > 0 && leftTokens[startLeft-1].tag == rightTokens[startRight-1].tag {
		if left.sequence == right.sequence && rangesOverlap(startLeft-1, left.index+windowSize-1, startRight-1, right.index+windowSize-1) {
			break
		}
		startLeft--
		startRight--
	}
	return startLeft, startRight
}

func expandTokenEnd(
	leftTokens []tokenEntry,
	rightTokens []tokenEntry,
	left tokenOccurrence,
	right tokenOccurrence,
	startLeft int,
	startRight int,
	windowSize int,
) (int, int) {
	endLeft := left.index + windowSize
	endRight := right.index + windowSize
	for endLeft < len(leftTokens) && endRight < len(rightTokens) && leftTokens[endLeft].tag == rightTokens[endRight].tag {
		if left.sequence == right.sequence && rangesOverlap(startLeft, endLeft, startRight, endRight) {
			break
		}
		endLeft++
		endRight++
	}
	return endLeft, endRight
}

func rangesOverlap(leftStart, leftEnd, rightStart, rightEnd int) bool {
	return max(leftStart, rightStart) <= min(leftEnd, rightEnd)
}

func tokenRange(tokens []tokenEntry) Range {
	return Range{
		Path:      tokens[0].file,
		StartLine: tokens[0].line,
		EndLine:   tokens[len(tokens)-1].line,
	}
}

func copiedFindingKey(finding Finding) string {
	return strings.Join([]string{
		finding.Left.Path,
		strconv.Itoa(finding.Left.StartLine),
		strconv.Itoa(finding.Left.EndLine),
		finding.Right.Path,
		strconv.Itoa(finding.Right.StartLine),
		strconv.Itoa(finding.Right.EndLine),
		strconv.Itoa(finding.Tokens),
	}, "|")
}

func overlapsCopiedFinding(candidate Finding, findings []Finding) bool {
	for _, existing := range findings {
		if existing.Kind != "copied-block" {
			continue
		}
		if sameRangePair(candidate.Left, candidate.Right, existing.Left, existing.Right) ||
			sameRangePair(candidate.Left, candidate.Right, existing.Right, existing.Left) {
			return true
		}
	}
	return false
}

func sameRangePair(left, right, existingLeft, existingRight Range) bool {
	return left.Path == existingLeft.Path &&
		right.Path == existingRight.Path &&
		rangesOverlapRange(left, existingLeft) &&
		rangesOverlapRange(right, existingRight)
}
