package cli

import (
	"context"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/dutifuldev/slophammer/go/internal/app"
	"github.com/dutifuldev/slophammer/go/internal/toolchecks"
)

func Run(ctx context.Context, args []string, out io.Writer, errOut io.Writer) int {
	if len(args) == 0 {
		printUsage(errOut)
		return app.ExitError
	}
	run, ok := rootCommand(args[0])
	if !ok {
		_, _ = fmt.Fprintf(errOut, "unknown command: %s\n", args[0])
		printUsage(errOut)
		return app.ExitError
	}
	return run(ctx, args[1:], out, errOut)
}

func rootCommand(name string) (goCommandRunner, bool) {
	if run, ok := goSubcommand(name); ok {
		return run, true
	}
	switch name {
	case "check":
		return runCheck, true
	case "explain":
		return runExplainCommand, true
	case "rules":
		return runRules, true
	case "go":
		return runGo, true
	case "-h", "--help", "help":
		return runHelp, true
	default:
		return nil, false
	}
}

func runHelp(_ context.Context, _ []string, out io.Writer, _ io.Writer) int {
	printUsage(out)
	return app.ExitOK
}

func runCheck(ctx context.Context, args []string, out io.Writer, errOut io.Writer) int {
	return runParsed(ctx, args, out, errOut, parseCheckArgs, app.Check)
}

func runExplain(args []string, out io.Writer, errOut io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(errOut, "usage: slophammer-go explain <rule-id>")
		return app.ExitError
	}
	return app.Explain(args[0], out, errOut)
}

func runExplainCommand(_ context.Context, args []string, out io.Writer, errOut io.Writer) int {
	return runExplain(args, out, errOut)
}

func runRules(_ context.Context, args []string, out io.Writer, errOut io.Writer) int {
	options, ok := parseRulesArgs(args, errOut)
	if !ok {
		return app.ExitError
	}
	return app.Rules(options, out, errOut)
}

func parseRulesArgs(args []string, errOut io.Writer) (app.RulesOptions, bool) {
	options := app.RulesOptions{Format: "text"}
	for i := 0; i < len(args); i++ {
		advance, ok := parseRulesArg(&options, args, i, errOut)
		if !ok {
			return app.RulesOptions{}, false
		}
		i += advance
	}
	return options, true
}

func parseRulesArg(options *app.RulesOptions, args []string, index int, errOut io.Writer) (int, bool) {
	switch args[index] {
	case "--format":
		return parseRulesFormat(options, args, index, errOut)
	case "--json":
		options.Format = "json"
		return 0, true
	default:
		_, _ = fmt.Fprintln(errOut, "usage: slophammer-go rules [--format text|json]")
		return 0, false
	}
}

func parseRulesFormat(options *app.RulesOptions, args []string, index int, errOut io.Writer) (int, bool) {
	value, ok := nextArg(args, index)
	if !ok {
		_, _ = fmt.Fprintln(errOut, "--format requires a value")
		return 0, false
	}
	if value != "text" && value != "json" {
		_, _ = fmt.Fprintf(errOut, "unsupported rules format: %s\n", value)
		return 0, false
	}
	options.Format = value
	return 1, true
}

func runGo(ctx context.Context, args []string, out io.Writer, errOut io.Writer) int {
	if len(args) == 0 {
		printGoUsage(errOut)
		return app.ExitError
	}
	if run, ok := goSubcommand(args[0]); ok {
		return run(ctx, args[1:], out, errOut)
	}
	_, _ = fmt.Fprintf(errOut, "unknown go command: %s\n", args[0])
	printGoUsage(errOut)
	return app.ExitError
}

type goCommandRunner func(context.Context, []string, io.Writer, io.Writer) int

func goSubcommand(name string) (goCommandRunner, bool) {
	commands := map[string]goCommandRunner{
		"dry":    runGoDry,
		"crap":   runGoCRAP,
		"mutate": runGoMutation,
	}
	run, ok := commands[name]
	return run, ok
}

func runGoDry(ctx context.Context, args []string, out io.Writer, errOut io.Writer) int {
	return runParsed(ctx, args, out, errOut, parseGoDryArgs, app.CheckGoDry)
}

func runGoCRAP(ctx context.Context, args []string, out io.Writer, errOut io.Writer) int {
	return runParsed(ctx, args, out, errOut, parseGoCRAPArgs, app.CheckGoCRAP)
}

