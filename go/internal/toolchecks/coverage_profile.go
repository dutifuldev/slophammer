package toolchecks

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

func targetedCoverageInputs(ctx context.Context, root string, options CoverageOptions, errOut io.Writer, runner Runner) (targetedCoverageConfig, bool) {
	modulePath, ok := goListModulePath(ctx, root, errOut, runner)
	if !ok {
		return targetedCoverageConfig{}, false
	}
	if options.CoverageProfile != "" {
		return targetedCoverageConfig{modulePath: modulePath}, true
	}
	coverPackages, ok := goListPackages(ctx, root, coveragePackagePatterns(options), errOut, runner)
	if !ok {
		return targetedCoverageConfig{}, false
	}
	testPackages, ok := goListPackages(ctx, root, []string{"./..."}, errOut, runner)
	if !ok {
		return targetedCoverageConfig{}, false
	}
	return targetedCoverageConfig{modulePath: modulePath, coverPackages: coverPackages, testPackages: testPackages}, true
}

func targetedCoverageProfile(
	ctx context.Context,
	root string,
	modulePath string,
	profile string,
	targets []string,
	coverPackages []string,
	testPackages []string,
	out io.Writer,
	errOut io.Writer,
	runner Runner,
) (string, []byte, map[string]bool, func(), bool) {
	if strings.TrimSpace(profile) != "" {
		profilePath, ok := suppliedCoverageProfilePath(root, profile, errOut)
		if !ok {
			return "", nil, nil, func() {}, false
		}
		coverOutput, scopedFiles, ok := suppliedCoverageProfileOutput(ctx, root, modulePath, profilePath, targets, errOut, runner)
		return profilePath, coverOutput, scopedFiles, func() {}, ok
	}

	profilePath, cleanup, ok := runTargetedCoverage(ctx, root, coverPackages, testPackages, out, errOut, runner)
	if !ok {
		return "", nil, nil, cleanup, false
	}
	coverOutput, ok := goToolCoverFunc(ctx, root, profilePath, errOut, runner)
	return profilePath, coverOutput, nil, cleanup, ok
}

func suppliedCoverageProfilePath(root string, profile string, errOut io.Writer) (string, bool) {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		_, _ = fmt.Fprintln(errOut, "coverage profile path cannot be empty")
		return "", false
	}
	profilePath, ok := resolveSuppliedCoverageProfilePath(root, profile, errOut)
	if !ok {
		return "", false
	}
	return validateSuppliedCoverageProfileFile(profilePath, errOut)
}

func resolveSuppliedCoverageProfilePath(root string, profile string, errOut io.Writer) (string, bool) {
	if !filepath.IsAbs(profile) {
		absolute, err := filepath.Abs(filepath.Join(root, profile))
		if err != nil {
			_, _ = fmt.Fprintf(errOut, "coverage profile path resolution failed: %v\n", err)
			return "", false
		}
		profile = absolute
	}
	return filepath.Clean(profile), true
}

func validateSuppliedCoverageProfileFile(profilePath string, errOut io.Writer) (string, bool) {
	info, err := os.Stat(profilePath)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "coverage profile %s is not readable: %v\n", profilePath, err)
		return "", false
	}
	if info.IsDir() {
		_, _ = fmt.Fprintf(errOut, "coverage profile %s is a directory\n", profilePath)
		return "", false
	}
	if info.Size() == 0 {
		_, _ = fmt.Fprintf(errOut, "coverage profile %s is empty\n", profilePath)
		return "", false
	}
	return profilePath, true
}

func suppliedCoverageProfileOutput(
	ctx context.Context,
	root string,
	modulePath string,
	profilePath string,
	targets []string,
	errOut io.Writer,
	runner Runner,
) ([]byte, map[string]bool, bool) {
	coverOutput, ok := goToolCoverFunc(ctx, root, profilePath, errOut, runner)
	if !ok {
		return nil, nil, false
	}
	scopedFiles, ok := validateCoverageProfileScope(ctx, root, profilePath, modulePath, targets, errOut, runner)
	if !ok {
		return nil, nil, false
	}
	return coverOutput, scopedFiles, true
}

