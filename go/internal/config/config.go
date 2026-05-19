package config

import (
	"fmt"
	"path"
	"strings"

	"github.com/dutifuldev/slophammer/go/internal/repo"
	"gopkg.in/yaml.v3"
)

const (
	DefaultFileName = "slophammer.yml"
	AltFileName     = "slophammer.yaml"

	MinimumGoCoverageThreshold = 85
	MaximumGoCRAPScore         = 8
)

type Config struct {
	SourceDir string                `yaml:"-"`
	Rules     map[string]RuleConfig `yaml:"rules"`
	Go        GoConfig              `yaml:"go"`
}

type RuleConfig struct {
	Severity  string  `yaml:"severity"`
	Disabled  bool    `yaml:"disabled"`
	Reason    string  `yaml:"reason"`
	Threshold float64 `yaml:"threshold"`
	Max       float64 `yaml:"max"`
}

type GoConfig struct {
	CoverageThreshold    float64              `yaml:"coverage_threshold"`
	CoverageProfile      string               `yaml:"coverage_profile"`
	Targets              []string             `yaml:"targets"`
	Exclude              []string             `yaml:"exclude"`
	DRYMaxCandidates     int                  `yaml:"dry_max_candidates"`
	DRYMaxCandidatesSet  bool                 `yaml:"-"`
	DRYPaths             []string             `yaml:"dry_paths"`
	DRYExclude           []string             `yaml:"dry_exclude"`
	DRY                  DryConfig            `yaml:"dry"`
	CRAPMaxScore         float64              `yaml:"crap_max_score"`
	Mutation             MutationConfig       `yaml:"mutation"`
	DependencyBoundaries []DependencyBoundary `yaml:"dependency_boundaries"`
}

type MutationConfig struct {
	Targets []string `yaml:"targets"`
	Exclude []string `yaml:"exclude"`
}

type DryConfig struct {
	MaxFindings    int                 `yaml:"max_findings"`
	MaxFindingsSet bool                `yaml:"-"`
	Paths          []string            `yaml:"paths"`
	Exclude        []string            `yaml:"exclude"`
	Structural     DryStructuralConfig `yaml:"structural"`
	CopiedBlocks   DryCopiedConfig     `yaml:"copied_blocks"`
}

type DryStructuralConfig struct {
	EnabledSet bool    `yaml:"-"`
	Enabled    bool    `yaml:"enabled"`
	Threshold  float64 `yaml:"threshold"`
	MinLines   int     `yaml:"min_lines"`
	MinNodes   int     `yaml:"min_nodes"`
}

type DryCopiedConfig struct {
	EnabledSet bool `yaml:"-"`
	Enabled    bool `yaml:"enabled"`
	MinTokens  int  `yaml:"min_tokens"`
}

type DependencyBoundary struct {
	From  string   `yaml:"from"`
	Allow []string `yaml:"allow"`
}

