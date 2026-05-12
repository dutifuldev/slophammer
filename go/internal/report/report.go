package report

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/dutifuldev/slophammer/go/internal/rules"
)

func WriteJSON(out io.Writer, report rules.Report) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func WriteText(out io.Writer, report rules.Report) error {
	if report.OK {
		_, err := fmt.Fprintln(out, "OK: no findings")
		return err
	}
	for _, finding := range report.Findings {
		if _, err := fmt.Fprintf(out, "%s %s %s: %s\n", finding.Severity, finding.RuleID, finding.Path, finding.Message); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(out, "\n%d finding(s)\n", len(report.Findings))
	return err
}