func runTargetedCoverage(
	ctx context.Context,
	root string,
	coverPackages []string,
	testPackages []string,
	out io.Writer,
	errOut io.Writer,
	runner Runner,
) (string, func(), bool) {
	profileDir, err := os.MkdirTemp("", "slophammer-go-coverage-*")
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "coverage profile setup failed: %v\n", err)
		return "", func() {}, false
	}
	cleanup := func() {
		_ = os.RemoveAll(profileDir)
	}
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
		cleanup()
		return "", func() {}, false
	}
	return profilePath, cleanup, true
}

func goToolCoverFunc(ctx context.Context, root string, profilePath string, errOut io.Writer, runner Runner) ([]byte, bool) {
	result, err := runner.Run(ctx, root, "go", "tool", "cover", "-func="+profilePath)
	writeBytes(errOut, result.Stderr)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "go tool cover failed: %v\n", err)
		return nil, false
	}
	if len(bytes.TrimSpace(result.Stdout)) == 0 {
		_, _ = fmt.Fprintln(errOut, "go tool cover returned no output")
		return nil, false
	}
	return result.Stdout, true
}

func coverFunctionCoverageFromOutput(modulePath string, output []byte) (map[string]float64, bool) {
	coverage := map[string]float64{}
	for _, line := range strings.Split(string(output), "\n") {
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
	if len(coverage) == 0 {
		return nil, false
	}
	return coverage, true
}

func addRawFunctionLineCoverage(root string, profilePath string, modulePath string, scopedFiles map[string]bool, coverage map[string]float64, complexity map[string]functionComplexity, errOut io.Writer) bool {
	if !hasMissingFunctionCoverage(coverage, complexity) {
		return true
	}
	// #nosec G304 -- profilePath is an explicit coverage artifact path validated before use.
	content, err := os.ReadFile(profilePath)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "coverage profile %s is not readable: %v\n", profilePath, err)
		return false
	}
	scopedBlocks := scopedCoverageBlocks(content, modulePath, scopedFiles)
	for key, item := range complexity {
		if _, ok := coverage[key]; ok {
			continue
		}
		blocks := rawCoverageBlocksForFunction(root, key, item, scopedBlocks)
		if value, ok := coveragePercentFromBlocks(blocks); ok {
			coverage[key] = value
		}
	}
	return true
}

func hasMissingFunctionCoverage(coverage map[string]float64, complexity map[string]functionComplexity) bool {
	for key := range complexity {
		if _, ok := coverage[key]; !ok {
			return true
		}
	}
	return false
}

func rawCoverageBlocksForFunction(root string, key string, item functionComplexity, blocks map[string]coverageProfileBlock) []coverageProfileBlock {
	filePath, line, ok := splitFunctionLineKey(key)
	if !ok {
		return nil
	}
	literalRange, ok := functionLiteralRange(root, filePath, line, item.Column)
	if !ok {
		return nil
	}
	matched := make([]coverageProfileBlock, 0)
	for _, block := range blocks {
		if block.filePath != filePath {
			continue
		}
		if literalRange.contains(block.sourceRange()) {
			matched = append(matched, block)
		}
	}
	return matched
}

func splitFunctionLineKey(key string) (string, int, bool) {
	parts := strings.Split(key, ":")
	if len(parts) < 2 {
		return "", 0, false
	}
	filePath := parts[0]
	lineText := parts[1]
	if filePath == "" || lineText == "" {
		return "", 0, false
	}
	line, err := strconv.Atoi(lineText)
	if err != nil || line <= 0 {
		return "", 0, false
	}
	return filePath, line, true
}

func functionLiteralRange(root string, filePath string, line int, column int) (sourceRange, bool) {
	fset := token.NewFileSet()
	sourcePath := filepath.Join(root, filepath.FromSlash(filePath))
	file, err := parser.ParseFile(fset, sourcePath, nil, 0)
	if err != nil {
		return sourceRange{}, false
	}
	var matched sourceRange
	ast.Inspect(file, func(node ast.Node) bool {
		if matched.start.line != 0 {
			return false
		}
		literal, ok := node.(*ast.FuncLit)
		if !ok {
			return true
		}
		start := fset.Position(literal.Type.Func)
		if start.Line != line {
			return true
		}
		if column > 0 && start.Column != column {
			return true
		}
		end := fset.Position(literal.End())
		matched = sourceRange{
			start: sourcePosition{line: start.Line, column: start.Column},
			end:   sourcePosition{line: end.Line, column: end.Column},
		}
		return false
	})
	return matched, matched.start.line != 0 && matched.end.line != 0
}

