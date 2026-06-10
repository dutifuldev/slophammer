use crate::core::{Finding, Severity};
use crate::scan::Snapshot;
use serde::Deserialize;
use std::collections::BTreeMap;
use thiserror::Error;
use yaml_serde::Value;

#[derive(Clone, Debug, Default, PartialEq)]
pub struct Config {
    pub rules: BTreeMap<String, RuleConfig>,
    pub rust: Option<RustConfig>,
}

#[derive(Clone, Debug, Default, Deserialize, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct RuleConfig {
    pub severity: Option<RuleSeverity>,
    #[serde(default)]
    pub disabled: bool,
    pub reason: Option<String>,
    pub threshold: Option<f64>,
    pub max: Option<f64>,
}

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum RuleSeverity {
    Error,
    Warn,
}

impl From<RuleSeverity> for Severity {
    fn from(value: RuleSeverity) -> Self {
        match value {
            RuleSeverity::Error => Self::Error,
            RuleSeverity::Warn => Self::Warn,
        }
    }
}

#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct RustConfig {
    pub coverage: Option<RustCoverage>,
    pub complexity: Option<RustComplexity>,
    #[serde(default)]
    pub targets: Vec<String>,
    #[serde(default)]
    pub exclude: Vec<String>,
    pub dry: Option<RustDry>,
    #[serde(rename = "unsafe")]
    pub unsafe_policy: Option<RustUnsafe>,
    pub mutation: Option<RustMutation>,
    #[serde(default)]
    pub dependency_boundaries: Vec<DependencyBoundary>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct RustCoverage {
    pub threshold: u32,
    #[serde(default)]
    pub paths: Vec<String>,
    #[serde(default)]
    pub exclude: Vec<String>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct RustComplexity {
    pub cognitive_max: u32,
}

#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct RustDry {
    pub max_findings: usize,
    #[serde(default)]
    pub paths: Vec<String>,
    #[serde(default)]
    pub exclude: Vec<String>,
    pub copied_blocks: Option<CopiedBlocks>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct CopiedBlocks {
    pub enabled: bool,
    pub min_tokens: usize,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct RustUnsafe {
    pub policy: UnsafePolicy,
    #[serde(default)]
    pub allow: Vec<UnsafeAllow>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum UnsafePolicy {
    Forbid,
    AllowDocumented,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct UnsafeAllow {
    pub path: String,
    pub reason: String,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct RustMutation {
    #[serde(default)]
    pub targets: Vec<String>,
    #[serde(default)]
    pub exclude: Vec<String>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq)]
#[serde(deny_unknown_fields)]
pub struct DependencyBoundary {
    pub from: String,
    #[serde(default)]
    pub allow: Vec<String>,
}

#[derive(Debug, Error)]
pub enum ConfigError {
    #[error("parse failed: {0}")]
    Parse(#[from] yaml_serde::Error),
    #[error("{0}")]
    Validation(String),
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct RootConfig {
    #[serde(default)]
    rules: BTreeMap<String, RuleConfig>,
    #[serde(default)]
    go: Option<Value>,
    #[serde(default)]
    typescript: Option<Value>,
    #[serde(default)]
    rust: Option<RustConfig>,
}

pub fn load(snapshot: &Snapshot) -> Result<Config, ConfigError> {
    let Some(content) = snapshot
        .file("slophammer.yml")
        .or_else(|| snapshot.file("slophammer.yaml"))
        .map(|file| file.content.as_str())
    else {
        return Ok(Config::default());
    };
    parse(content)
}

pub fn parse(content: &str) -> Result<Config, ConfigError> {
    let root: RootConfig = yaml_serde::from_str(content)?;
    let _known_sections = (&root.go, &root.typescript);
    let config = Config {
        rules: root.rules,
        rust: root.rust,
    };
    validate(&config)?;
    Ok(config)
}

fn validate(config: &Config) -> Result<(), ConfigError> {
    for (rule_id, rule) in &config.rules {
        if rule.disabled && rule.reason.as_deref().unwrap_or("").trim().is_empty() {
            return Err(ConfigError::Validation(format!(
                "rules.{rule_id}.reason is required when disabled is true"
            )));
        }
    }
    let Some(rust) = &config.rust else {
        return Ok(());
    };
    let coverage_threshold = rust.coverage.as_ref().map(|coverage| coverage.threshold);
    if let Some(threshold) = coverage_threshold {
        if threshold < 85 {
            return Err(ConfigError::Validation(
                "rust coverage threshold must be at least 85".to_owned(),
            ));
        }
    }
    if let Some(complexity) = &rust.complexity {
        if complexity.cognitive_max > 8 {
            return Err(ConfigError::Validation(
                "rust complexity cognitive_max must be at most 8".to_owned(),
            ));
        }
    }
    if let Some(dry) = &rust.dry {
        if dry.max_findings != 0 {
            return Err(ConfigError::Validation(
                "rust dry max_findings must be 0 for production code".to_owned(),
            ));
        }
        if let Some(copied_blocks) = &dry.copied_blocks {
            if copied_blocks.min_tokens == 0 {
                return Err(ConfigError::Validation(
                    "rust dry copied_blocks min_tokens must be positive".to_owned(),
                ));
            }
        }
    }
    if let Some(unsafe_policy) = &rust.unsafe_policy {
        for allow in &unsafe_policy.allow {
            if allow.path.trim().is_empty() || allow.reason.trim().is_empty() {
                return Err(ConfigError::Validation(
                    "rust unsafe allow entries require path and reason".to_owned(),
                ));
            }
        }
    }
    for boundary in &rust.dependency_boundaries {
        if boundary.from.trim().is_empty() {
            return Err(ConfigError::Validation(
                "rust dependency boundaries require from".to_owned(),
            ));
        }
    }
    Ok(())
}

pub fn apply_rule_config(config: &Config, findings: &mut [Finding]) {
    for finding in findings {
        if let Some(severity) = rule_severity(config, &finding.rule_id) {
            finding.severity = severity.into();
        }
    }
}

pub fn rule_severity(config: &Config, rule_id: &str) -> Option<RuleSeverity> {
    config.rules.get(rule_id).and_then(|rule| rule.severity)
}

pub fn rust_targets(config: &Config) -> Vec<String> {
    config
        .rust
        .as_ref()
        .map(|rust| {
            if rust.targets.is_empty() {
                vec![".".to_owned()]
            } else {
                rust.targets.clone()
            }
        })
        .unwrap_or_else(|| vec![".".to_owned()])
}

pub fn rust_exclude(config: &Config) -> Vec<String> {
    config
        .rust
        .as_ref()
        .map(|rust| rust.exclude.clone())
        .unwrap_or_default()
}

pub fn rust_coverage_threshold(config: &Config) -> u32 {
    config
        .rust
        .as_ref()
        .and_then(|rust| rust.coverage.as_ref().map(|coverage| coverage.threshold))
        .unwrap_or(85)
}

pub fn rust_dry_paths(config: &Config) -> Vec<String> {
    config
        .rust
        .as_ref()
        .and_then(|rust| rust.dry.as_ref())
        .map(|dry| {
            if dry.paths.is_empty() {
                rust_targets(config)
            } else {
                dry.paths.clone()
            }
        })
        .unwrap_or_else(|| rust_targets(config))
}

pub fn rust_dry_exclude(config: &Config) -> Vec<String> {
    config
        .rust
        .as_ref()
        .and_then(|rust| rust.dry.as_ref())
        .map(|dry| dry.exclude.clone())
        .unwrap_or_else(|| rust_exclude(config))
}

pub fn rust_dry_min_tokens(config: &Config) -> usize {
    config
        .rust
        .as_ref()
        .and_then(|rust| rust.dry.as_ref())
        .and_then(|dry| dry.copied_blocks.as_ref())
        .map(|copied_blocks| copied_blocks.min_tokens)
        .unwrap_or(100)
}

pub fn rust_dry_copied_blocks_enabled(config: &Config) -> bool {
    config
        .rust
        .as_ref()
        .and_then(|rust| rust.dry.as_ref())
        .and_then(|dry| dry.copied_blocks.as_ref())
        .map(|copied_blocks| copied_blocks.enabled)
        .unwrap_or(true)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rejects_weak_rust_coverage() {
        let error = parse("rust:\n  coverage:\n    threshold: 40\n").unwrap_err();
        assert!(error.to_string().contains("at least 85"));
    }

    #[test]
    fn rejects_unknown_rust_keys() {
        let error = parse("rust:\n  surprise: true\n").unwrap_err();
        assert!(error.to_string().contains("unknown field"));
    }

    #[test]
    fn parses_full_rust_config() {
        let config = parse(
            r#"
go:
  ignored: true
typescript:
  ignored: true
rules:
  repo.readme-required:
    severity: warn
rust:
  coverage:
    threshold: 90
    paths:
      - crates
    exclude:
      - target/**
  complexity:
    cognitive_max: 7
  targets:
    - crates
  exclude:
    - generated/**
  dry:
    max_findings: 0
    paths:
      - crates/slophammer-cli/src/rust_rules
    exclude:
      - tests/**
    copied_blocks:
      enabled: true
      min_tokens: 50
  unsafe:
    policy: allow_documented
    allow:
      - path: crates/slophammer-cli/src/rust_rules/ffi.rs
        reason: ffi boundary
  mutation:
    targets:
      - crates/slophammer-cli/src/rust_rules
    exclude:
      - generated/**
  dependency_boundaries:
    - from: crates/slophammer-cli
      allow: []
"#,
        )
        .expect("config");
        assert_eq!(rust_coverage_threshold(&config), 90);
        assert_eq!(rust_targets(&config), vec!["crates"]);
        assert_eq!(
            rust_dry_paths(&config),
            vec!["crates/slophammer-cli/src/rust_rules".to_owned()]
        );
        assert_eq!(rust_dry_min_tokens(&config), 50);
        assert_eq!(
            rule_severity(&config, "repo.readme-required"),
            Some(RuleSeverity::Warn)
        );
    }

    #[test]
    fn rejects_weak_complexity_and_dry_budget() {
        let complexity = parse("rust:\n  complexity:\n    cognitive_max: 9\n").unwrap_err();
        assert!(complexity.to_string().contains("at most 8"));

        let dry = parse("rust:\n  dry:\n    max_findings: 1\n").unwrap_err();
        assert!(dry.to_string().contains("must be 0"));
    }

    #[test]
    fn rejects_unknown_rule_config_keys() {
        let error = parse(
            r#"
rules:
  repo.readme-required:
    surprise: true
"#,
        )
        .unwrap_err();
        assert!(error.to_string().contains("unknown field"));
    }

    #[test]
    fn disabled_rules_require_a_reason() {
        let error = parse(
            r#"
rules:
  repo.readme-required:
    disabled: true
"#,
        )
        .unwrap_err();
        assert!(error.to_string().contains("reason is required"));
    }

    #[test]
    fn applies_rule_severity_overrides() {
        let config = parse(
            r#"
rules:
  repo.readme-required:
    severity: warn
"#,
        )
        .expect("config");
        let mut findings = vec![Finding {
            rule_id: "repo.readme-required".to_owned(),
            severity: Severity::Error,
            path: "README.md".to_owned(),
            message: "missing".to_owned(),
        }];

        apply_rule_config(&config, &mut findings);

        assert_eq!(findings[0].severity, Severity::Warn);
    }
}
