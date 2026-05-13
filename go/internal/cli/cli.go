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
	switch args[0] {
	case "check":
		return runCheck(ctx, args[1:], out, errOut)
	case "explain":
		return runExplain(args[1:], out, errOut)
	case "go":
		return runGo(ctx, args[1:], out, errOut)
	case "-h", "--help", "help":
		printUsage(out)
		return app.ExitOK
	default:
		_, _ = fmt.Fprintf(errOut, "unknown command: %s\n", args[0])
		printUsage(errOut)
		return app.ExitError
	}
}

func runCheck(ctx context.Context, args []string, out io.Writer, errOut io.Writer) int {
	return runParsed(ctx, args, out, errOut, parseCheckArgs, app.Check)
}

func runExplain(args []string, out io.Writer, errOut io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(errOut, "usage: slophammer explain <rule-id>")
		return app.ExitError
	}
	return app.Explain(args[0], out, errOut)
}

func runGo(ctx context.Context, args []string, out io.Writer, errOut io.Writer) int {
	if len(args) == 0 {
		printGoUsage(errOut)
		return app.ExitError
	}
	switch args[0] {
	case "dry":
		return runGoDry(ctx, args[1:], out, errOut)
	case "crap":
		return runGoCRAP(ctx, args[1:], out, errOut)
	case "mutate":
		return runGoMutation(ctx, args[1:], out, errOut)
	default:
		_, _ = fmt.Fprintf(errOut, "unknown go command: %s\n", args[0])
		printGoUsage(errOut)
		return app.ExitError
	}
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
		arg := args[i]
		switch arg {
		case "--format":
			if i+1 >= len(args) {
				_, _ = fmt.Fprintln(errOut, "--format requires a value")
				return app.CheckOptions{}, false
			}
			i++
			options.Format = args[i]
		case "--json":
			options.Format = "json"
		default:
			if len(arg) > 0 && arg[0] == '-' {
				_, _ = fmt.Fprintf(errOut, "unknown check option: %s\n", arg)
				return app.CheckOptions{}, false
			}
			if options.Root != "" {
				_, _ = fmt.Fprintln(errOut, "check accepts exactly one path")
				return app.CheckOptions{}, false
			}
			options.Root = arg
		}
	}
	if options.Root == "" {
		_, _ = fmt.Fprintln(errOut, "usage: slophammer check <path> [--format text|json]")
		return app.CheckOptions{}, false
	}
	return options, true
}

func parseGoDryArgs(args []string, errOut io.Writer) (toolchecks.DryOptions, bool) {
	options := toolchecks.DryOptions{MaximumCandidates: toolchecks.DefaultMaximumDRYCandidates}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--show-report":
			options.ShowReport = true
		case "--max-candidates":
			if i+1 >= len(args) {
				_, _ = fmt.Fprintln(errOut, "--max-candidates requires a value")
				return toolchecks.DryOptions{}, false
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value < 0 {
				_, _ = fmt.Fprintln(errOut, "--max-candidates must be a non-negative integer")
				return toolchecks.DryOptions{}, false
			}
			options.MaximumCandidates = value
			options.MaximumSet = true
		default:
			root, ok := parseSinglePathOption(options.Root, arg, "go dry", errOut)
			if !ok {
				return toolchecks.DryOptions{}, false
			}
			options.Root = root
		}
	}
	return options, true
}

func parseGoCRAPArgs(args []string, errOut io.Writer) (toolchecks.CRAPOptions, bool) {
	options := toolchecks.CRAPOptions{MaximumScore: toolchecks.DefaultMaximumCRAPScore}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--max-score":
			if i+1 >= len(args) {
				_, _ = fmt.Fprintln(errOut, "--max-score requires a value")
				return toolchecks.CRAPOptions{}, false
			}
			i++
			value, err := strconv.ParseFloat(args[i], 64)
			if err != nil || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
				_, _ = fmt.Fprintln(errOut, "--max-score must be a non-negative number")
				return toolchecks.CRAPOptions{}, false
			}
			options.MaximumScore = value
			options.MaximumSet = true
		default:
			root, ok := parseSinglePathOption(options.Root, arg, "go crap", errOut)
			if !ok {
				return toolchecks.CRAPOptions{}, false
			}
			options.Root = root
		}
	}
	return options, true
}

func parseGoMutationArgs(args []string, errOut io.Writer) (toolchecks.MutationOptions, bool) {
	options := toolchecks.MutationOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--target":
			if i+1 >= len(args) {
				_, _ = fmt.Fprintln(errOut, "--target requires a value")
				return toolchecks.MutationOptions{}, false
			}
			i++
			if args[i] == "" || args[i][0] == '-' {
				_, _ = fmt.Fprintln(errOut, "--target requires a file value")
				return toolchecks.MutationOptions{}, false
			}
			options.Target = args[i]
		case "--scan":
			options.Scan = true
		default:
			root, ok := parseSinglePathOption(options.Root, arg, "go mutate", errOut)
			if !ok {
				return toolchecks.MutationOptions{}, false
			}
			options.Root = root
		}
	}
	if options.Target == "" {
		_, _ = fmt.Fprintln(errOut, "--target cannot be empty")
		return toolchecks.MutationOptions{}, false
	}
	return options, true
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
	_, _ = fmt.Fprintln(out, "  slophammer check <path> [--format text|json]")
	_, _ = fmt.Fprintln(out, "  slophammer explain <rule-id>")
	_, _ = fmt.Fprintln(out, "  slophammer go dry [path] [--max-candidates n] [--show-report]")
	_, _ = fmt.Fprintln(out, "  slophammer go crap [path] [--max-score n]")
	_, _ = fmt.Fprintln(out, "  slophammer go mutate [path] --target file [--scan]")
}

func printGoUsage(out io.Writer) {
	_, _ = fmt.Fprintln(out, "usage:")
	_, _ = fmt.Fprintln(out, "  slophammer go dry [path] [--max-candidates n] [--show-report]")
	_, _ = fmt.Fprintln(out, "  slophammer go crap [path] [--max-score n]")
	_, _ = fmt.Fprintln(out, "  slophammer go mutate [path] --target file [--scan]")
}