func coveragePercentFromBlocks(blocks []coverageProfileBlock) (float64, bool) {
	if len(blocks) == 0 {
		return 0, false
	}
	var coveredStatements int64
	var totalStatements int64
	for _, block := range blocks {
		totalStatements += block.statements
		if block.count > 0 {
			coveredStatements += block.statements
		}
	}
	if totalStatements == 0 {
		return 0, true
	}
	return 100 * float64(coveredStatements) / float64(totalStatements), true
}

func coverTotalCoverageFromOutput(output []byte) (float64, bool) {
	for _, line := range strings.Split(string(output), "\n") {
		if value, ok := parseCoverageTotalLine(line); ok {
			return value, true
		}
	}
	return 0, false
}

func coverageTotal(profilePath string, modulePath string, suppliedProfile string, scopedFiles map[string]bool, coverOutput []byte, errOut io.Writer) (float64, bool) {
	if strings.TrimSpace(suppliedProfile) == "" {
		return coverTotalCoverageFromOutput(coverOutput)
	}
	return suppliedCoverageProfileTotal(profilePath, modulePath, scopedFiles, errOut)
}

func suppliedCoverageProfileTotal(profilePath string, modulePath string, scopedFiles map[string]bool, errOut io.Writer) (float64, bool) {
	// #nosec G304 -- profilePath is an explicit coverage artifact path validated before use.
	content, err := os.ReadFile(profilePath)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "coverage profile %s is not readable: %v\n", profilePath, err)
		return 0, false
	}
	blocks := scopedCoverageBlocks(content, modulePath, scopedFiles)
	return coverageBlockTotal(profilePath, blocks, errOut)
}

func scopedCoverageBlocks(content []byte, modulePath string, scopedFiles map[string]bool) map[string]coverageProfileBlock {
	blocks := map[string]coverageProfileBlock{}
	for _, line := range strings.Split(string(content), "\n") {
		block, ok := parseCoverageProfileBlock(line, modulePath)
		if !ok || !profileFileIsInScope(block.filePath, scopedFiles) {
			continue
		}
		existing := blocks[block.key]
		if existing.key == "" || block.count > existing.count {
			blocks[block.key] = block
		}
	}
	return blocks
}

func coverageBlockTotal(profilePath string, blocks map[string]coverageProfileBlock, errOut io.Writer) (float64, bool) {
	var coveredStatements int64
	var totalStatements int64
	for _, block := range blocks {
		totalStatements += block.statements
		if block.count > 0 {
			coveredStatements += block.statements
		}
	}
	if totalStatements == 0 {
		_, _ = fmt.Fprintf(errOut, "coverage profile %s did not include scoped statements\n", profilePath)
		return 0, false
	}
	return 100 * float64(coveredStatements) / float64(totalStatements), true
}

type coverageProfileBlock struct {
	key         string
	filePath    string
	startLine   int
	startColumn int
	endLine     int
	endColumn   int
	statements  int64
	count       int64
}

func (block coverageProfileBlock) sourceRange() sourceRange {
	return sourceRange{
		start: sourcePosition{line: block.startLine, column: block.startColumn},
		end:   sourcePosition{line: block.endLine, column: block.endColumn},
	}
}

func parseCoverageProfileBlock(line string, modulePath string) (coverageProfileBlock, bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return coverageProfileBlock{}, false
	}
	metadata, ok := parseCoverageProfileBlockMetadata(fields[0], modulePath)
	if !ok {
		return coverageProfileBlock{}, false
	}
	statements, count, ok := parseCoverageProfileBlockCounts(fields)
	if !ok {
		return coverageProfileBlock{}, false
	}
	return coverageProfileBlock{
		key:         metadata.key,
		filePath:    metadata.filePath,
		startLine:   metadata.rangeValue.start.line,
		startColumn: metadata.rangeValue.start.column,
		endLine:     metadata.rangeValue.end.line,
		endColumn:   metadata.rangeValue.end.column,
		statements:  statements,
		count:       count,
	}, true
}

type coverageProfileBlockMetadata struct {
	key        string
	filePath   string
	rangeValue sourceRange
}

func parseCoverageProfileBlockMetadata(location string, modulePath string) (coverageProfileBlockMetadata, bool) {
	filePath, ok := coverFilePath(location, modulePath)
	if !ok {
		return coverageProfileBlockMetadata{}, false
	}
	key, ok := coverProfileBlockKey(location, modulePath)
	if !ok {
		return coverageProfileBlockMetadata{}, false
	}
	rangeValue, ok := coverProfileBlockRange(location, modulePath)
	if !ok {
		return coverageProfileBlockMetadata{}, false
	}
	return coverageProfileBlockMetadata{key: key, filePath: filePath, rangeValue: rangeValue}, true
}

