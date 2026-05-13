package toolchecks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/dutifuldev/slophammer/go/internal/gotools"
)

const (
	DefaultMaximumDRYCandidates = 40
	DefaultMaximumCRAPScore     = 30
)

type Runner interface {
	Run(ctx context.Context, dir string, name string, args ...string) (CommandResult, error)
}

type ExecRunner struct{}

type CommandResult struct {
	Stdout []byte
	Stderr []byte
}

func (ExecRunner) Run(ctx context.Context, dir string, name string, args ...string) (CommandResult, error) {
	// #nosec G204 -- callers provide tool commands intentionally through the runner boundary.
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return CommandResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, err
}

type DryOptions struct {
	Root              string
	MaximumCandidates int
	MaximumSet        bool
	ShowReport        bool
}

type CRAPOptions struct {
	Root         string
	MaximumScore float64
	MaximumSet   bool
}

type MutationOptions struct {
	Root   string
	Target string
	Scan   bool
}

func CheckDry(ctx context.Context, options DryOptions, out io.Writer, errOut io.Writer, runner Runner) int {
	root := defaultRoot(options.Root)
	maximumCandidates := options.MaximumCandidates
	if !options.MaximumSet && maximumCandidates == 0 {
		maximumCandidates = DefaultMaximumDRYCandidates
	}

	result, err := runner.Run(ctx, root, "go", gotools.Dry4Go.GoRunArgs(gotools.Latest, "--format", "json", ".")...)
	if options.ShowReport || err != nil {
		writeBytes(out, result.Stdout)
	}
	writeBytes(errOut, result.Stderr)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "dry4go failed: %v\n", err)
		return 2
	}

	candidateCount, err := CountDRYCandidates(result.Stdout)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "dry4go report parse failed: %v\n", err)
		return 2
	}
	_, _ = fmt.Fprintf(out, "DRY candidates: %d; maximum: %d\n", candidateCount, maximumCandidates)
	if candidateCount > maximumCandidates {
		return 1
	}
	return 0
}

func CheckCRAP(ctx context.Context, options CRAPOptions, out io.Writer, errOut io.Writer, runner Runner) int {
	root := defaultRoot(options.Root)
	maximumScore := options.MaximumScore
	if !options.MaximumSet && maximumScore == 0 {
		maximumScore = DefaultMaximumCRAPScore
	}

	result, err := runner.Run(ctx, root, "go", gotools.CRAP4Go.GoRunArgs(gotools.Latest)...)
	writeBytes(out, result.Stdout)
	writeBytes(errOut, result.Stderr)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "crap4go failed: %v\n", err)
		return 2
	}

	violations, err := CRAPViolations(result.Stdout, maximumScore)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "crap4go report parse failed: %v\n", err)
		return 2
	}
	for _, violation := range violations {
		_, _ = fmt.Fprintf(errOut, "CRAP score %.1f exceeds maximum %.1f for %s\n", violation.Score, maximumScore, violation.Name)
	}
	if len(violations) > 0 {
		return 1
	}
	return 0
}

func CheckMutation(ctx context.Context, options MutationOptions, out io.Writer, errOut io.Writer, runner Runner) int {
	root := defaultRoot(options.Root)
	target := options.Target
	if target == "" {
		_, _ = fmt.Fprintln(errOut, "--target is required")
		return 2
	}
	args := gotools.Mutate4Go.GoRunArgs(gotools.Latest, target)
	if options.Scan {
		args = append(args, "--scan")
	}

	result, err := runner.Run(ctx, root, "go", args...)
	writeBytes(out, result.Stdout)
	writeBytes(errOut, result.Stderr)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "mutate4go failed: %v\n", err)
		return 2
	}
	return 0
}

func CountDRYCandidates(report []byte) (int, error) {
	var parsed map[string][]json.RawMessage
	if err := json.Unmarshal(report, &parsed); err != nil {
		return 0, err
	}
	candidates, ok := parsed["candidates"]
	if !ok {
		return 0, errors.New("missing candidates field")
	}
	return len(candidates), nil
}

type CRAPViolation struct {
	Name  string
	Score float64
}

func CRAPViolations(report []byte, maximumScore float64) ([]CRAPViolation, error) {
	var violations []CRAPViolation
	for _, line := range strings.Split(string(report), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		score, err := strconv.ParseFloat(fields[len(fields)-1], 64)
		if err != nil {
			continue
		}
		if score > maximumScore {
			violations = append(violations, CRAPViolation{Name: fields[0], Score: score})
		}
	}
	return violations, nil
}

func defaultRoot(root string) string {
	if root == "" {
		return "."
	}
	return root
}

func writeBytes(out io.Writer, content []byte) {
	if len(content) == 0 {
		return
	}
	_, _ = out.Write(content)
	if content[len(content)-1] != '\n' {
		_, _ = fmt.Fprintln(out)
	}
}
