package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/dutifuldev/slophammer/go/internal/config"
	"github.com/dutifuldev/slophammer/go/internal/gotargets"
	"github.com/dutifuldev/slophammer/go/internal/repo"
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
	Root    string
	Format  string
	Execute bool
}

type RulesOptions struct {
	Format string
}

func Check(ctx context.Context, options CheckOptions, out io.Writer, errOut io.Writer) int {
	return check(ctx, options, out, errOut, toolchecks.ExecRunner{})
}

func check(ctx context.Context, options CheckOptions, out io.Writer, errOut io.Writer, runner toolchecks.Runner) int {
	snapshot, err := scan.Repo(options.Root)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "scan failed: %v\n", err)
		return ExitError
	}
	cfg, err := config.Load(snapshot)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "config failed: %v\n", err)
		return ExitError
	}
	result := rules.RunWithConfig(ctx, snapshot, rules.DefaultRules(), cfg)
	if options.Execute {
		findings := append([]rules.Finding(nil), result.Findings...)
		findings = append(findings, executeGoChecks(ctx, snapshot, options.Root, cfg, runner)...)
		result = rules.NewReport(findings)
	}
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

func Rules(options RulesOptions, out io.Writer, errOut io.Writer) int {
	switch options.Format {
	case "", "text":
		return writeRulesText(out, errOut)
	case "json":
		return writeRulesJSON(out, errOut)
	default:
		_, _ = fmt.Fprintf(errOut, "unsupported rules format: %s\n", options.Format)
		return ExitError
	}
}

func writeRulesText(out io.Writer, errOut io.Writer) int {
	writer := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "RULE ID\tCATEGORY\tSEVERITY\tSTATUS\tTOOL")
	for _, definition := range rules.DefaultDefinitions() {
		_, _ = fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\n",
			definition.ID,
			definition.Category,
			definition.Severity,
			definition.Status,
			definition.Tool,
		)
	}
	if err := writer.Flush(); err != nil {
		_, _ = fmt.Fprintf(errOut, "write failed: %v\n", err)
		return ExitError
	}
	return ExitOK
}

func writeRulesJSON(out io.Writer, errOut io.Writer) int {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(rules.DefaultDefinitions()); err != nil {
		_, _ = fmt.Fprintf(errOut, "write failed: %v\n", err)
		return ExitError
	}
	return ExitOK
}

func CheckGoDry(ctx context.Context, options toolchecks.DryOptions, out io.Writer, errOut io.Writer) int {
	return runConfiguredGoTool(ctx, options, out, errOut, applyDryConfig, checkDryInModules)
}

func CheckGoCRAP(ctx context.Context, options toolchecks.CRAPOptions, out io.Writer, errOut io.Writer) int {
	return runConfiguredGoTool(ctx, options, out, errOut, applyCRAPConfig, checkCRAPInModules)
}

func CheckGoMutation(ctx context.Context, options toolchecks.MutationOptions, out io.Writer, errOut io.Writer) int {
	return runConfiguredGoTool(ctx, options, out, errOut, applyMutationConfig, checkMutationInModules)
}

func writeReport(out io.Writer, format string, result rules.Report) error {
	switch format {
	case "", "text":
		return report.WriteText(out, result)
	case "json":
		return report.WriteJSON(out, result)
	case "sarif":
		return report.WriteSARIF(out, result)
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func runWithCommandConfig(root string, errOut io.Writer, run func(repo.Snapshot, config.Config) int) int {
	snapshot, err := scan.Repo(commandRoot(root))
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "config failed: %v\n", err)
		return ExitError
	}
	cfg, err := config.Load(snapshot)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "config failed: %v\n", err)
		return ExitError
	}
	return run(snapshot, cfg)
}

func runConfiguredGoTool[T interface{ RootPath() string }](
	ctx context.Context,
	options T,
	out io.Writer,
	errOut io.Writer,
	apply func(*T, config.Config),
	run func(context.Context, repo.Snapshot, T, io.Writer, io.Writer, toolchecks.Runner) int,
) int {
	return runWithCommandConfig(options.RootPath(), errOut, func(snapshot repo.Snapshot, cfg config.Config) int {
		apply(&options, cfg)
		return run(ctx, snapshot, options, out, errOut, toolchecks.ExecRunner{})
	})
}

func commandRoot(root string) string {
	if root == "" {
		return "."
	}
	return root
}

func applyDryConfig(options *toolchecks.DryOptions, cfg config.Config) {
	if !options.MaximumSet && cfg.Go.DRYMaxCandidatesSet {
		options.MaximumCandidates = cfg.Go.DRYMaxCandidates
		options.MaximumSet = true
	}
	options.Paths = append([]string(nil), cfg.Go.DRYPaths...)
	options.Exclude = append([]string(nil), cfg.Go.DRYExclude...)
	if cfg.Go.DRY.Structural.EnabledSet {
		options.StructuralEnabled = cfg.Go.DRY.Structural.Enabled
		options.StructuralSet = true
	}
	options.StructuralThreshold = cfg.Go.DRY.Structural.Threshold
	options.StructuralMinLines = cfg.Go.DRY.Structural.MinLines
	options.StructuralMinNodes = cfg.Go.DRY.Structural.MinNodes
	if cfg.Go.DRY.CopiedBlocks.EnabledSet {
		options.CopiedBlockEnabled = cfg.Go.DRY.CopiedBlocks.Enabled
		options.CopiedBlockSet = true
	}
	options.CopiedBlockTokens = cfg.Go.DRY.CopiedBlocks.MinTokens
}

