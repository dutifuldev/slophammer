package app

import (
	"context"
	"fmt"
	"io"

	"github.com/dutifuldev/slophammer/go/internal/report"
	"github.com/dutifuldev/slophammer/go/internal/rules"
	"github.com/dutifuldev/slophammer/go/internal/scan"
	"github.com/dutifuldev/slophammer/go/internal/toolchecks"
)

const (
	ExitOK       = 0
	ExitFindings = 1
	ExitError    = 2
)

type CheckOptions struct {
	Root   string
	Format string
}

func Check(ctx context.Context, options CheckOptions, out io.Writer, errOut io.Writer) int {
	snapshot, err := scan.Repo(options.Root)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "scan failed: %v\n", err)
		return ExitError
	}
	result := rules.Run(ctx, snapshot, rules.DefaultRules())
	if err := writeReport(out, options.Format, result); err != nil {
		_, _ = fmt.Fprintf(errOut, "report failed: %v\n", err)
		return ExitError
	}
	if result.OK {
		return ExitOK
	}
	return ExitFindings
}

func Explain(ruleID string, out io.Writer, errOut io.Writer) int {
	text, ok := rules.Explain(rules.DefaultRules(), ruleID)
	if !ok {
		_, _ = fmt.Fprintf(errOut, "unknown rule: %s\n", ruleID)
		return ExitError
	}
	_, err := io.WriteString(out, text)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "write failed: %v\n", err)
		return ExitError
	}
	return ExitOK
}

func CheckGoDry(ctx context.Context, options toolchecks.DryOptions, out io.Writer, errOut io.Writer) int {
	return toolchecks.CheckDry(ctx, options, out, errOut, toolchecks.ExecRunner{})
}

func CheckGoCRAP(ctx context.Context, options toolchecks.CRAPOptions, out io.Writer, errOut io.Writer) int {
	return toolchecks.CheckCRAP(ctx, options, out, errOut, toolchecks.ExecRunner{})
}

func CheckGoMutation(ctx context.Context, options toolchecks.MutationOptions, out io.Writer, errOut io.Writer) int {
	return toolchecks.CheckMutation(ctx, options, out, errOut, toolchecks.ExecRunner{})
}

func writeReport(out io.Writer, format string, result rules.Report) error {
	switch format {
	case "", "text":
		return report.WriteText(out, result)
	case "json":
		return report.WriteJSON(out, result)
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}