func parseCoverageProfileBlockCounts(fields []string) (int64, int64, bool) {
	statements, err := strconv.ParseInt(fields[len(fields)-2], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	count, err := strconv.ParseInt(fields[len(fields)-1], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return statements, count, true
}

func coverProfileBlockKey(location string, modulePath string) (string, bool) {
	if !strings.HasPrefix(location, modulePath+"/") {
		return "", false
	}
	key := strings.TrimPrefix(location, modulePath+"/")
	if key == "" {
		return "", false
	}
	return strings.ReplaceAll(key, "\\", "/"), true
}

func coverProfileBlockRange(location string, modulePath string) (sourceRange, bool) {
	position, ok := coverProfileBlockPosition(location, modulePath)
	if !ok {
		return sourceRange{}, false
	}
	return parseCoverageBlockRange(position)
}

func coverProfileBlockPosition(location string, modulePath string) (string, bool) {
	key, ok := coverProfileBlockKey(location, modulePath)
	if !ok {
		return "", false
	}
	_, position, ok := strings.Cut(key, ":")
	if !ok || position == "" {
		return "", false
	}
	return position, true
}

func parseCoverageBlockRange(position string) (sourceRange, bool) {
	startPosition, endPosition, ok := strings.Cut(position, ",")
	if !ok {
		return sourceRange{}, false
	}
	start, ok := parseCoveragePosition(startPosition)
	if !ok {
		return sourceRange{}, false
	}
	end, ok := parseCoveragePosition(endPosition)
	if !ok {
		return sourceRange{}, false
	}
	return sourceRange{start: start, end: end}, true
}

type sourceRange struct {
	start sourcePosition
	end   sourcePosition
}

func (item sourceRange) contains(other sourceRange) bool {
	return !sourcePositionLess(other.start, item.start) && !sourcePositionLess(item.end, other.end)
}

type sourcePosition struct {
	line   int
	column int
}

func sourcePositionLess(left sourcePosition, right sourcePosition) bool {
	if left.line != right.line {
		return left.line < right.line
	}
	return left.column < right.column
}

func parseCoveragePosition(position string) (sourcePosition, bool) {
	lineText, columnText, ok := strings.Cut(position, ".")
	if !ok || lineText == "" || columnText == "" {
		return sourcePosition{}, false
	}
	line, err := strconv.Atoi(lineText)
	if err != nil || line <= 0 {
		return sourcePosition{}, false
	}
	column, err := strconv.Atoi(columnText)
	if err != nil || column <= 0 {
		return sourcePosition{}, false
	}
	return sourcePosition{line: line, column: column}, true
}

func profileFileIsInScope(filePath string, allowed map[string]bool) bool {
	return allowed == nil || allowed[filePath]
}

func validateCoverageProfileScope(ctx context.Context, root string, profilePath string, modulePath string, targets []string, errOut io.Writer, runner Runner) (map[string]bool, bool) {
	files, ok := coveredModuleFilesFromProfile(profilePath, modulePath, errOut)
	if !ok {
		return nil, false
	}
	if len(files) == 0 {
		_, _ = fmt.Fprintf(errOut, "coverage profile %s does not include files for module %s\n", profilePath, modulePath)
		return nil, false
	}
	required, ok := requiredCoverageScopeFiles(ctx, root, targets, errOut, runner)
	if !ok {
		return nil, false
	}
	if !validateCoverageProfileTargets(profilePath, files, required, errOut) {
		return nil, false
	}
	return required, true
}

func requiredCoverageScopeFiles(ctx context.Context, root string, targets []string, errOut io.Writer, runner Runner) (map[string]bool, bool) {
	patterns := []string{"./..."}
	allowed := map[string]bool{}
	if len(targets) > 0 {
		patterns = packageDirs(targets)
		allowed = targetFileSet(targets)
	}
	files, ok := goListPackageFiles(ctx, root, patterns, errOut, runner)
	if !ok {
		return nil, false
	}
	required := map[string]bool{}
	for filePath := range files {
		if len(allowed) > 0 && !allowed[filePath] {
			continue
		}
		if goFileMayHaveCoverageBlocks(root, filePath) {
			required[filePath] = true
		}
	}
	return required, true
}

func validateCoverageProfileTargets(profilePath string, files map[string]bool, required map[string]bool, errOut io.Writer) bool {
	for filePath := range required {
		if !files[filePath] {
			_, _ = fmt.Fprintf(errOut, "coverage profile %s does not include configured Go scope file %s\n", profilePath, filePath)
			return false
		}
	}
	return true
}

func goListPackageFiles(ctx context.Context, root string, patterns []string, errOut io.Writer, runner Runner) (map[string]bool, bool) {
	args := []string{"list", "-f", "{{.Dir}}|{{range .GoFiles}}{{.}} {{end}}"}
	args = append(args, patterns...)
	result, err := runner.Run(ctx, root, "go", args...)
	writeBytes(errOut, result.Stderr)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "go list package files failed: %v\n", err)
		return nil, false
	}
	files, ok := parseGoListPackageFiles(root, result.Stdout)
	if !ok {
		_, _ = fmt.Fprintf(errOut, "go list returned no package files for %s\n", strings.Join(patterns, " "))
	}
	return files, ok
}