func applyCRAPConfig(options *toolchecks.CRAPOptions, cfg config.Config) {
	if options.MaximumSet || cfg.Go.CRAPMaxScore <= 0 {
		return
	}
	options.MaximumScore = cfg.Go.CRAPMaxScore
	options.MaximumSet = true
}

func applyMutationConfig(options *toolchecks.MutationOptions, cfg config.Config) {
	if options.Target != "" || len(options.Targets) > 0 {
		return
	}
	targets, exclude := cfg.GoMutationScope()
	options.Targets = targets
	options.Exclude = exclude
}

func executeGoChecks(ctx context.Context, snapshot repo.Snapshot, root string, cfg config.Config, runner toolchecks.Runner) []rules.Finding {
	var findings []rules.Finding
	if cfg.Go.DRYMaxCandidatesSet {
		options := toolchecks.DryOptions{
			Root:                commandRoot(root),
			MaximumCandidates:   cfg.Go.DRYMaxCandidates,
			MaximumSet:          true,
			Paths:               append([]string(nil), cfg.Go.DRYPaths...),
			Exclude:             append([]string(nil), cfg.Go.DRYExclude...),
			StructuralEnabled:   cfg.Go.DRY.Structural.Enabled,
			StructuralSet:       cfg.Go.DRY.Structural.EnabledSet,
			StructuralThreshold: cfg.Go.DRY.Structural.Threshold,
			StructuralMinLines:  cfg.Go.DRY.Structural.MinLines,
			StructuralMinNodes:  cfg.Go.DRY.Structural.MinNodes,
			CopiedBlockEnabled:  cfg.Go.DRY.CopiedBlocks.Enabled,
			CopiedBlockSet:      cfg.Go.DRY.CopiedBlocks.EnabledSet,
			CopiedBlockTokens:   cfg.Go.DRY.CopiedBlocks.MinTokens,
		}
		findings = appendToolFinding(findings, rules.GoDryRequiredRuleID, cfg, "DRY check exceeded the configured candidate budget", func(out, errOut io.Writer) int {
			return checkDryInModules(ctx, snapshot, options, out, errOut, runner)
		})
	}
	if cfg.Go.CRAPMaxScore > 0 {
		options := toolchecks.CRAPOptions{
			Root:         commandRoot(root),
			MaximumScore: cfg.Go.CRAPMaxScore,
			MaximumSet:   true,
		}
		findings = appendToolFinding(findings, rules.GoCRAPRequiredRuleID, cfg, "crap4go found functions above the configured score", func(out, errOut io.Writer) int {
			return checkCRAPInModules(ctx, snapshot, options, out, errOut, runner)
		})
	}
	targets, exclude := cfg.GoMutationScope()
	if len(targets) > 0 {
		options := toolchecks.MutationOptions{
			Root:    commandRoot(root),
			Targets: targets,
			Exclude: exclude,
			Scan:    true,
		}
		findings = appendToolFinding(findings, rules.GoMutationRequiredRuleID, cfg, "mutate4go failed for at least one configured target", func(out, errOut io.Writer) int {
			return checkMutationInModules(ctx, snapshot, options, out, errOut, runner)
		})
	}
	return findings
}

func resolveGoMutationTargets(snapshot repo.Snapshot, options toolchecks.MutationOptions) (toolchecks.MutationOptions, error) {
	targets := mutationTargetPatterns(options)
	resolved, err := gotargets.ResolveWithSingleModuleFallback(snapshot, gotargets.Options{
		Targets: targets,
		Exclude: options.Exclude,
	}, goModuleRoots(snapshot), ".")
	if err != nil {
		return toolchecks.MutationOptions{}, err
	}
	options.Target = ""
	options.Targets = resolved
	return options, nil
}

func mutationTargetPatterns(options toolchecks.MutationOptions) []string {
	if options.Target != "" {
		return []string{options.Target}
	}
	return append([]string(nil), options.Targets...)
}

func appendToolFinding(
	findings []rules.Finding,
	ruleID string,
	cfg config.Config,
	message string,
	run func(io.Writer, io.Writer) int,
) []rules.Finding {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run(&out, &errOut)
	if code == ExitOK {
		return findings
	}
	if code == ExitError {
		message = strings.TrimSpace(message + ": " + firstNonEmpty(errOut.String(), out.String()))
	}
	return append(findings, rules.Finding{
		RuleID:   ruleID,
		Severity: rules.Severity(cfg.RuleSeverity(ruleID, string(rules.SeverityError))),
		Path:     "slophammer.yml",
		Message:  message,
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "tool returned an error"
}
