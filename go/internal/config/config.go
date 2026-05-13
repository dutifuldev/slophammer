package config

import (
	"fmt"

	"github.com/dutifuldev/slophammer/go/internal/repo"
	"gopkg.in/yaml.v3"
)

const (
	DefaultFileName = "slophammer.yml"
	AltFileName     = "slophammer.yaml"
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
	CRAPMaxScore         float64              `yaml:"crap_max_score"`
	MutationTargets      []string             `yaml:"mutation_targets"`
	DependencyBoundaries []DependencyBoundary `yaml:"dependency_boundaries"`
}

type DependencyBoundary struct {
	From  string   `yaml:"from"`
	Allow []string `yaml:"allow"`
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
	for _, file := range snapshot.FilesNamedFold(DefaultFileName, AltFileName) {
		return file, true
	}
	return repo.File{}, false
}

func validate(cfg Config) error {
	for ruleID, rule := range cfg.Rules {
		switch rule.Severity {
		case "", "error", "warn":
		default:
			return fmt.Errorf("rules.%s.severity must be error or warn", ruleID)
		}
		if rule.Disabled && rule.Reason == "" {
			return fmt.Errorf("rules.%s.reason is required when disabled is true", ruleID)
		}
	}
	for i, boundary := range cfg.Go.DependencyBoundaries {
		if boundary.From == "" {
			return fmt.Errorf("go.dependency_boundaries[%d].from cannot be empty", i)
		}
	}
	return nil
}