func parseGoListPackageFiles(root string, output []byte) (map[string]bool, bool) {
	files := map[string]bool{}
	for _, line := range strings.Split(string(output), "\n") {
		dir, namesText, ok := strings.Cut(strings.TrimSpace(line), "|")
		if !ok {
			continue
		}
		for _, name := range strings.Fields(namesText) {
			filePath, ok := packageFilePath(root, dir, name)
			if ok {
				files[filePath] = true
			}
		}
	}
	return files, len(files) > 0
}

func packageFilePath(root string, dir string, name string) (string, bool) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	absolutePath := filepath.Join(dir, name)
	relPath, err := filepath.Rel(absoluteRoot, absolutePath)
	relPath = filepath.ToSlash(relPath)
	if err != nil || relPath == ".." || strings.HasPrefix(relPath, "../") {
		return "", false
	}
	return cleanGoTargetPath(relPath), true
}

func goFileMayHaveCoverageBlocks(root string, filePath string) bool {
	if strings.HasSuffix(filePath, "_test.go") {
		return false
	}
	sourcePath := filepath.Join(root, filepath.FromSlash(filePath))
	file, err := parser.ParseFile(token.NewFileSet(), sourcePath, nil, parser.SkipObjectResolution)
	if err != nil {
		return true
	}
	hasCoverableSyntax := false
	ast.Inspect(file, func(node ast.Node) bool {
		switch item := node.(type) {
		case *ast.FuncDecl:
			if item.Body != nil {
				hasCoverableSyntax = true
			}
		case *ast.FuncLit:
			if item.Body != nil {
				hasCoverableSyntax = true
			}
		}
		return !hasCoverableSyntax
	})
	return hasCoverableSyntax
}

func coveredModuleFilesFromProfile(profilePath string, modulePath string, errOut io.Writer) (map[string]bool, bool) {
	// #nosec G304 -- profilePath is an explicit coverage artifact path validated before use.
	content, err := os.ReadFile(profilePath)
	if err != nil {
		_, _ = fmt.Fprintf(errOut, "coverage profile %s is not readable: %v\n", profilePath, err)
		return nil, false
	}
	files := map[string]bool{}
	for _, line := range strings.Split(string(content), "\n") {
		block, ok := parseCoverageProfileBlock(line, modulePath)
		if ok {
			files[block.filePath] = true
		}
	}
	return files, true
}

func targetFileSet(targets []string) map[string]bool {
	set := map[string]bool{}
	for _, target := range targets {
		cleaned := cleanGoTargetPath(target)
		if cleaned != "" {
			set[cleaned] = true
		}
	}
	return set
}

func cleanGoTargetPath(target string) string {
	target = strings.TrimSpace(strings.ReplaceAll(target, "\\", "/"))
	if target == "" {
		return ""
	}
	return strings.TrimPrefix(path.Clean(target), "./")
}

func parseCoverageTotalLine(line string) (float64, bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 || fields[0] != "total:" || !strings.HasSuffix(fields[len(fields)-1], "%") {
		return 0, false
	}
	value, err := strconv.ParseFloat(strings.TrimSuffix(fields[len(fields)-1], "%"), 64)
	if err != nil {
		return 0, false
	}
	return value, true
}