func runGoMutation(ctx context.Context, args []string, out io.Writer, errOut io.Writer) int {
	return runParsed(ctx, args, out, errOut, parseGoMutationArgs, app.CheckGoMutation)
}

func runParsed[T any](
	ctx context.Context,
	args []string,
	out io.Writer,
	errOut io.Writer,
	parse func([]string, io.Writer) (T, bool),
	check func(context.Context, T, io.Writer, io.Writer) int,
) int {
	options, ok := parse(args, errOut)
	if !ok {
		return app.ExitError
	}
	return check(ctx, options, out, errOut)
}

func parseCheckArgs(args []string, errOut io.Writer) (app.CheckOptions, bool) {
	options := app.CheckOptions{Format: "text"}
	for i := 0; i < len(args); i++ {
		advance, ok := parseCheckArg(&options, args, i, errOut)
		if !ok {
			return app.CheckOptions{}, false
		}
		i += advance
	}
	if options.Root == "" {
		_, _ = fmt.Fprintln(errOut, "usage: slophammer-go check <path> [--format text|json|sarif] [--execute]")
		return app.CheckOptions{}, false
	}
	return options, true
}

func parseCheckArg(options *app.CheckOptions, args []string, index int, errOut io.Writer) (int, bool) {
	switch args[index] {
	case "--format":
		value, ok := nextArg(args, index)
		if !ok {
			_, _ = fmt.Fprintln(errOut, "--format requires a value")
			return 0, false
		}
		options.Format = value
		return 1, true
	case "--json":
		options.Format = "json"
		return 0, true
	case "--execute":
		options.Execute = true
		return 0, true
	default:
		return 0, parseCheckPath(options, args[index], errOut)
	}
}

func parseCheckPath(options *app.CheckOptions, arg string, errOut io.Writer) bool {
	if len(arg) > 0 && arg[0] == '-' {
		_, _ = fmt.Fprintf(errOut, "unknown check option: %s\n", arg)
		return false
	}
	if options.Root != "" {
		_, _ = fmt.Fprintln(errOut, "check accepts exactly one path")
		return false
	}
	options.Root = arg
	return true
}

func parseGoDryArgs(args []string, errOut io.Writer) (toolchecks.DryOptions, bool) {
	return parseGoToolArgs(args, errOut, toolchecks.DryOptions{MaximumCandidates: toolchecks.DefaultMaximumDRYCandidates}, parseGoDryArg)
}

func parseGoDryArg(options *toolchecks.DryOptions, args []string, index int, errOut io.Writer) (int, bool) {
	switch args[index] {
	case "--show-report":
		options.ShowReport = true
		return 0, true
	case "--format":
		return parseGoDryFormat(options, args, index, errOut)
	case "--max-candidates":
		value, ok := parseNonNegativeIntFlag(args, index, "--max-candidates", errOut)
		if !ok {
			return 0, false
		}
		options.MaximumCandidates = value
		options.MaximumSet = true
		return 1, true
	default:
		root, ok := parseSinglePathOption(options.Root, args[index], "go dry", errOut)
		options.Root = root
		return 0, ok
	}
}

func parseGoDryFormat(options *toolchecks.DryOptions, args []string, index int, errOut io.Writer) (int, bool) {
	if index+1 >= len(args) {
		_, _ = fmt.Fprintln(errOut, "--format requires a value")
		return 0, false
	}
	switch args[index+1] {
	case "json", "text":
		options.Format = args[index+1]
		return 1, true
	default:
		_, _ = fmt.Fprintf(errOut, "unsupported go dry format: %s\n", args[index+1])
		return 0, false
	}
}

func parseGoCRAPArgs(args []string, errOut io.Writer) (toolchecks.CRAPOptions, bool) {
	return parseGoToolArgs(args, errOut, toolchecks.CRAPOptions{MaximumScore: toolchecks.DefaultMaximumCRAPScore}, parseGoCRAPArg)
}

func parseGoCRAPArg(options *toolchecks.CRAPOptions, args []string, index int, errOut io.Writer) (int, bool) {
	if args[index] == "--max-score" {
		value, ok := parseNonNegativeFloatFlag(args, index, "--max-score", errOut)
		if !ok {
			return 0, false
		}
		options.MaximumScore = value
		options.MaximumSet = true
		return 1, true
	}
	root, ok := parseSinglePathOption(options.Root, args[index], "go crap", errOut)
	options.Root = root
	return 0, ok
}

