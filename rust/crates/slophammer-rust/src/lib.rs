mod boundaries;
mod definitions;
mod dry;
mod evidence;
mod scope;
mod unsafe_policy;

use definitions::definition;
pub use definitions::{default_definitions, rule_ids};
pub use dry::dry_findings;
use slophammer_config::Config;
use slophammer_core::{Finding, find_definition};
use slophammer_scan::Snapshot;

pub fn run_rules(snapshot: &Snapshot, config: &Config, only_rule_ids: &[String]) -> Vec<Finding> {
    let definitions = default_definitions();
    let wanted = |rule_id: &str| {
        only_rule_ids.is_empty() || only_rule_ids.iter().any(|wanted| wanted == rule_id)
    };
    let mut findings = Vec::new();
    for rule_id in definitions
        .iter()
        .map(|item| item.id)
        .filter(|id| wanted(id))
    {
        findings.extend(run_rule(rule_id, snapshot, config));
    }
    findings
}

pub fn explain(rule_id: &str) -> Option<String> {
    find_definition(&default_definitions(), rule_id).map(|definition| {
        format!(
            "{}\n{}\nseverity: {}\npath: {}\n",
            definition.id, definition.description, definition.severity, definition.path
        )
    })
}

pub fn known_rule(rule_id: &str) -> bool {
    find_definition(&default_definitions(), rule_id).is_some()
}

fn run_rule(rule_id: &str, snapshot: &Snapshot, config: &Config) -> Vec<Finding> {
    match rule_id {
        rule_ids::README_REQUIRED => repo_readme(snapshot),
        rule_ids::AGENTS_REQUIRED => repo_agents(snapshot),
        rule_ids::CI_REQUIRED => repo_ci(snapshot),
        rule_ids::RUST_MANIFEST_REQUIRED => rust_manifest(snapshot),
        rule_ids::RUST_MSRV_REQUIRED => rust_msrv(snapshot),
        rule_ids::RUST_CHECK_REQUIRED => cargo_command(snapshot, rule_id, "cargo check"),
        rule_ids::RUST_FMT_REQUIRED => rust_fmt(snapshot),
        rule_ids::RUST_CLIPPY_REQUIRED => rust_clippy(snapshot),
        rule_ids::RUST_TEST_REQUIRED => cargo_command(snapshot, rule_id, "cargo test"),
        rule_ids::RUST_COVERAGE_REQUIRED => rust_coverage(snapshot, config),
        rule_ids::RUST_COMPLEXITY_REQUIRED => rust_complexity(snapshot),
        rule_ids::RUST_DRY_REQUIRED => rust_dry_declaration(snapshot),
        rule_ids::RUST_MUTATION_REQUIRED => cargo_command(snapshot, rule_id, "cargo mutants"),
        rule_ids::RUST_UNSAFE_POLICY_REQUIRED => unsafe_policy::policy_findings(snapshot, config),
        rule_ids::RUST_DEPENDENCY_AUDIT_REQUIRED => rust_dependency_audit(snapshot),
        rule_ids::RUST_DEPENDENCY_BOUNDARIES_REQUIRED => {
            boundaries::boundary_findings(snapshot, config)
        }
        _ => Vec::new(),
    }
}

fn repo_readme(snapshot: &Snapshot) -> Vec<Finding> {
    missing(
        !snapshot.has_case_insensitive("README.md"),
        rule_ids::README_REQUIRED,
    )
}

fn repo_agents(snapshot: &Snapshot) -> Vec<Finding> {
    missing(
        !snapshot.has_case_insensitive("AGENTS.md"),
        rule_ids::AGENTS_REQUIRED,
    )
}

fn repo_ci(snapshot: &Snapshot) -> Vec<Finding> {
    missing(!snapshot.has_workflow(), rule_ids::CI_REQUIRED)
}

fn rust_manifest(snapshot: &Snapshot) -> Vec<Finding> {
    missing(
        is_rust_project(snapshot) && rust_manifests(snapshot).is_empty(),
        rule_ids::RUST_MANIFEST_REQUIRED,
    )
}

fn rust_msrv(snapshot: &Snapshot) -> Vec<Finding> {
    missing(
        is_rust_project(snapshot) && !has_msrv(snapshot),
        rule_ids::RUST_MSRV_REQUIRED,
    )
}

fn cargo_command(snapshot: &Snapshot, rule_id: &str, command: &str) -> Vec<Finding> {
    missing(
        is_rust_project(snapshot) && !evidence::command_text(snapshot).contains(command),
        rule_id,
    )
}

fn rust_fmt(snapshot: &Snapshot) -> Vec<Finding> {
    let evidence = evidence::command_text(snapshot);
    missing(
        is_rust_project(snapshot)
            && !(evidence.contains("cargo fmt") && evidence.contains("--check")),
        rule_ids::RUST_FMT_REQUIRED,
    )
}

fn rust_clippy(snapshot: &Snapshot) -> Vec<Finding> {
    let evidence = evidence::command_text(snapshot);
    missing(
        is_rust_project(snapshot)
            && !(evidence.contains("cargo clippy")
                && (evidence.contains("-d warnings") || evidence.contains("--deny warnings"))),
        rule_ids::RUST_CLIPPY_REQUIRED,
    )
}

