package config

import (
	"fmt"

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
	Rules map[string]RuleConfig `yaml:"rules"`
	Go    GoConfig              `yaml:"go"`
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
	DRYMaxCandidates     int                  `yaml:"dry_max_candidates"`
	DRYMaxCandidatesSet  bool                 `yaml:"-"`
	DRYPaths             []string             `yaml:"dry_paths"`
	DRYExclude           []string             `yaml:"dry_exclude"`
	CRAPMaxScore         float64              `yaml:"crap_max_score"`
	MutationTargets      []string             `yaml:"mutation_targets"`
	DependencyBoundaries []DependencyBoundary `yaml:"dependency_boundaries"`
}

type DependencyBoundary struct {
	From  string   `yaml:"from"`
	Allow []string `yaml:"allow"`
}

func (cfg *GoConfig) UnmarshalYAML(value *yaml.Node) error {
	type goConfig struct {
		CoverageThreshold    float64              `yaml:"coverage_threshold"`
		DRYMaxCandidates     *int                 `yaml:"dry_max_candidates"`
		DRYPaths             []string             `yaml:"dry_paths"`
		DRYExclude           []string             `yaml:"dry_exclude"`
		CRAPMaxScore         float64              `yaml:"crap_max_score"`
		MutationTargets      []string             `yaml:"mutation_targets"`
		DependencyBoundaries []DependencyBoundary `yaml:"dependency_boundaries"`
	}
	var parsed goConfig
	if err := value.Decode(&parsed); err != nil {
		return err
	}
	cfg.CoverageThreshold = parsed.CoverageThreshold
	if parsed.DRYMaxCandidates != nil {
		cfg.DRYMaxCandidates = *parsed.DRYMaxCandidates
		cfg.DRYMaxCandidatesSet = true
	}
	cfg.DRYPaths = parsed.DRYPaths
	cfg.DRYExclude = parsed.DRYExclude
	cfg.CRAPMaxScore = parsed.CRAPMaxScore
	cfg.MutationTargets = parsed.MutationTargets
	cfg.DependencyBoundaries = parsed.DependencyBoundaries
	return nil
}

func Load(snapshot repo.Snapshot) (Config, error) {
	file, ok := configFile(snapshot)
	if !ok {
		return Config{}, nil
	}
	var cfg Config
	if err := yaml.Unmarshal([]byte(file.Content), &cfg); err != nil {
		return Config{}, fmt.Errorf("%s: %w", file.Path, err)
	}
	if err := validate(cfg); err != nil {
		return Config{}, fmt.Errorf("%s: %w", file.Path, err)
	}
	return cfg, nil
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
	if cfg.DRYMaxCandidatesSet && cfg.DRYMaxCandidates < 0 {
		return fmt.Errorf("go.dry_max_candidates must be non-negative")
	}
	if cfg.CoverageThreshold > 0 && cfg.CoverageThreshold < MinimumGoCoverageThreshold {
		return fmt.Errorf("go.coverage_threshold must be at least %.1f", float64(MinimumGoCoverageThreshold))
	}
	if cfg.CRAPMaxScore > 0 && cfg.CRAPMaxScore > MaximumGoCRAPScore {
		return fmt.Errorf("go.crap_max_score must be at most %.1f", float64(MaximumGoCRAPScore))
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
