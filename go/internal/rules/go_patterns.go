package rules

import "regexp"

var (
	shellNamePattern               = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	coverageThresholdPattern       = regexp.MustCompile(`(?im)\b(total|cover|coverage|minimum|threshold|required)\b[^\n]*(>=|<=|-ge\b|-le\b|-gt\b|-lt\b)|(?:>=|<=|-ge\b|-le\b|-gt\b|-lt\b)[^\n]*\b(total|cover|coverage|minimum|threshold|required)\b`)
	crapThresholdPattern           = regexp.MustCompile(`(?im)\b(crap|maximum|max|score)\b[^\n]*(>=|<=|-ge\b|-le\b|-gt\b|-lt\b)|(?:>=|<=|-ge\b|-le\b|-gt\b|-lt\b)[^\n]*\b(crap|maximum|max|score)\b`)
	strictCoverageThresholdPattern = regexp.MustCompile(`(?im)\b(total|minimum|threshold|required)\b[^\n]*(>|<)[^\n]*(\b(total|minimum|threshold|required)\b|[0-9]+(?:\.[0-9]+)?)|([0-9]+(?:\.[0-9]+)?|\b(total|minimum|threshold|required)\b)[^\n]*(>|<)[^\n]*\b(total|minimum|threshold|required)\b`)
	strictCRAPThresholdPattern     = regexp.MustCompile(`(?im)\b(crap|score|maximum|max)\b[^\n]*(>|<)[^\n]*(\b(score|maximum|max)\b|[0-9]+(?:\.[0-9]+)?)`)
)
