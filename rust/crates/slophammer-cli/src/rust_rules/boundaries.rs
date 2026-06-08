use super::definitions::{definition, rule_ids};
use crate::config::{Config, DependencyBoundary};
use crate::core::Finding;
use crate::scan::Snapshot;
use std::collections::BTreeMap;
use std::path::{Component, Path, PathBuf};
use toml_edit::{DocumentMut, Item};

pub fn boundary_findings(snapshot: &Snapshot, config: &Config) -> Vec<Finding> {
    if !super::is_rust_project(snapshot) {
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
    let workspace_paths = workspace_dependency_paths(snapshot, &boundary.from);
    dependency_paths(&document, &boundary.from, &workspace_paths)
        .into_iter()
        .filter_map(|dependency| resolve_dependency_path(&dependency.base, &dependency.path))
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

#[derive(Clone, Debug, Eq, PartialEq)]
struct DependencyPath {
    base: String,
    path: String,
}

fn dependency_paths(
    document: &DocumentMut,
    direct_base: &str,
    workspace_paths: &BTreeMap<String, DependencyPath>,
) -> Vec<DependencyPath> {
    ["dependencies", "dev-dependencies", "build-dependencies"]
        .into_iter()
        .flat_map(|section| {
            dependency_paths_in_item(document.get(section), direct_base, workspace_paths)
        })
        .collect()
}

fn dependency_paths_in_item(
    item: Option<&Item>,
    direct_base: &str,
    workspace_paths: &BTreeMap<String, DependencyPath>,
) -> Vec<DependencyPath> {
    let Some(table) = item.and_then(Item::as_table) else {
        return Vec::new();
    };
    table
        .iter()
        .filter_map(|(name, item)| dependency_path(name, item, direct_base, workspace_paths))
        .collect()
}

fn dependency_path(
    name: &str,
    item: &Item,
    direct_base: &str,
    workspace_paths: &BTreeMap<String, DependencyPath>,
) -> Option<DependencyPath> {
    if let Some(path) = direct_dependency_path(item) {
        return Some(DependencyPath {
            base: repo_dir(direct_base),
            path,
        });
    }
    if dependency_is_workspace(item) {
        return workspace_paths.get(name).cloned();
    }
    None
}

fn direct_dependency_path(item: &Item) -> Option<String> {
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

fn dependency_is_workspace(item: &Item) -> bool {
    if let Some(value) = item.as_value() {
        if let Some(inline) = value.as_inline_table() {
            return inline
                .get("workspace")
                .and_then(|workspace| workspace.as_bool())
                .unwrap_or(false);
        }
    }
    item.as_table()
        .and_then(|table| table.get("workspace"))
        .and_then(Item::as_bool)
        .unwrap_or(false)
}

fn workspace_dependency_paths(snapshot: &Snapshot, from: &str) -> BTreeMap<String, DependencyPath> {
    let Some((manifest_path, document)) = workspace_manifest(snapshot, from) else {
        return BTreeMap::new();
    };
    let base = manifest_dir(&manifest_path);
    let Some(dependencies) = document
        .get("workspace")
        .and_then(Item::as_table)
        .and_then(|workspace| workspace.get("dependencies"))
    else {
        return BTreeMap::new();
    };
    workspace_dependency_paths_in_item(Some(dependencies), &base)
}

fn workspace_manifest(snapshot: &Snapshot, from: &str) -> Option<(String, DocumentMut)> {
    ancestor_manifest_paths(from)
        .into_iter()
        .filter_map(|path| {
            let file = snapshot.file(&path)?;
            let document = file.content.parse::<DocumentMut>().ok()?;
            document.get("workspace")?;
            Some((path, document))
        })
        .next()
}

fn ancestor_manifest_paths(from: &str) -> Vec<String> {
    let mut paths = Vec::new();
    let mut current = repo_dir(from);
    loop {
        paths.push(manifest_path(&current));
        if current == "." {
            break;
        }
        current = current
            .rsplit_once('/')
            .map(|(parent, _)| parent.to_owned())
            .unwrap_or_else(|| ".".to_owned());
    }
    paths
}

fn manifest_dir(manifest_path: &str) -> String {
    manifest_path
        .strip_suffix("/Cargo.toml")
        .filter(|path| !path.is_empty())
        .map(str::to_owned)
        .unwrap_or_else(|| ".".to_owned())
}

fn workspace_dependency_paths_in_item(
    item: Option<&Item>,
    base: &str,
) -> BTreeMap<String, DependencyPath> {
    let Some(table) = item.and_then(Item::as_table) else {
        return BTreeMap::new();
    };
    table
        .iter()
        .filter_map(|(name, item)| {
            direct_dependency_path(item).map(|path| {
                (
                    name.to_owned(),
                    DependencyPath {
                        base: repo_dir(base),
                        path,
                    },
                )
            })
        })
        .collect()
}

fn manifest_path(from: &str) -> String {
    let dir = repo_dir(from);
    if dir == "." {
        "Cargo.toml".to_owned()
    } else {
        format!("{dir}/Cargo.toml")
    }
}

fn resolve_dependency_path(from: &str, dependency_path: &str) -> Option<String> {
    let base = repo_dir(from);
    let base = if base == "." { "" } else { base.as_str() };
    let path = normalize_path(Path::new(base).join(dependency_path));
    path.to_str().map(|path| match path.replace('\\', "/") {
        path if path.is_empty() => ".".to_owned(),
        path => path,
    })
}

fn repo_dir(path: &str) -> String {
    let mut trimmed = path.trim().trim_end_matches('/');
    while let Some(path) = trimmed.strip_prefix("./") {
        trimmed = path;
    }
    if trimmed.is_empty() || trimmed == "." {
        ".".to_owned()
    } else {
        trimmed.to_owned()
    }
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
    snapshot.files.contains_key(&manifest_path(path))
}

fn allowed(boundary: &DependencyBoundary, path: &str) -> bool {
    boundary
        .allow
        .iter()
        .any(|allowed| path == allowed || path.starts_with(&format!("{allowed}/")))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::{Config, RustConfig};
    use crate::scan::{RepoFile, Snapshot};
    use std::collections::BTreeMap;
    use std::path::PathBuf;

    #[test]
    fn resolves_parent_paths() {
        assert_eq!(
            resolve_dependency_path("rust/crates/a", "../b").as_deref(),
            Some("rust/crates/b")
        );
    }

    #[test]
    fn inherited_workspace_path_dependencies_are_checked() {
        let findings = boundary_findings(
            &workspace_dependency_snapshot(),
            &Config {
                rust: Some(RustConfig {
                    dependency_boundaries: vec![DependencyBoundary {
                        from: "crates/a".to_owned(),
                        allow: Vec::new(),
                    }],
                    ..RustConfig::default()
                }),
                ..Config::default()
            },
        );
        assert_eq!(findings.len(), 1);
        assert!(findings[0].message.contains("crates/b"));
    }

    #[test]
    fn allowed_inherited_workspace_path_dependencies_pass() {
        let findings = boundary_findings(
            &workspace_dependency_snapshot(),
            &Config {
                rust: Some(RustConfig {
                    dependency_boundaries: vec![DependencyBoundary {
                        from: "crates/a".to_owned(),
                        allow: vec!["crates/b".to_owned()],
                    }],
                    ..RustConfig::default()
                }),
                ..Config::default()
            },
        );
        assert!(findings.is_empty());
    }

    fn workspace_dependency_snapshot() -> Snapshot {
        Snapshot {
            root: PathBuf::from("."),
            files: BTreeMap::from([
                (
                    "Cargo.toml".to_owned(),
                    RepoFile {
                        path: "Cargo.toml".to_owned(),
                        content: r#"
[workspace]
members = ["crates/a", "crates/b"]

[workspace.dependencies]
b = { path = "crates/b" }
"#
                        .to_owned(),
                    },
                ),
                (
                    "crates/a/Cargo.toml".to_owned(),
                    RepoFile {
                        path: "crates/a/Cargo.toml".to_owned(),
                        content: r#"
[package]
name = "a"
rust-version = "1.86"

[dependencies]
b.workspace = true
"#
                        .to_owned(),
                    },
                ),
                (
                    "crates/b/Cargo.toml".to_owned(),
                    RepoFile {
                        path: "crates/b/Cargo.toml".to_owned(),
                        content: r#"
[package]
name = "b"
rust-version = "1.86"
"#
                        .to_owned(),
                    },
                ),
            ]),
        }
    }
}
