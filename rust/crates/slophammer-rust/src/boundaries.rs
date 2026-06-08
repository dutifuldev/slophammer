use crate::definitions::{definition, rule_ids};
use slophammer_config::{Config, DependencyBoundary};
use slophammer_core::Finding;
use slophammer_scan::Snapshot;
use std::path::{Component, Path, PathBuf};
use toml_edit::{DocumentMut, Item};

pub fn boundary_findings(snapshot: &Snapshot, config: &Config) -> Vec<Finding> {
    if !crate::is_rust_project(snapshot) {
        return Vec::new();
    }
    let Some(rust) = &config.rust else {
        return vec![Finding::new(definition(
            rule_ids::RUST_DEPENDENCY_BOUNDARIES_REQUIRED,
        ))];
    };
    if rust.dependency_boundaries.is_empty() {
        return vec![Finding::new(definition(
            rule_ids::RUST_DEPENDENCY_BOUNDARIES_REQUIRED,
        ))];
    }
    rust.dependency_boundaries
        .iter()
        .flat_map(|boundary| check_boundary(snapshot, boundary))
        .collect()
}

fn check_boundary(snapshot: &Snapshot, boundary: &DependencyBoundary) -> Vec<Finding> {
    let manifest_path = manifest_path(&boundary.from);
    let Some(file) = snapshot.file(&manifest_path) else {
        return vec![Finding::at_path(
            definition(rule_ids::RUST_DEPENDENCY_BOUNDARIES_REQUIRED),
            manifest_path,
        )];
    };
    let Ok(document) = file.content.parse::<DocumentMut>() else {
        return vec![Finding::at_path(
            definition(rule_ids::RUST_DEPENDENCY_BOUNDARIES_REQUIRED),
            file.path.clone(),
        )];
    };
    dependency_paths(&document)
        .into_iter()
        .filter_map(|relative| resolve_dependency_path(&boundary.from, &relative))
        .filter(|path| local_repo_path(snapshot, path))
        .filter(|path| !allowed(boundary, path))
        .map(|path| {
            Finding::with_message(
                definition(rule_ids::RUST_DEPENDENCY_BOUNDARIES_REQUIRED),
                file.path.clone(),
                format!("Rust dependency boundary forbids dependency on {path}"),
            )
        })
        .collect()
}

fn dependency_paths(document: &DocumentMut) -> Vec<String> {
    ["dependencies", "dev-dependencies", "build-dependencies"]
        .into_iter()
        .flat_map(|section| dependency_paths_in_item(document.get(section)))
        .collect()
}

fn dependency_paths_in_item(item: Option<&Item>) -> Vec<String> {
    let Some(table) = item.and_then(Item::as_table) else {
        return Vec::new();
    };
    table
        .iter()
        .filter_map(|(_, item)| dependency_path(item))
        .collect()
}

fn dependency_path(item: &Item) -> Option<String> {
    if let Some(value) = item.as_value() {
        if let Some(inline) = value.as_inline_table() {
            return inline
                .get("path")
                .and_then(|path| path.as_str())
                .map(str::to_owned);
        }
    }
    item.as_table()
        .and_then(|table| table.get("path"))
        .and_then(Item::as_str)
        .map(str::to_owned)
}

fn manifest_path(from: &str) -> String {
    let trimmed = from.trim_end_matches('/');
    if trimmed.is_empty() || trimmed == "." {
        "Cargo.toml".to_owned()
    } else {
        format!("{trimmed}/Cargo.toml")
    }
}

fn resolve_dependency_path(from: &str, dependency_path: &str) -> Option<String> {
    let base = Path::new(from);
    let path = normalize_path(base.join(dependency_path));
    path.to_str().map(|path| path.replace('\\', "/"))
}

fn normalize_path(path: PathBuf) -> PathBuf {
    let mut normalized = PathBuf::new();
    for component in path.components() {
        match component {
            Component::ParentDir => {
                normalized.pop();
            }
            Component::CurDir => {}
            other => normalized.push(other.as_os_str()),
        }
    }
    normalized
}

fn local_repo_path(snapshot: &Snapshot, path: &str) -> bool {
    snapshot
        .files
        .contains_key(&format!("{}/Cargo.toml", path.trim_end_matches('/')))
}

fn allowed(boundary: &DependencyBoundary, path: &str) -> bool {
    boundary
        .allow
        .iter()
        .any(|allowed| path == allowed || path.starts_with(&format!("{}/", allowed)))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn resolves_parent_paths() {
        assert_eq!(
            resolve_dependency_path("rust/crates/a", "../b").as_deref(),
            Some("rust/crates/b")
        );
    }
}
