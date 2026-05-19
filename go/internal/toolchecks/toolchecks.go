package toolchecks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
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
	Targets      []string
	Exclude      []string
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
	targets := crapTargets(options)
	if len(targets) > 0 {
		return checkTargetedCRAP(ctx, root, targets, maximumScore, out, errOut, runner)
	}

	result, err := runner.Run(ctx, root, "go", gotools.CRAP4Go.GoRunArgs(gotools.Latest, crapTargets(options)...)...)
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

func checkTargetedCRAP(
	ctx context.Context,
	root string,
	targets []string,
	maximumScore float64,
	out io.Writer,
	errOut io.Writer,
	runner Runner,
) int {
	modulePath, ok := goListModulePath(ctx, root, errOut, runner)
	if !ok {
		return 2
	}
	coverPackages, ok := goListPackages(ctx, root, packageDirs(targets), errOut, runner)
	if !ok {
		return 2
	}
	testPackages, ok := goListPackages(ctx, root, []string{"./..."}, errOut, runner)
	if !ok {
		return 2
	}
	profileDir, err := os.MkdirTemp("", "slophammer-crap-*")
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "coverage profile setup failed: %v\n", err)
		return 2
	}
	defer os.RemoveAll(profileDir)
	profilePath := filepath.Join(profileDir, "coverage.out")
	args := []string{
		"test",
		"-count=1",
		"-covermode=count",
		"-coverpkg=" + strings.Join(coverPackages, ","),
		"-coverprofile=" + profilePath,
	}
	args = append(args, testPackages...)
	result, err := runner.Run(ctx, root, "go", args...)
	if err != nil {
		writeBytes(out, result.Stdout)
		writeBytes(errOut, result.Stderr)
		_, _ = fmt.Fprintf(errOut, "coverage test failed: %v\n", err)
		return 2
	}

	complexity, ok := gocycloComplexity(ctx, root, targets, errOut, runner)
	if !ok {
		return 2
	}
	coverage, ok := coverFunctionCoverage(ctx, root, modulePath, profilePath, errOut, runner)
	if !ok {
		return 2
	}

	violations := targetedCRAPViolations(complexity, coverage, maximumScore)
	for _, violation := range violations {
		_, _ = fmt.Fprintf(errOut, "CRAP score %.1f exceeds maximum %.1f for %s\n", violation.Score, maximumScore, violation.Name)
	}
	if len(violations) > 0 {
		return 1
	}
	_, _ = fmt.Fprintf(out, "CRAP scores meet maximum %.1f\n", maximumScore)
	return 0
}

func goListModulePath(ctx context.Context, root string, errOut io.Writer, runner Runner) (string, bool) {
	result, err := runner.Run(ctx, root, "go", "list", "-m")
	writeBytes(errOut, result.Stderr)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "go list -m failed: %v\n", err)
		return "", false
	}
	modulePath := strings.TrimSpace(string(result.Stdout))
	if modulePath == "" {
		_, _ = fmt.Fprintln(errOut, "go list -m returned an empty module path")
		return "", false
	}
	return modulePath, true
}

func goListPackages(ctx context.Context, root string, patterns []string, errOut io.Writer, runner Runner) ([]string, bool) {
	args := append([]string{"list"}, patterns...)
	result, err := runner.Run(ctx, root, "go", args...)
	writeBytes(errOut, result.Stderr)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "go list failed: %v\n", err)
		return nil, false
	}
	packages := strings.Fields(string(result.Stdout))
	if len(packages) == 0 {
		_, _ = fmt.Fprintf(errOut, "go list returned no packages for %s\n", strings.Join(patterns, " "))
		return nil, false
	}
	return packages, true
}