func (cfg *GoConfig) UnmarshalYAML(value *yaml.Node) error {
	type dryStructuralConfig struct {
		Enabled   *bool   `yaml:"enabled"`
		Threshold float64 `yaml:"threshold"`
		MinLines  int     `yaml:"min_lines"`
		MinNodes  int     `yaml:"min_nodes"`
	}
	type dryCopiedConfig struct {
		Enabled   *bool `yaml:"enabled"`
		MinTokens int   `yaml:"min_tokens"`
	}
	type dryConfig struct {
		MaxFindings  *int                `yaml:"max_findings"`
		Paths        []string            `yaml:"paths"`
		Exclude      []string            `yaml:"exclude"`
		Structural   dryStructuralConfig `yaml:"structural"`
		CopiedBlocks dryCopiedConfig     `yaml:"copied_blocks"`
	}
	type goConfig struct {
		CoverageThreshold    float64              `yaml:"coverage_threshold"`
		CoverageProfile      string               `yaml:"coverage_profile"`
		Targets              []string             `yaml:"targets"`
		Exclude              []string             `yaml:"exclude"`
		DRYMaxCandidates     *int                 `yaml:"dry_max_candidates"`
		DRYPaths             []string             `yaml:"dry_paths"`
		DRYExclude           []string             `yaml:"dry_exclude"`
		DRY                  dryConfig            `yaml:"dry"`
		CRAPMaxScore         float64              `yaml:"crap_max_score"`
		Mutation             MutationConfig       `yaml:"mutation"`
		DependencyBoundaries []DependencyBoundary `yaml:"dependency_boundaries"`
	}
	var parsed goConfig
	if err := value.Decode(&parsed); err != nil {
		return err
	}
	cfg.CoverageThreshold = parsed.CoverageThreshold
	cfg.CoverageProfile = parsed.CoverageProfile
	cfg.Targets = parsed.Targets
	cfg.Exclude = parsed.Exclude
	if parsed.DRYMaxCandidates != nil {
		cfg.DRYMaxCandidates = *parsed.DRYMaxCandidates
		cfg.DRYMaxCandidatesSet = true
	}
	cfg.DRYPaths = parsed.DRYPaths
	cfg.DRYExclude = parsed.DRYExclude
	if parsed.DRY.MaxFindings != nil {
		cfg.DRY.MaxFindings = *parsed.DRY.MaxFindings
		cfg.DRY.MaxFindingsSet = true
		cfg.DRYMaxCandidates = *parsed.DRY.MaxFindings
		cfg.DRYMaxCandidatesSet = true
	}
	if len(parsed.DRY.Paths) > 0 {
		cfg.DRY.Paths = parsed.DRY.Paths
		cfg.DRYPaths = parsed.DRY.Paths
	}
	if len(parsed.DRY.Exclude) > 0 {
		cfg.DRY.Exclude = parsed.DRY.Exclude
		cfg.DRYExclude = parsed.DRY.Exclude
	}
	cfg.DRY.Structural.Threshold = parsed.DRY.Structural.Threshold
	cfg.DRY.Structural.MinLines = parsed.DRY.Structural.MinLines
	cfg.DRY.Structural.MinNodes = parsed.DRY.Structural.MinNodes
	if parsed.DRY.Structural.Enabled != nil {
		cfg.DRY.Structural.Enabled = *parsed.DRY.Structural.Enabled
		cfg.DRY.Structural.EnabledSet = true
	}
	cfg.DRY.CopiedBlocks.MinTokens = parsed.DRY.CopiedBlocks.MinTokens
	if parsed.DRY.CopiedBlocks.Enabled != nil {
		cfg.DRY.CopiedBlocks.Enabled = *parsed.DRY.CopiedBlocks.Enabled
		cfg.DRY.CopiedBlocks.EnabledSet = true
	}
	cfg.CRAPMaxScore = parsed.CRAPMaxScore
	cfg.Mutation = parsed.Mutation
	cfg.DependencyBoundaries = parsed.DependencyBoundaries
	return nil
}

func Load(snapshot repo.Snapshot) (Config, error) {
	file, ok := configFile(snapshot)
	if !ok {
		return Config{}, nil
	}
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(file.Content), &root); err != nil {
		return Config{}, fmt.Errorf("%s: %w", file.Path, err)
	}
	if err := validateKnownKeys(&root); err != nil {
		return Config{}, fmt.Errorf("%s: %w", file.Path, err)
	}
	var cfg Config
	if err := root.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("%s: %w", file.Path, err)
	}
	cfg.SourceDir = configSourceDir(file.Path)
	if err := validate(cfg); err != nil {
		return Config{}, fmt.Errorf("%s: %w", file.Path, err)
	}
	return cfg, nil
}

func (cfg Config) GoMutationScope() ([]string, []string) {
	targets := cfg.Go.Targets
	exclude := cfg.Go.Exclude
	if len(cfg.Go.Mutation.Targets) > 0 {
		targets = cfg.Go.Mutation.Targets
		exclude = cfg.Go.Mutation.Exclude
	} else if len(cfg.Go.Mutation.Exclude) > 0 {
		exclude = cfg.Go.Mutation.Exclude
	}
	return scopedConfigPaths(cfg.SourceDir, targets), scopedConfigExcludePaths(cfg.SourceDir, exclude)
}

