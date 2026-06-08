use crate::config::Config;
use crate::core::{EXIT_ERROR, EXIT_FINDINGS, EXIT_OK, Finding, Report, RuleDefinition};
use crate::exec::{RealRunner, Runner};
use crate::report::{new_report, write_json, write_sarif, write_text};
use crate::scan::{Snapshot, scan_repo};
use std::fmt;
use thiserror::Error;

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum OutputFormat {
    Text,
    Json,
    Sarif,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct CheckOptions {
    pub root: String,
    pub format: OutputFormat,
    pub execute: bool,
    pub only_rule_ids: Vec<String>,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct DirectOptions {
    pub root: String,
    pub format: OutputFormat,
    pub max_findings: Option<usize>,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct AppResult {
    pub code: i32,
    pub stdout: String,
    pub stderr: String,
}

#[derive(Debug, Error)]
pub enum AppError {
    #[error("scan failed: {0}")]
    Scan(#[from] crate::scan::ScanError),
    #[error("config failed: {0}")]
    Config(#[from] crate::config::ConfigError),
    #[error("report failed: {0}")]
    Report(#[from] crate::report::ReportError),
    #[error("unknown rule: {0}")]
    UnknownRule(String),
}

pub fn check(options: CheckOptions) -> AppResult {
    check_with_runner(options, &RealRunner)
}

pub fn check_with_runner(options: CheckOptions, runner: &impl Runner) -> AppResult {
    match check_inner(options, runner) {
        Ok(result) => result,
        Err(error) => AppResult {
            code: EXIT_ERROR,
            stdout: String::new(),
            stderr: format!("check failed: {error}\n"),
        },
    }
}

pub fn dry(options: DirectOptions) -> AppResult {
    direct(options, |snapshot, config, max_findings| {
        crate::rust_rules::dry_findings(snapshot, config, max_findings)
    })
}

pub fn boundaries(options: DirectOptions) -> AppResult {
    direct(options, |snapshot, config, _| {
        crate::rust_rules::run_rules(
            snapshot,
            config,
            &[crate::rust_rules::rule_ids::RUST_DEPENDENCY_BOUNDARIES_REQUIRED.to_owned()],
        )
    })
}

pub fn unsafe_policy(options: DirectOptions) -> AppResult {
    direct(options, |snapshot, config, _| {
        crate::rust_rules::run_rules(
            snapshot,
            config,
            &[crate::rust_rules::rule_ids::RUST_UNSAFE_POLICY_REQUIRED.to_owned()],
        )
    })
}

pub fn explain(rule_id: &str) -> AppResult {
    match crate::rust_rules::explain(rule_id) {
        Some(text) => AppResult {
            code: EXIT_OK,
            stdout: text,
            stderr: String::new(),
        },
        None => AppResult {
            code: EXIT_ERROR,
            stdout: String::new(),
            stderr: format!("unknown rule: {rule_id}\n"),
        },
    }
}

pub fn rules(format: OutputFormat) -> AppResult {
    let definitions = crate::rust_rules::default_definitions();
    match render_rules(format, &definitions) {
        Ok(stdout) => AppResult {
            code: EXIT_OK,
            stdout,
            stderr: String::new(),
        },
        Err(error) => AppResult {
            code: EXIT_ERROR,
            stdout: String::new(),
            stderr: format!("rules failed: {error}\n"),
        },
    }
}

fn check_inner(options: CheckOptions, runner: &impl Runner) -> Result<AppResult, AppError> {
    validate_only_rules(&options.only_rule_ids)?;
    let snapshot = scan_repo(command_root(&options.root))?;
    let config = crate::config::load(&snapshot)?;
    let mut findings = crate::rust_rules::run_rules(&snapshot, &config, &options.only_rule_ids);
    if options.execute {
        findings.extend(crate::exec::execute_rust_checks(
            &snapshot,
            &config,
            &options.only_rule_ids,
            runner,
        ));
    }
    crate::config::apply_rule_config(&config, &mut findings);
    let report = new_report(findings);
    let stdout = render_report(options.format, &report)?;
    Ok(AppResult {
        code: if report.ok { EXIT_OK } else { EXIT_FINDINGS },
        stdout,
        stderr: String::new(),
    })
}

fn direct(
    options: DirectOptions,
    check: impl FnOnce(&Snapshot, &Config, usize) -> Vec<Finding>,
) -> AppResult {
    match direct_inner(options, check) {
        Ok(result) => result,
        Err(error) => AppResult {
            code: EXIT_ERROR,
            stdout: String::new(),
            stderr: format!("check failed: {error}\n"),
        },
    }
}

fn direct_inner(
    options: DirectOptions,
    check: impl FnOnce(&Snapshot, &Config, usize) -> Vec<Finding>,
) -> Result<AppResult, AppError> {
    let snapshot = scan_repo(command_root(&options.root))?;
    let config = crate::config::load(&snapshot)?;
    let mut findings = check(&snapshot, &config, options.max_findings.unwrap_or(0));
    crate::config::apply_rule_config(&config, &mut findings);
    let report = new_report(findings);
    let stdout = render_report(options.format, &report)?;
    Ok(AppResult {
        code: if report.ok { EXIT_OK } else { EXIT_FINDINGS },
        stdout,
        stderr: String::new(),
    })
}

fn validate_only_rules(only_rule_ids: &[String]) -> Result<(), AppError> {
    for rule_id in only_rule_ids {
        if !crate::rust_rules::known_rule(rule_id) {
            return Err(AppError::UnknownRule(rule_id.clone()));
        }
    }
    Ok(())
}

fn command_root(root: &str) -> &str {
    if root.is_empty() { "." } else { root }
}

fn render_report(format: OutputFormat, report: &Report) -> Result<String, AppError> {
    match format {
        OutputFormat::Text => Ok(write_text(report)),
        OutputFormat::Json => Ok(write_json(report)?),
        OutputFormat::Sarif => Ok(write_sarif(report)?),
    }
}

fn render_rules(
    format: OutputFormat,
    definitions: &[RuleDefinition],
) -> Result<String, serde_json::Error> {
    match format {
        OutputFormat::Text | OutputFormat::Sarif => Ok(rules_text(definitions)),
        OutputFormat::Json => Ok(format!("{}\n", serde_json::to_string_pretty(definitions)?)),
    }
}

fn rules_text(definitions: &[RuleDefinition]) -> String {
    let mut output = String::from("RULE ID\tCATEGORY\tSEVERITY\tSTATUS\tTOOL\n");
    for definition in definitions {
        output.push_str(&format!(
            "{}\t{}\t{}\t{}\t{}\n",
            definition.id,
            definition.category,
            definition.severity,
            definition.status,
            definition.tool.unwrap_or("")
        ));
    }
    output
}

impl fmt::Display for OutputFormat {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Text => formatter.write_str("text"),
            Self::Json => formatter.write_str("json"),
            Self::Sarif => formatter.write_str("sarif"),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::core::{EXIT_FINDINGS, EXIT_OK};
    use std::fs;
    use std::path::{Path, PathBuf};
    use std::time::{SystemTime, UNIX_EPOCH};

    #[test]
    fn unknown_only_rule_is_error() {
        let result = check(CheckOptions {
            root: ".".to_owned(),
            format: OutputFormat::Json,
            execute: false,
            only_rule_ids: vec!["missing.rule".to_owned()],
        });
        assert_eq!(result.code, EXIT_ERROR);
        assert!(result.stderr.contains("unknown rule"));
    }

    #[test]
    fn clean_fixture_has_no_findings() {
        let result = check(CheckOptions {
            root: fixture("rust-clean"),
            format: OutputFormat::Json,
            execute: false,
            only_rule_ids: Vec::new(),
        });
        assert_eq!(result.code, EXIT_OK);
        assert!(result.stdout.contains("\"ok\": true"));
    }

    #[test]
    fn rust_failure_fixtures_report_findings() {
        for fixture_name in [
            "rust-bad-dependency",
            "rust-missing-audit",
            "rust-missing-ci",
            "rust-missing-clippy",
            "rust-missing-coverage",
            "rust-missing-dry",
            "rust-missing-fmt",
            "rust-missing-msrv",
            "rust-missing-mutation",
            "rust-missing-tests",
            "rust-unsafe",
        ] {
            let result = check(CheckOptions {
                root: fixture(fixture_name),
                format: OutputFormat::Json,
                execute: false,
                only_rule_ids: Vec::new(),
            });
            assert_eq!(result.code, EXIT_FINDINGS, "{fixture_name}");
            assert!(result.stdout.contains("\"ok\": false"), "{fixture_name}");
        }
    }

    #[test]
    fn direct_commands_use_report_contract() {
        let dry = dry(DirectOptions {
            root: fixture("rust-clean"),
            format: OutputFormat::Text,
            max_findings: None,
        });
        assert_eq!(dry.code, EXIT_OK);
        assert!(dry.stdout.contains("OK: no findings"));

        let boundaries = boundaries(DirectOptions {
            root: fixture("rust-bad-dependency"),
            format: OutputFormat::Sarif,
            max_findings: None,
        });
        assert_eq!(boundaries.code, EXIT_FINDINGS);
        assert!(
            boundaries
                .stdout
                .contains("rust.dependency-boundaries-required")
        );

        let unsafe_result = unsafe_policy(DirectOptions {
            root: fixture("rust-unsafe"),
            format: OutputFormat::Json,
            max_findings: None,
        });
        assert_eq!(unsafe_result.code, EXIT_FINDINGS);
        assert!(unsafe_result.stdout.contains("rust.unsafe-policy-required"));
    }

    #[test]
    fn rules_and_explain_are_available() {
        let catalog = rules(OutputFormat::Json);
        assert_eq!(catalog.code, EXIT_OK);
        assert!(catalog.stdout.contains("rust.check-required"));

        let explanation = explain("rust.check-required");
        assert_eq!(explanation.code, EXIT_OK);
        assert!(
            explanation
                .stdout
                .contains("Rust projects should declare cargo check")
        );
    }

    #[test]
    fn check_applies_rule_severity_overrides() {
        let root = temp_root("rule-severity");
        write_file(&root, "AGENTS.md", "# Agents\n");
        write_file(
            &root,
            ".github/workflows/ci.yml",
            "jobs:\n  ci:\n    steps:\n      - run: echo ok\n",
        );
        write_file(
            &root,
            "slophammer.yml",
            "rules:\n  repo.readme-required:\n    severity: warn\n",
        );

        let result = check(CheckOptions {
            root: root.to_string_lossy().into_owned(),
            format: OutputFormat::Json,
            execute: false,
            only_rule_ids: vec!["repo.readme-required".to_owned()],
        });

        fs::remove_dir_all(&root).ok();
        assert_eq!(result.code, EXIT_FINDINGS);
        assert!(
            result
                .stdout
                .contains("\"rule_id\": \"repo.readme-required\"")
        );
        assert!(result.stdout.contains("\"severity\": \"warn\""));
    }

    #[test]
    fn config_error_fixtures_return_error_code() {
        for fixture_name in ["rust-invalid-config", "rust-unknown-config"] {
            let result = check(CheckOptions {
                root: fixture(fixture_name),
                format: OutputFormat::Json,
                execute: false,
                only_rule_ids: Vec::new(),
            });
            assert_eq!(result.code, EXIT_ERROR, "{fixture_name}");
            assert!(result.stderr.contains("config failed"), "{fixture_name}");
        }
    }

    fn fixture(name: &str) -> String {
        let manifest_dir = std::path::Path::new(env!("CARGO_MANIFEST_DIR"));
        manifest_dir
            .join("../../..")
            .join("fixtures/repos")
            .join(name)
            .to_string_lossy()
            .into_owned()
    }

    fn temp_root(name: &str) -> PathBuf {
        let nonce = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("clock")
            .as_nanos();
        let root = std::env::temp_dir().join(format!(
            "slophammer-rs-{name}-{}-{nonce}",
            std::process::id()
        ));
        fs::create_dir_all(&root).expect("create temp root");
        root
    }

    fn write_file(root: &Path, path: &str, content: &str) {
        let path = root.join(path);
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent).expect("create parent");
        }
        fs::write(path, content).expect("write file");
    }
}
