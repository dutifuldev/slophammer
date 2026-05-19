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

	"github.com/dutifuldev/slophammer/go/internal/dry"
	"github.com/dutifuldev/slophammer/go/internal/gotools"
)

const (
	DefaultMaximumDRYCandidates = 0
	DefaultMaximumCRAPScore     = 8
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
	Root                string
	MaximumCandidates   int
	MaximumSet          bool
	ShowReport          bool
	Format              string
	Paths               []string
	Exclude             []string
	StructuralEnabled   bool
	StructuralSet       bool
	StructuralThreshold float64
	StructuralMinLines  int
	StructuralMinNodes  int
	CopiedBlockEnabled  bool
	CopiedBlockSet      bool
	CopiedBlockTokens   int
}

func (options DryOptions) RootPath() string {
	return options.Root
}

type CRAPOptions struct {
	Root         string
	MaximumScore float64
	MaximumSet   bool
}

func (options CRAPOptions) RootPath() string {
	return options.Root
}

type MutationOptions struct {
	Root    string
	Target  string
	Targets []string
	Exclude []string
	Scan    bool
}

func (options MutationOptions) RootPath() string {
	return options.Root
}

func CheckDry(ctx context.Context, options DryOptions, out io.Writer, errOut io.Writer, runner Runner) int {
	_ = ctx
	_ = runner
	root := defaultRoot(options.Root)
	maximumCandidates := dryCandidateLimit(options)

	report, err := dry.Find(dry.Options{
		Root:                root,
		Paths:               dryPaths(options),
		StructuralEnabled:   dryStructuralEnabled(options),
		StructuralSet:       options.StructuralSet,
		StructuralThreshold: options.StructuralThreshold,
		StructuralMinLines:  options.StructuralMinLines,
		StructuralMinNodes:  options.StructuralMinNodes,
		CopiedBlockEnabled:  dryCopiedBlockEnabled(options),
		CopiedBlockSet:      options.CopiedBlockSet,
		CopiedBlockTokens:   options.CopiedBlockTokens,
	})
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "dry check failed: %v\n", err)
		return 2
	}

	if options.Format == "json" || options.ShowReport {
		content, err := dry.WriteJSON(report)
		if err != nil {
			_, _ = fmt.Fprintf(errOut, "dry report render failed: %v\n", err)
			return 2
		}
		writeBytes(out, content)
	}
	if options.Format == "text" {
		_, _ = io.WriteString(out, dry.FormatText(report))
	}

	candidateCount := len(report.Findings)
	if options.Format != "json" {
		_, _ = fmt.Fprintf(out, "DRY candidates: %d; maximum: %d\n", candidateCount, maximumCandidates)
	}
	if candidateCount > maximumCandidates {
		return 1
	}
	return 0
}

func dryStructuralEnabled(options DryOptions) bool {
	return dryBoolDefault(options.StructuralSet, options.StructuralEnabled)
}

func dryCopiedBlockEnabled(options DryOptions) bool {
	return dryBoolDefault(options.CopiedBlockSet, options.CopiedBlockEnabled)
}

func dryBoolDefault(configured bool, value bool) bool {
	if configured {
		return value
	}
	return true
}

func dryCandidateLimit(options DryOptions) int {
	if !options.MaximumSet && options.MaximumCandidates == 0 {
		return DefaultMaximumDRYCandidates
	}
	return options.MaximumCandidates
}

func CheckCRAP(ctx context.Context, options CRAPOptions, out io.Writer, errOut io.Writer, runner Runner) int {
	root := defaultRoot(options.Root)
	maximumScore := crapScoreLimit(options)

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

func crapScoreLimit(options CRAPOptions) float64 {
	if options.MaximumSet || options.MaximumScore != 0 {
		return options.MaximumScore
	}
	return DefaultMaximumCRAPScore
}

func CheckMutation(ctx context.Context, options MutationOptions, out io.Writer, errOut io.Writer, runner Runner) int {
	root := defaultRoot(options.Root)
	targets := mutationTargets(options)
	if len(targets) == 0 {
		_, _ = fmt.Fprintln(errOut, "--target is required")
		return 2
	}
	for _, target := range targets {
		args := gotools.Mutate4Go.GoRunArgs(gotools.Latest, target)
		if options.Scan {
			args = append(args, "--scan")
		}

		result, err := runner.Run(ctx, root, "go", args...)
		writeBytes(out, result.Stdout)
		writeBytes(errOut, result.Stderr)
		if err != nil {
			_, _ = fmt.Fprintf(errOut, "mutate4go failed for %s: %v\n", target, err)
			return 2
		}
	}
	return 0
}

func CountDRYCandidates(report []byte) (int, error) {
	var parsed map[string][]json.RawMessage
	if err := json.Unmarshal(report, &parsed); err != nil {
		return 0, err
	}
	if findings, ok := parsed["findings"]; ok {
		return len(findings), nil
	}
	if candidates, ok := parsed["candidates"]; ok {
		return len(candidates), nil
	}
	return 0, errors.New("missing findings field")
}

func DryPaths(options DryOptions) []string {
	return dryPaths(options)
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

func dryPaths(options DryOptions) []string {
	paths := make([]string, 0, len(options.Paths))
	for _, targetPath := range options.Paths {
		if targetPath != "" {
			paths = append(paths, targetPath)
		}
	}
	if len(paths) == 0 {
		return []string{"."}
	}
	return paths
}

func mutationTargets(options MutationOptions) []string {
	if options.Target != "" {
		return []string{options.Target}
	}
	targets := make([]string, 0, len(options.Targets))
	for _, target := range options.Targets {
		if target != "" {
			targets = append(targets, target)
		}
	}
	return targets
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
