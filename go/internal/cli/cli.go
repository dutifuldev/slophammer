package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/dutifuldev/slophammer/go/internal/app"
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
	options, ok := parseCheckArgs(args, errOut)
	if !ok {
		return app.ExitError
	}
	return app.Check(ctx, options, out, errOut)
}

func runExplain(args []string, out io.Writer, errOut io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(errOut, "usage: slophammer explain <rule-id>")
		return app.ExitError
	}
	return app.Explain(args[0], out, errOut)
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

func printUsage(out io.Writer) {
	_, _ = fmt.Fprintln(out, "usage:")
	_, _ = fmt.Fprintln(out, "  slophammer check <path> [--format text|json]")
	_, _ = fmt.Fprintln(out, "  slophammer explain <rule-id>")
}
