use globset::{Glob, GlobSet, GlobSetBuilder};
use slophammer_config::Config;
use slophammer_scan::Snapshot;

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
        &slophammer_config::rust_targets(config),
        &slophammer_config::rust_exclude(config),
    )
}

pub fn dry_rust_files(snapshot: &Snapshot, config: &Config) -> Vec<String> {
    rust_files(
        snapshot,
        &slophammer_config::rust_dry_paths(config),
        &slophammer_config::rust_dry_exclude(config),
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