fn rust_coverage(snapshot: &Snapshot, config: &Config) -> Vec<Finding> {
    let evidence = evidence::command_text(snapshot);
    let threshold = slophammer_config::rust_coverage_threshold(config).to_string();
    missing(
        is_rust_project(snapshot)
            && !(evidence.contains("cargo llvm-cov")
                && evidence.contains("--fail-under")
                && evidence.contains(&threshold)),
        rule_ids::RUST_COVERAGE_REQUIRED,
    )
}

fn rust_complexity(snapshot: &Snapshot) -> Vec<Finding> {
    missing(
        is_rust_project(snapshot) && !has_complexity_limit(snapshot),
        rule_ids::RUST_COMPLEXITY_REQUIRED,
    )
}

fn rust_dry_declaration(snapshot: &Snapshot) -> Vec<Finding> {
    missing(
        is_rust_project(snapshot)
            && !evidence::command_text(snapshot).contains("slophammer-rs dry"),
        rule_ids::RUST_DRY_REQUIRED,
    )
}

fn rust_dependency_audit(snapshot: &Snapshot) -> Vec<Finding> {
    let evidence = evidence::command_text(snapshot);
    missing(
        is_rust_project(snapshot)
            && !(evidence.contains("cargo audit") || evidence.contains("cargo deny")),
        rule_ids::RUST_DEPENDENCY_AUDIT_REQUIRED,
    )
}

fn missing(condition: bool, rule_id: &str) -> Vec<Finding> {
    if condition {
        vec![Finding::new(definition(rule_id))]
    } else {
        Vec::new()
    }
}

pub fn is_rust_project(snapshot: &Snapshot) -> bool {
    !rust_manifests(snapshot).is_empty()
        || snapshot
            .files
            .keys()
            .any(|path| rust_source_path(path) && !ignored_project_path(path))
        || evidence::command_text(snapshot).contains("cargo ")
}

fn rust_manifests(snapshot: &Snapshot) -> Vec<&str> {
    snapshot
        .files
        .keys()
        .map(String::as_str)
        .filter(|path| path.ends_with("Cargo.toml") && !ignored_project_path(path))
        .collect()
}

fn has_msrv(snapshot: &Snapshot) -> bool {
    snapshot
        .files
        .values()
        .filter(|file| file.path.ends_with("Cargo.toml") && !ignored_project_path(&file.path))
        .any(|file| file.content.contains("rust-version"))
        || snapshot
            .file("rust-toolchain.toml")
            .is_some_and(|file| toolchain_declares_version(&file.content))
}

fn toolchain_declares_version(content: &str) -> bool {
    content
        .lines()
        .filter_map(|line| line.split_once('='))
        .any(|(key, value)| key.trim() == "channel" && value.chars().any(|ch| ch.is_ascii_digit()))
}

fn has_complexity_limit(snapshot: &Snapshot) -> bool {
    snapshot
        .files
        .values()
        .filter(|file| {
            file.path.ends_with("clippy.toml")
                || file.path.ends_with(".clippy.toml")
                || file.path.ends_with("rust/clippy.toml")
        })
        .any(|file| {
            let normalized = file.content.to_ascii_lowercase();
            normalized.contains("cognitive-complexity-threshold")
                && numeric_limit_at_most(&normalized, 8)
        })
        || snapshot.files.values().any(|file| {
            rust_source_path(&file.path)
                && file
                    .content
                    .to_ascii_lowercase()
                    .contains("clippy::cognitive_complexity")
        })
}

fn numeric_limit_at_most(content: &str, maximum: u32) -> bool {
    content
        .split(|ch: char| !ch.is_ascii_digit())
        .filter(|part| !part.is_empty())
        .filter_map(|part| part.parse::<u32>().ok())
        .any(|value| value <= maximum)
}

fn rust_source_path(path: &str) -> bool {
    path.ends_with(".rs")
}

fn ignored_project_path(path: &str) -> bool {
    path.starts_with("fixtures/")
        || path.starts_with("templates/")
        || path.contains("/fixtures/")
        || path.contains("/target/")
        || path.contains("/node_modules/")
}

#[cfg(test)]
mod tests {
    use super::*;
    use slophammer_scan::{RepoFile, Snapshot};
    use std::collections::BTreeMap;
    use std::path::PathBuf;

    #[test]
    fn missing_rust_checks_are_reported() {
        let snapshot = Snapshot {
            root: PathBuf::from("."),
            files: BTreeMap::from([
                (
                    "README.md".to_owned(),
                    RepoFile {
                        path: "README.md".to_owned(),
                        content: String::new(),
                    },
                ),
                (
                    "AGENTS.md".to_owned(),
                    RepoFile {
                        path: "AGENTS.md".to_owned(),
                        content: String::new(),
                    },
                ),
                (
                    "Cargo.toml".to_owned(),
                    RepoFile {
                        path: "Cargo.toml".to_owned(),
                        content: "[package]\nname = \"x\"\nrust-version = \"1.86\"\n".to_owned(),
                    },
                ),
            ]),
        };
        let findings = run_rules(&snapshot, &Config::default(), &[]);
        assert!(
            findings
                .iter()
                .any(|finding| finding.rule_id == rule_ids::RUST_CHECK_REQUIRED)
        );
    }
}