func parseGoMutationArgs(args []string, errOut io.Writer) (toolchecks.MutationOptions, bool) {
	options := toolchecks.MutationOptions{}
	return parseGoToolArgs(args, errOut, options, parseGoMutationArg)
}

func parseGoToolArgs[T any](
	args []string,
	errOut io.Writer,
	options T,
	parseArg func(*T, []string, int, io.Writer) (int, bool),
) (T, bool) {
	for i := 0; i < len(args); i++ {
		advance, ok := parseArg(&options, args, i, errOut)
		if !ok {
			var zero T
			return zero, false
		}
		i += advance
	}
	return options, true
}

func parseGoMutationArg(options *toolchecks.MutationOptions, args []string, index int, errOut io.Writer) (int, bool) {
	switch args[index] {
	case "--target":
		value, ok := parseTargetFlag(args, index, errOut)
		if !ok {
			return 0, false
		}
		options.Target = value
		return 1, true
	case "--scan":
		options.Scan = true
		return 0, true
	default:
		root, ok := parseSinglePathOption(options.Root, args[index], "go mutate", errOut)
		options.Root = root
		return 0, ok
	}
}

func parseNonNegativeIntFlag(args []string, index int, flag string, errOut io.Writer) (int, bool) {
	valueText, ok := nextArg(args, index)
	if !ok {
		_, _ = fmt.Fprintf(errOut, "%s requires a value\n", flag)
		return 0, false
	}
	value, err := strconv.Atoi(valueText)
	if err != nil || value < 0 {
		_, _ = fmt.Fprintf(errOut, "%s must be a non-negative integer\n", flag)
		return 0, false
	}
	return value, true
}

func parseNonNegativeFloatFlag(args []string, index int, flag string, errOut io.Writer) (float64, bool) {
	valueText, ok := nextArg(args, index)
	if !ok {
		_, _ = fmt.Fprintf(errOut, "%s requires a value\n", flag)
		return 0, false
	}
	value, err := strconv.ParseFloat(valueText, 64)
	if err != nil || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		_, _ = fmt.Fprintf(errOut, "%s must be a non-negative number\n", flag)
		return 0, false
	}
	return value, true
}

func parseTargetFlag(args []string, index int, errOut io.Writer) (string, bool) {
	value, ok := nextArg(args, index)
	if !ok {
		_, _ = fmt.Fprintln(errOut, "--target requires a value")
		return "", false
	}
	if value == "" || value[0] == '-' {
		_, _ = fmt.Fprintln(errOut, "--target requires a file value")
		return "", false
	}
	return value, true
}

func nextArg(args []string, index int) (string, bool) {
	if index+1 >= len(args) {
		return "", false
	}
	return args[index+1], true
}

func parseSinglePathOption(currentRoot string, arg string, command string, errOut io.Writer) (string, bool) {
	if len(arg) > 0 && arg[0] == '-' {
		_, _ = fmt.Fprintf(errOut, "unknown %s option: %s\n", command, arg)
		return "", false
	}
	if currentRoot != "" {
		_, _ = fmt.Fprintf(errOut, "%s accepts exactly one path\n", command)
		return "", false
	}
	return arg, true
}

func printUsage(out io.Writer) {
	_, _ = fmt.Fprintln(out, "usage:")
	_, _ = fmt.Fprintln(out, "  slophammer-go check <path> [--format text|json|sarif] [--execute]")
	_, _ = fmt.Fprintln(out, "  slophammer-go explain <rule-id>")
	_, _ = fmt.Fprintln(out, "  slophammer-go rules [--format text|json]")
	_, _ = fmt.Fprintln(out, "  slophammer-go dry [path] [--max-candidates n] [--show-report] [--format json|text]")
	_, _ = fmt.Fprintln(out, "  slophammer-go crap [path] [--max-score n]")
	_, _ = fmt.Fprintln(out, "  slophammer-go mutate [path] [--target file] [--scan]")
}

func printGoUsage(out io.Writer) {
	_, _ = fmt.Fprintln(out, "usage:")
	_, _ = fmt.Fprintln(out, "  slophammer-go dry [path] [--max-candidates n] [--show-report] [--format json|text]")
	_, _ = fmt.Fprintln(out, "  slophammer-go crap [path] [--max-score n]")
	_, _ = fmt.Fprintln(out, "  slophammer-go mutate [path] [--target file] [--scan]")
}
