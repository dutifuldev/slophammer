use super::definitions::{definition, rule_ids};
use crate::config::{Config, RustConfig};
use crate::core::Finding;
use crate::scan::Snapshot;
use globset::{Glob, GlobSet, GlobSetBuilder};

/// Configured scope must account for every production Rust file: each one is
/// either inside a configured paths/targets scope or covered by an exclude
/// (conventional or reasoned). Anything else is a finding, so narrowing scope
/// cannot hide code from checking.
pub fn scope_findings(snapshot: &Snapshot, config: &Config) -> Vec<Finding> {
    let Some(rust) = &config.rust else {
        return Vec::new();
    };
    let scopes = configured_scopes(rust);
    if scopes.is_empty() {
        return Vec::new();
    }
    let uncovered = uncovered_production_dirs(snapshot, config, &scopes);
    if uncovered.is_empty() {
        return Vec::new();
    }
    let template = definition(rule_ids::RUST_SCOPE_INCOMPLETE);
    vec![Finding::with_message(
        template,
        template.path,
        format!("{}: {}", template.message, uncovered.join(", ")),
    )]
}

/// Scope coverage counts for the report: production files the configured
/// scope covers versus all production files. None when no scope is
/// configured.
pub fn scope_counts(snapshot: &Snapshot, config: &Config) -> Option<(usize, usize)> {
    let rust = config.rust.as_ref()?;
    let scopes = configured_scopes(rust);
    if scopes.is_empty() {
        return None;
    }
    let production = production_rust_files(snapshot);
    let scanned = production
        .iter()
        .filter(|path| in_targets(path, &scopes))
        .count();
    Some((scanned, production.len()))
}

/// Mutation targets participate so narrowing mutation scope stays visible.
fn configured_scopes(rust: &RustConfig) -> Vec<String> {
    let mut scopes = rust.targets.clone();
    if let Some(dry) = &rust.dry {
        scopes.extend(dry.paths.iter().cloned());
    }
    if let Some(coverage) = &rust.coverage {
        scopes.extend(coverage.paths.iter().cloned());
    }
    if let Some(mutation) = &rust.mutation {
        scopes.extend(mutation.targets.iter().cloned());
    }
    scopes
}

fn uncovered_production_dirs(
    snapshot: &Snapshot,
    config: &Config,
    scopes: &[String],
) -> Vec<String> {
    let exclude_set = globset(&all_exclude_patterns(config));
    let mut dirs: Vec<String> = production_rust_files(snapshot)
        .into_iter()
        .filter(|path| !in_targets(path, scopes))
        .filter(|path| {
            exclude_set
                .as_ref()
                .is_none_or(|set| !set.is_match(path.as_str()))
        })
        .map(|path| parent_dir(&path))
        .collect();
    dirs.sort();
    dirs.dedup();
    dirs
}

fn all_exclude_patterns(config: &Config) -> Vec<String> {
    let mut patterns = crate::config::rust_exclude(config);
    patterns.extend(crate::config::rust_dry_exclude(config));
    if let Some(coverage) = config.rust.as_ref().and_then(|rust| rust.coverage.as_ref()) {
        patterns.extend(crate::config::exclude_patterns(&coverage.exclude));
    }
    patterns
}

fn production_rust_files(snapshot: &Snapshot) -> Vec<String> {
    snapshot
        .files
        .keys()
        .filter(|path| path.ends_with(".rs"))
        .filter(|path| !conventional_path(path))
        .cloned()
        .collect()
}

/// Path-level form of the conventional non-production list in
/// specs/CONFIG.md.
fn conventional_path(path: &str) -> bool {
    const DIRS: [&str; 12] = [
        "tests",
        "fixtures",
        "templates",
        "testdata",
        "dist",
        "build",
        "coverage",
        "target",
        "node_modules",
        "vendor",
        "scripts",
        "benches",
    ];
    if path.ends_with("_test.rs") || path.contains("generated") {
        return true;
    }
    path.split('/').any(|segment| DIRS.contains(&segment))
}

fn parent_dir(path: &str) -> String {
    match path.rsplit_once('/') {
        Some((dir, _)) => dir.to_owned(),
        None => ".".to_owned(),
    }
}

pub fn rust_files(snapshot: &Snapshot, paths: &[String], excludes: &[String]) -> Vec<String> {
    let exclude_set = globset(excludes);
    snapshot
        .files
        .keys()
        .filter(|path| path.ends_with(".rs"))
        .filter(|path| in_targets(path, paths))
        .filter(|path| !default_excluded(path))
        .filter(|path| {
            exclude_set
                .as_ref()
                .is_none_or(|set| !set.is_match(path.as_str()))
        })
        .cloned()
        .collect()
}

pub fn configured_rust_files(snapshot: &Snapshot, config: &Config) -> Vec<String> {
    rust_files(
        snapshot,
        &crate::config::rust_targets(config),
        &crate::config::rust_exclude(config),
    )
}

pub fn dry_rust_files(snapshot: &Snapshot, config: &Config) -> Vec<String> {
    rust_files(
        snapshot,
        &crate::config::rust_dry_paths(config),
        &crate::config::rust_dry_exclude(config),
    )
}

fn in_targets(path: &str, targets: &[String]) -> bool {
    targets.is_empty()
        || targets.iter().any(|target| {
            let normalized = target.trim_end_matches('/');
            normalized == "." || path == normalized || path.starts_with(&format!("{normalized}/"))
        })
}

fn default_excluded(path: &str) -> bool {
    path.starts_with("fixtures/")
        || path.starts_with("templates/")
        || path.contains("/fixtures/")
        || path.contains("/target/")
        || path.contains("/node_modules/")
        || path.ends_with("_test.rs")
}

fn globset(patterns: &[String]) -> Option<GlobSet> {
    if patterns.is_empty() {
        return None;
    }
    let mut builder = GlobSetBuilder::new();
    for pattern in patterns {
        if let Ok(glob) = Glob::new(pattern) {
            builder.add(glob);
        }
    }
    builder.build().ok()
}