func (cfg Config) GoDRYScope() ([]string, []string) {
	paths := cfg.Go.DRYPaths
	exclude := cfg.Go.DRYExclude
	if len(paths) == 0 {
		paths = cfg.Go.Targets
	}
	if len(exclude) == 0 {
		exclude = cfg.Go.Exclude
	}
	return scopedConfigPaths(cfg.SourceDir, paths), scopedConfigExcludePaths(cfg.SourceDir, exclude)
}

func (cfg Config) GoScope() ([]string, []string) {
	return scopedConfigPaths(cfg.SourceDir, cfg.Go.Targets), scopedConfigExcludePaths(cfg.SourceDir, cfg.Go.Exclude)
}

func (cfg Config) GoCoverageProfile() string {
	return scopedConfigPath(cfg.SourceDir, cfg.Go.CoverageProfile)
}

func (cfg Config) RuleSeverity(ruleID string, fallback string) string {
	rule, ok := cfg.Rules[ruleID]
	if !ok || rule.Severity == "" {
		return fallback
	}
	return rule.Severity
}

func configFile(snapshot repo.Snapshot) (repo.File, bool) {
	for _, name := range []string{DefaultFileName, AltFileName} {
		for filePath, file := range snapshot.Files {
			if filePath == name {
				return file, true
			}
		}
	}
	for _, file := range snapshot.FilesNamedFold(DefaultFileName, AltFileName) {
		return file, true
	}
	return repo.File{}, false
}

func configSourceDir(filePath string) string {
	dir := path.Dir(strings.ReplaceAll(filePath, "\\", "/"))
	if dir == "." || dir == "/" {
		return "."
	}
	return strings.TrimPrefix(dir, "./")
}

func scopedConfigPaths(sourceDir string, values []string) []string {
	scoped := make([]string, 0, len(values))
	for _, value := range values {
		if sourceDir == "" || sourceDir == "." {
			scoped = append(scoped, value)
			continue
		}
		scoped = append(scoped, path.Join(sourceDir, value))
	}
	return scoped
}

func scopedConfigPath(sourceDir string, value string) string {
	if value == "" || sourceDir == "" || sourceDir == "." || isAbsoluteConfigPath(value) {
		return value
	}
	return path.Join(sourceDir, value)
}

func isAbsoluteConfigPath(value string) bool {
	return path.IsAbs(value) || isWindowsAbsoluteConfigPath(value)
}

func isWindowsAbsoluteConfigPath(value string) bool {
	return strings.HasPrefix(value, `\\`) || isWindowsDriveAbsolutePath(value)
}

func isWindowsDriveAbsolutePath(value string) bool {
	if len(value) < 3 {
		return false
	}
	return isASCIIAlpha(value[0]) && value[1] == ':' && isWindowsPathSeparator(value[2])
}

func isASCIIAlpha(value byte) bool {
	return (value >= 'A' && value <= 'Z') || (value >= 'a' && value <= 'z')
}

func isWindowsPathSeparator(value byte) bool {
	return value == '\\' || value == '/'
}

func scopedConfigExcludePaths(sourceDir string, values []string) []string {
	scoped := make([]string, 0, len(values))
	for _, value := range values {
		if sourceDir == "" || sourceDir == "." || !strings.Contains(value, "/") {
			scoped = append(scoped, value)
			continue
		}
		scoped = append(scoped, value)
		scoped = append(scoped, path.Join(sourceDir, value))
	}
	return scoped
}

func validateKnownKeys(root *yaml.Node) error {
	node, ok, err := documentMapping(root, "root")
	if err != nil || !ok {
		return err
	}
	return validateMappingKeys(node, "root", set("rules", "go", "typescript"), validateTopLevelSection)
}

func validateTopLevelSection(key string, value *yaml.Node) error {
	switch key {
	case "rules":
		return validateRulesKeys(value)
	case "go":
		return validateGoKeys(value)
	case "typescript":
		return validateTypeScriptKeys(value)
	default:
		return nil
	}
}

