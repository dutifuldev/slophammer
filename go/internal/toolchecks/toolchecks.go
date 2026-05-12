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
)

const (
	DefaultMaximumDRYCandidates = 40
	DefaultMaximumCRAPScore     = 30
	DefaultMutationTarget       = "internal/rules/rules.go"
)

type Runner interface {
	Run(ctx context.Context, dir string, name string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	err := cmd.Run()
	return combined.Bytes(), err
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

	output, err := runner.Run(ctx, root, "go", "run", "github.com/unclebob/dry4go/cmd/dry4go@latest", "--format", "json", ".")
	if options.ShowReport || err != nil {
		writeBytes(out, output)
	}
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "dry4go failed: %v\n", err)
		return 2
	}

	candidateCount, err := CountDRYCandidates(output)
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

	output, err := runner.Run(ctx, root, "go", "run", "github.com/unclebob/crap4go/cmd/crap4go@latest")
	writeBytes(out, output)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "crap4go failed: %v\n", err)
		return 2
	}

	violations, err := CRAPViolations(output, maximumScore)
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
		target = DefaultMutationTarget
	}
	args := []string{"run", "github.com/unclebob/mutate4go/cmd/mutate4go@latest", target}
	if options.Scan {
		args = append(args, "--scan")
	}

	output, err := runner.Run(ctx, root, "go", args...)
	writeBytes(out, output)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "mutate4go failed: %v\n", err)
		return 2
	}
	return 0
}

func CountDRYCandidates(report []byte) (int, error) {
	var parsed struct {
		Candidates []json.RawMessage `json:"candidates"`
	}
	if err := json.Unmarshal(report, &parsed); err != nil {
		return 0, err
	}
	if parsed.Candidates == nil {
		return 0, errors.New("missing candidates field")
	}
	return len(parsed.Candidates), nil
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
