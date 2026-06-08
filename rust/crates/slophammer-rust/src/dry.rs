use crate::definitions::{definition, rule_ids};
use crate::scope;
use slophammer_config::Config;
use slophammer_core::Finding;
use slophammer_scan::Snapshot;
use std::collections::BTreeMap;

pub fn dry_findings(snapshot: &Snapshot, config: &Config, max_findings: usize) -> Vec<Finding> {
    let min_tokens = slophammer_config::rust_dry_min_tokens(config);
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
    let mut seen: BTreeMap<Vec<String>, String> = BTreeMap::new();
    let mut findings = Vec::new();
    for path in scope::dry_rust_files(snapshot, config) {
        let Some(file) = snapshot.file(&path) else {
            continue;
        };
        let tokens = tokens(&file.content);
        if tokens.len() < min_tokens {
            continue;
        }
        for window in tokens.windows(min_tokens) {
            let key = window.to_vec();
            if let Some(first_path) = seen.get(&key) {
                if first_path != &path {
                    findings.push(Finding::with_message(
                        definition(rule_ids::RUST_DRY_REQUIRED),
                        path.clone(),
                        format!("Rust production code duplicates a copied block from {first_path}"),
                    ));
                    break;
                }
            } else {
                seen.insert(key, path.clone());
            }
        }
    }
    findings
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

    #[test]
    fn tokenizes_rust_source() {
        assert_eq!(
            tokens("fn demo_value() -> usize { 1 }"),
            vec!["fn", "demo_value", "usize", "1"]
        );
    }
}