func packageDirs(targets []string) []string {
	seen := map[string]bool{}
	dirs := make([]string, 0, len(targets))
	for _, target := range targets {
		dir := path.Dir(strings.ReplaceAll(target, "\\", "/"))
		if dir == "." {
			dir = "."
		}
		pattern := "./" + strings.TrimPrefix(dir, "./")
		if pattern == "./." {
			pattern = "."
		}
		if !seen[pattern] {
			seen[pattern] = true
			dirs = append(dirs, pattern)
		}
	}
	sort.Strings(dirs)
	return dirs
}

type functionComplexity struct {
	Name       string
	Package    string
	Complexity float64
}

func gocycloComplexity(ctx context.Context, root string, targets []string, errOut io.Writer, runner Runner) (map[string]functionComplexity, bool) {
	args := gotools.Gocyclo.GoRunArgs(gotools.Latest, append([]string{"-over", "0"}, targets...)...)
	result, err := runner.Run(ctx, root, "go", args...)
	writeBytes(errOut, result.Stderr)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "gocyclo failed: %v\n", err)
		return nil, false
	}
	complexity := map[string]functionComplexity{}
	for _, line := range strings.Split(string(result.Stdout), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		value, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			continue
		}
		key, ok := fileLineKey(fields[3])
		if !ok {
			continue
		}
		complexity[key] = functionComplexity{
			Name:       fields[2],
			Package:    fields[1],
			Complexity: value,
		}
	}
	return complexity, true
}

func coverFunctionCoverage(ctx context.Context, root string, modulePath string, profilePath string, errOut io.Writer, runner Runner) (map[string]float64, bool) {
	result, err := runner.Run(ctx, root, "go", "tool", "cover", "-func="+profilePath)
	writeBytes(errOut, result.Stderr)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "go tool cover failed: %v\n", err)
		return nil, false
	}
	coverage := map[string]float64{}
	for _, line := range strings.Split(string(result.Stdout), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || !strings.HasSuffix(fields[len(fields)-1], "%") {
			continue
		}
		key, ok := coverFileLineKey(fields[0], modulePath)
		if !ok {
			continue
		}
		valueText := strings.TrimSuffix(fields[len(fields)-1], "%")
		value, err := strconv.ParseFloat(valueText, 64)
		if err != nil {
			continue
		}
		coverage[key] = value
	}
	return coverage, true
}

func targetedCRAPViolations(complexity map[string]functionComplexity, coverage map[string]float64, maximumScore float64) []CRAPViolation {
	violations := make([]CRAPViolation, 0)
	for key, item := range complexity {
		covered, ok := coverage[key]
		if !ok {
			continue
		}
		uncovered := 1 - covered/100
		score := item.Complexity*item.Complexity*uncovered*uncovered*uncovered + item.Complexity
		rounded := roundTenths(score)
		if rounded > maximumScore {
			violations = append(violations, CRAPViolation{
				Name:  item.Package + "." + item.Name,
				Score: rounded,
			})
		}
	}
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].Score == violations[j].Score {
			return violations[i].Name < violations[j].Name
		}
		return violations[i].Score > violations[j].Score
	})
	return violations
}

func roundTenths(value float64) float64 {
	rounded, _ := strconv.ParseFloat(fmt.Sprintf("%.1f", value), 64)
	return rounded
}

func coverFileLineKey(location string, modulePath string) (string, bool) {
	if !strings.HasPrefix(location, modulePath+"/") {
		return "", false
	}
	return fileLineKey(strings.TrimPrefix(location, modulePath+"/"))
}

func fileLineKey(location string) (string, bool) {
	parts := strings.Split(location, ":")
	if len(parts) < 2 {
		return "", false
	}
	filePath := strings.ReplaceAll(parts[0], "\\", "/")
	line := parts[1]
	if filePath == "" || line == "" {
		return "", false
	}
	return filePath + ":" + line, true
}

func crapTargets(options CRAPOptions) []string {
	targets := make([]string, 0, len(options.Targets))
	for _, target := range options.Targets {
		if strings.TrimSpace(target) != "" {
			targets = append(targets, target)
		}
	}
	return targets
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
