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
    use std::path::Path;
    use tempfile::{Builder, TempDir};

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
        let fixture = fixture("rust-clean");
        let result = check(CheckOptions {
            root: fixture_path(&fixture),
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
            let fixture = fixture(fixture_name);
            let result = check(CheckOptions {
                root: fixture_path(&fixture),
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
        let clean_fixture = fixture("rust-clean");
        let dry = dry(DirectOptions {
            root: fixture_path(&clean_fixture),
            format: OutputFormat::Text,
            max_findings: None,
        });
        assert_eq!(dry.code, EXIT_OK);
        assert!(dry.stdout.contains("OK: no findings"));

        let bad_dependency_fixture = fixture("rust-bad-dependency");
        let boundaries = boundaries(DirectOptions {
            root: fixture_path(&bad_dependency_fixture),
            format: OutputFormat::Sarif,
            max_findings: None,
        });
        assert_eq!(boundaries.code, EXIT_FINDINGS);
        assert!(
            boundaries
                .stdout
                .contains("rust.dependency-boundaries-required")
        );

        let unsafe_fixture = fixture("rust-unsafe");
        let unsafe_result = unsafe_policy(DirectOptions {
            root: fixture_path(&unsafe_fixture),
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
        write_file(root.path(), "AGENTS.md", "# Agents\n");
        write_file(
            root.path(),
            ".github/workflows/ci.yml",
            "jobs:\n  ci:\n    steps:\n      - run: echo ok\n",
        );
        write_file(
            root.path(),
            "slophammer.yml",
            "rules:\n  repo.readme-required:\n    severity: warn\n",
        );

        let result = check(CheckOptions {
            root: fixture_path(&root),
            format: OutputFormat::Json,
            execute: false,
            only_rule_ids: vec!["repo.readme-required".to_owned()],
        });

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
            let fixture = fixture(fixture_name);
            let result = check(CheckOptions {
                root: fixture_path(&fixture),
                format: OutputFormat::Json,
                execute: false,
                only_rule_ids: Vec::new(),
            });
            assert_eq!(result.code, EXIT_ERROR, "{fixture_name}");
            assert!(result.stderr.contains("config failed"), "{fixture_name}");
        }
    }

    fn fixture(name: &str) -> TempDir {
        let root = temp_root(name);
        write_rust_fixture(root.path(), name);
        root
    }

    fn fixture_path(root: &TempDir) -> String {
        root.path().to_string_lossy().into_owned()
    }

    fn write_rust_fixture(root: &Path, name: &str) {
        match name {
            "rust-clean" => write_clean_fixture(
                root,
                CLEAN_WORKFLOW,
                "pub fn message() -> &'static str {\n    \"ok\"\n}\n",
            ),
            "rust-bad-dependency" => write_bad_dependency_fixture(root),
            "rust-invalid-config" => write_invalid_config_fixture(root),
            "rust-unknown-config" => write_unknown_config_fixture(root),
            "rust-unsafe" => write_unsafe_fixture(root),
            _ => write_clean_fixture(
                root,
                "jobs:\n  rust:\n    steps:\n      - run: cargo check --workspace\n",
                "pub fn message() -> &'static str {\n    \"ok\"\n}\n",
            ),
        }
    }

    fn write_clean_fixture(root: &Path, workflow: &str, source: &str) {
        write_file(root, "README.md", "# Rust Fixture\n");
        write_file(root, "AGENTS.md", "# Agents\n");
        write_file(
            root,
            "Cargo.toml",
            "[package]\nname = \"rust-fixture\"\nversion = \"0.1.0\"\nedition = \"2024\"\nrust-version = \"1.86\"\n",
        );
        write_file(root, "src/lib.rs", source);
        write_file(root, "clippy.toml", "cognitive-complexity-threshold = 8\n");
        write_file(root, ".github/workflows/ci.yml", workflow);
        write_file(root, "slophammer.yml", CLEAN_CONFIG);
    }

    fn write_bad_dependency_fixture(root: &Path) {
        write_clean_fixture(
            root,
            CLEAN_WORKFLOW,
            "pub fn message() -> &'static str {\n    local_dep::message()\n}\n",
        );
        write_file(
            root,
            "Cargo.toml",
            "[package]\nname = \"rust-fixture\"\nversion = \"0.1.0\"\nedition = \"2024\"\nrust-version = \"1.86\"\n\n[dependencies]\nlocal-dep = { path = \"local-dep\" }\n",
        );
        write_file(
            root,
            "local-dep/Cargo.toml",
            "[package]\nname = \"local-dep\"\nversion = \"0.1.0\"\nedition = \"2024\"\nrust-version = \"1.86\"\n",
        );
        write_file(
            root,
            "local-dep/src/lib.rs",
            "pub fn message() -> &'static str {\n    \"dep\"\n}\n",
        );
    }

    fn write_unsafe_fixture(root: &Path) {
        write_clean_fixture(
            root,
            CLEAN_WORKFLOW,
            "pub fn pointer_is_null() -> bool {\n    let value = unsafe { core::ptr::null::<u8>().as_ref() };\n    value.is_none()\n}\n",
        );
    }

    fn write_invalid_config_fixture(root: &Path) {
        write_clean_fixture(
            root,
            CLEAN_WORKFLOW,
            "pub fn message() -> &'static str {\n    \"ok\"\n}\n",
        );
        write_file(
            root,
            "slophammer.yml",
            "rust:\n  coverage:\n    threshold: 40\n",
        );
    }

    fn write_unknown_config_fixture(root: &Path) {
        write_clean_fixture(
            root,
            CLEAN_WORKFLOW,
            "pub fn message() -> &'static str {\n    \"ok\"\n}\n",
        );
        write_file(root, "slophammer.yml", "rust:\n  not_a_real_key: true\n");
    }

    fn temp_root(name: &str) -> TempDir {
        Builder::new()
            .prefix(&format!("slophammer-rs-{name}-"))
            .tempdir()
            .expect("create temp root")
    }

    fn write_file(root: &Path, path: &str, content: &str) {
        let path = root.join(path);
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent).expect("create parent");
        }
        fs::write(path, content).expect("write file");
    }

    const CLEAN_WORKFLOW: &str = r#"name: CI
on: [push]
jobs:
  rust:
    runs-on: ubuntu-latest
    steps:
      - run: cargo check --workspace
      - run: cargo fmt --check
      - run: cargo clippy --workspace --all-targets -- -D warnings
      - run: cargo test --workspace --all-targets
      - run: cargo llvm-cov --workspace --fail-under-lines 85
      - run: cargo audit
      - run: slophammer-rs dry .
      - run: cargo mutants --workspace
"#;

    const CLEAN_CONFIG: &str = r#"rust:
  coverage:
    threshold: 85
  complexity:
    cognitive_max: 8
  targets:
    - src
  dry:
    max_findings: 0
    paths:
      - src
    copied_blocks:
      enabled: true
      min_tokens: 100
  unsafe:
    policy: forbid
  mutation:
    targets:
      - src
  dependency_boundaries:
    - from: .
      allow: []
"#;
}