func validateRulesKeys(node *yaml.Node) error {
	return validateMappingKeys(node, "rules", nil, func(ruleID string, value *yaml.Node) error {
		return validateMappingKeys(value, "rules."+ruleID, set("severity", "disabled", "reason", "threshold", "max"), nil)
	})
}

func validateGoKeys(node *yaml.Node) error {
	return validateMappingKeys(
		node,
		"go",
		set(
			"coverage_threshold",
			"coverage_profile",
			"targets",
			"exclude",
			"dry_max_candidates",
			"dry_paths",
			"dry_exclude",
			"dry",
			"crap_max_score",
			"mutation",
			"dependency_boundaries",
		),
		func(key string, value *yaml.Node) error {
			switch key {
			case "dry":
				return validateGoDryKeys(value)
			case "mutation":
				return validateMappingKeys(value, "go.mutation", set("targets", "exclude"), nil)
			case "dependency_boundaries":
				return validateDependencyBoundaryKeys(value, "go.dependency_boundaries")
			default:
				return nil
			}
		},
	)
}

func validateGoDryKeys(node *yaml.Node) error {
	return validateMappingKeys(
		node,
		"go.dry",
		set("max_findings", "paths", "exclude", "structural", "copied_blocks"),
		func(key string, value *yaml.Node) error {
			switch key {
			case "structural":
				return validateMappingKeys(value, "go.dry.structural", set("enabled", "threshold", "min_lines", "min_nodes"), nil)
			case "copied_blocks":
				return validateMappingKeys(value, "go.dry.copied_blocks", set("enabled", "min_tokens"), nil)
			default:
				return nil
			}
		},
	)
}

func validateTypeScriptKeys(node *yaml.Node) error {
	return validateMappingKeys(
		node,
		"typescript",
		set("coverage_threshold", "complexity_max", "dry", "mutation_targets", "dependency_boundaries"),
		func(key string, value *yaml.Node) error {
			switch key {
			case "dry":
				return validateMappingKeys(value, "typescript.dry", set("max_findings", "paths", "exclude", "copied_blocks"), func(key string, value *yaml.Node) error {
					if key == "copied_blocks" {
						return validateMappingKeys(value, "typescript.dry.copied_blocks", set("enabled", "min_tokens"), nil)
					}
					return nil
				})
			case "dependency_boundaries":
				return validateDependencyBoundaryKeys(value, "typescript.dependency_boundaries")
			default:
				return nil
			}
		},
	)
}

func validateDependencyBoundaryKeys(node *yaml.Node, field string) error {
	if node.Kind == 0 || node.Tag == "!!null" {
		return nil
	}
	if node.Kind != yaml.SequenceNode {
		return fmt.Errorf("%s must be a sequence", field)
	}
	for i, item := range node.Content {
		if err := validateMappingKeys(item, fmt.Sprintf("%s[%d]", field, i), set("from", "allow"), nil); err != nil {
			return err
		}
	}
	return nil
}

func validateMappingKeys(
	node *yaml.Node,
	field string,
	allowed map[string]struct{},
	visit func(string, *yaml.Node) error,
) error {
	if node.Kind == 0 || node.Tag == "!!null" {
		return nil
	}
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("%s must be a mapping", field)
	}
	seen := map[string]struct{}{}
	for i := 0; i < len(node.Content); i += 2 {
		if err := validateMappingEntry(field, allowed, visit, seen, node.Content[i], node.Content[i+1]); err != nil {
			return err
		}
	}
	return nil
}

func validateMappingEntry(
	field string,
	allowed map[string]struct{},
	visit func(string, *yaml.Node) error,
	seen map[string]struct{},
	keyNode *yaml.Node,
	valueNode *yaml.Node,
) error {
	key, err := mappingKey(field, keyNode)
	if err != nil {
		return err
	}
	if _, ok := seen[key]; ok {
		return fmt.Errorf("%s.%s is duplicated", field, key)
	}
	seen[key] = struct{}{}
	if err := validateAllowedKey(field, allowed, key); err != nil {
		return err
	}
	if visit == nil {
		return nil
	}
	return visit(key, valueNode)
}

