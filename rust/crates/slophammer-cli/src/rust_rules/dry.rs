use super::definitions::{definition, rule_ids};
use super::scope;
use crate::config::Config;
use crate::core::Finding;
use crate::scan::Snapshot;
use std::collections::BTreeMap;

pub fn dry_findings(snapshot: &Snapshot, config: &Config, max_findings: usize) -> Vec<Finding> {
    if !crate::config::rust_dry_copied_blocks_enabled(config) {
        return Vec::new();
    }
    let min_tokens = crate::config::rust_dry_min_tokens(config);
    let duplicates = copied_block_findings(snapshot, config, min_tokens);
    if duplicates.len() > max_findings {
        duplicates
    } else {
        Vec::new()
    }
}

fn copied_block_findings(snapshot: &Snapshot, config: &Config, min_tokens: usize) -> Vec<Finding> {
    if min_tokens == 0 {
        return Vec::new();
    }
    let mut seen: BTreeMap<Vec<String>, Vec<Occurrence>> = BTreeMap::new();
    let mut findings = Vec::new();
    for path in scope::dry_rust_files(snapshot, config) {
        let Some(file) = snapshot.file(&path) else {
            continue;
        };
        let tokens = tokens(&file.content);
        if tokens.len() < min_tokens {
            continue;
        }
        for (start, window) in tokens.windows(min_tokens).enumerate() {
            let key = window.to_vec();
            let occurrences = seen.entry(key).or_default();
            if let Some(first) = occurrences
                .iter()
                .find(|occurrence| duplicate_occurrence(occurrence, &path, start, min_tokens))
            {
                findings.push(Finding::with_message(
                    definition(rule_ids::RUST_DRY_REQUIRED),
                    path.clone(),
                    format!(
                        "Rust production code duplicates a copied block from {}",
                        first.path
                    ),
                ));
                break;
            }
            occurrences.push(Occurrence {
                path: path.clone(),
                start,
            });
        }
    }
    findings
}

#[derive(Clone, Debug, Eq, PartialEq)]
struct Occurrence {
    path: String,
    start: usize,
}

fn duplicate_occurrence(
    previous: &Occurrence,
    path: &str,
    start: usize,
    window_size: usize,
) -> bool {
    previous.path != path || non_overlapping(previous.start, start, window_size)
}

fn non_overlapping(left: usize, right: usize, window_size: usize) -> bool {
    left + window_size <= right || right + window_size <= left
}

fn tokens(content: &str) -> Vec<String> {
    content
        .split(|ch: char| !(ch.is_ascii_alphanumeric() || ch == '_'))
        .filter(|part| !part.is_empty())
        .map(str::to_ascii_lowercase)
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::{CopiedBlocks, RustConfig, RustDry};
    use crate::scan::{RepoFile, Snapshot};
    use std::collections::BTreeMap;
    use std::path::PathBuf;

    #[test]
    fn tokenizes_rust_source() {
        assert_eq!(
            tokens("fn demo_value() -> usize { 1 }"),
            vec!["fn", "demo_value", "usize", "1"]
        );
    }

    #[test]
    fn disabled_copied_blocks_suppresses_findings() {
        let findings = dry_findings(
            &duplicate_snapshot(),
            &Config {
                rust: Some(RustConfig {
                    dry: Some(RustDry {
                        max_findings: 0,
                        paths: vec!["src".to_owned()],
                        copied_blocks: Some(CopiedBlocks {
                            enabled: false,
                            min_tokens: 3,
                        }),
                        ..RustDry::default()
                    }),
                    ..RustConfig::default()
                }),
                ..Config::default()
            },
            0,
        );
        assert!(findings.is_empty());
    }

    #[test]
    fn detects_same_file_copied_blocks() {
        let findings = dry_findings(
            &Snapshot {
                root: PathBuf::from("."),
                files: BTreeMap::from([(
                    "src/lib.rs".to_owned(),
                    RepoFile {
                        path: "src/lib.rs".to_owned(),
                        content: [
                            "fn first() { let alpha = 1; let beta = 2; let gamma = alpha + beta; }",
                            "fn second() { let alpha = 1; let beta = 2; let gamma = alpha + beta; }",
                        ]
                        .join("\n"),
                    },
                )]),
            },
            &dry_config(6),
            0,
        );
        assert_eq!(findings.len(), 1);
        assert_eq!(findings[0].path, "src/lib.rs");
    }

    fn duplicate_snapshot() -> Snapshot {
        Snapshot {
            root: PathBuf::from("."),
            files: BTreeMap::from([
                (
                    "src/a.rs".to_owned(),
                    RepoFile {
                        path: "src/a.rs".to_owned(),
                        content: "fn copied() { let alpha = 1; let beta = 2; }".to_owned(),
                    },
                ),
                (
                    "src/b.rs".to_owned(),
                    RepoFile {
                        path: "src/b.rs".to_owned(),
                        content: "fn copied() { let alpha = 1; let beta = 2; }".to_owned(),
                    },
                ),
            ]),
        }
    }

    fn dry_config(min_tokens: usize) -> Config {
        Config {
            rust: Some(RustConfig {
                dry: Some(RustDry {
                    max_findings: 0,
                    paths: vec!["src".to_owned()],
                    copied_blocks: Some(CopiedBlocks {
                        enabled: true,
                        min_tokens,
                    }),
                    ..RustDry::default()
                }),
                ..RustConfig::default()
            }),
            ..Config::default()
        }
    }
}