func mappingKey(field string, node *yaml.Node) (string, error) {
	if node.Kind != yaml.ScalarNode || node.Value == "" {
		return "", fmt.Errorf("%s contains an invalid key", field)
	}
	return node.Value, nil
}

func validateAllowedKey(field string, allowed map[string]struct{}, key string) error {
	if allowed == nil {
		return nil
	}
	if _, ok := allowed[key]; ok {
		return nil
	}
	return fmt.Errorf("%s.%s is not supported", field, key)
}

func documentMapping(root *yaml.Node, field string) (*yaml.Node, bool, error) {
	node := unwrapDocumentNode(root)
	if emptyYAMLNode(node) {
		return nil, false, nil
	}
	if node.Kind != yaml.MappingNode {
		return nil, false, fmt.Errorf("%s must be a mapping", field)
	}
	return node, true, nil
}

func unwrapDocumentNode(root *yaml.Node) *yaml.Node {
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		return root.Content[0]
	}
	return root
}

func emptyYAMLNode(node *yaml.Node) bool {
	return node.Kind == 0 || node.Tag == "!!null"
}

func set(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func validate(cfg Config) error {
	for ruleID, rule := range cfg.Rules {
		if err := validateRule(ruleID, rule); err != nil {
			return err
		}
	}
	for i, boundary := range cfg.Go.DependencyBoundaries {
		if err := validateDependencyBoundary(i, boundary); err != nil {
			return err
		}
	}
	return validateGoTargets(cfg.Go)
}

func validateGoTargets(cfg GoConfig) error {
	if err := validateDryBudgets(cfg); err != nil {
		return err
	}
	if err := validateDryEngineTargets(cfg.DRY); err != nil {
		return err
	}
	return validateGoThresholds(cfg)
}

func validateDryBudgets(cfg GoConfig) error {
	if cfg.DRYMaxCandidatesSet && cfg.DRYMaxCandidates < 0 {
		return fmt.Errorf("go.dry_max_candidates must be non-negative")
	}
	if cfg.DRY.MaxFindingsSet && cfg.DRY.MaxFindings < 0 {
		return fmt.Errorf("go.dry.max_findings must be non-negative")
	}
	return nil
}

func validateGoThresholds(cfg GoConfig) error {
	if cfg.CoverageThreshold > 0 && cfg.CoverageThreshold < MinimumGoCoverageThreshold {
		return fmt.Errorf("go.coverage_threshold must be at least %.1f", float64(MinimumGoCoverageThreshold))
	}
	if cfg.CRAPMaxScore > 0 && cfg.CRAPMaxScore > MaximumGoCRAPScore {
		return fmt.Errorf("go.crap_max_score must be at most %.1f", float64(MaximumGoCRAPScore))
	}
	return nil
}

func validateDryEngineTargets(cfg DryConfig) error {
	if cfg.Structural.Threshold < 0 || cfg.Structural.Threshold > 1 {
		return fmt.Errorf("go.dry.structural.threshold must be between 0 and 1")
	}
	if cfg.Structural.MinLines < 0 {
		return fmt.Errorf("go.dry.structural.min_lines must be non-negative")
	}
	if cfg.Structural.MinNodes < 0 {
		return fmt.Errorf("go.dry.structural.min_nodes must be non-negative")
	}
	if cfg.CopiedBlocks.MinTokens < 0 {
		return fmt.Errorf("go.dry.copied_blocks.min_tokens must be non-negative")
	}
	return nil
}

func validateRule(ruleID string, rule RuleConfig) error {
	switch rule.Severity {
	case "", "error", "warn":
	default:
		return fmt.Errorf("rules.%s.severity must be error or warn", ruleID)
	}
	if rule.Disabled && rule.Reason == "" {
		return fmt.Errorf("rules.%s.reason is required when disabled is true", ruleID)
	}
	return nil
}

func validateDependencyBoundary(index int, boundary DependencyBoundary) error {
	if boundary.From == "" {
		return fmt.Errorf("go.dependency_boundaries[%d].from cannot be empty", index)
	}
	return nil
}
