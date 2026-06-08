use crate::definitions::{definition, rule_ids};
use crate::scope;
use slophammer_config::{Config, UnsafePolicy};
use slophammer_core::Finding;
use slophammer_scan::Snapshot;
use syn::visit::{self, Visit};

pub fn policy_findings(snapshot: &Snapshot, config: &Config) -> Vec<Finding> {
    if !crate::is_rust_project(snapshot) {
        return Vec::new();
    }
    let Some(rust) = &config.rust else {
        return vec![Finding::new(definition(
            rule_ids::RUST_UNSAFE_POLICY_REQUIRED,
        ))];
    };
    let Some(policy) = &rust.unsafe_policy else {
        return vec![Finding::new(definition(
            rule_ids::RUST_UNSAFE_POLICY_REQUIRED,
        ))];
    };
    match policy.policy {
        UnsafePolicy::Forbid => forbid_findings(snapshot, config),
        UnsafePolicy::AllowDocumented => documented_findings(snapshot, config),
    }
}

fn forbid_findings(snapshot: &Snapshot, config: &Config) -> Vec<Finding> {
    unsafe_paths(snapshot, config)
        .into_iter()
        .filter(|path| !unsafe_allowed(config, path))
        .map(|path| Finding::at_path(definition(rule_ids::RUST_UNSAFE_POLICY_REQUIRED), path))
        .collect()
}

fn documented_findings(snapshot: &Snapshot, config: &Config) -> Vec<Finding> {
    unsafe_paths(snapshot, config)
        .into_iter()
        .filter(|path| !unsafe_allowed(config, path))
        .map(|path| {
            Finding::with_message(
                definition(rule_ids::RUST_UNSAFE_POLICY_REQUIRED),
                path,
                "Rust unsafe code must be documented with an allow entry",
            )
        })
        .collect()
}

fn unsafe_paths(snapshot: &Snapshot, config: &Config) -> Vec<String> {
    scope::configured_rust_files(snapshot, config)
        .into_iter()
        .filter(|path| {
            snapshot
                .file(path)
                .is_some_and(|file| file_contains_unsafe(&file.content))
        })
        .collect()
}

fn file_contains_unsafe(content: &str) -> bool {
    let Ok(parsed) = syn::parse_file(content) else {
        return content.contains("unsafe");
    };
    let mut visitor = UnsafeVisitor::default();
    visitor.visit_file(&parsed);
    visitor.found
}

fn unsafe_allowed(config: &Config, path: &str) -> bool {
    config
        .rust
        .as_ref()
        .and_then(|rust| rust.unsafe_policy.as_ref())
        .is_some_and(|policy| {
            policy.allow.iter().any(|allow| {
                !allow.reason.trim().is_empty()
                    && (path == allow.path || path.starts_with(&format!("{}/", allow.path)))
            })
        })
}

#[derive(Default)]
struct UnsafeVisitor {
    found: bool,
}

impl<'ast> Visit<'ast> for UnsafeVisitor {
    fn visit_expr_unsafe(&mut self, node: &'ast syn::ExprUnsafe) {
        self.found = true;
        visit::visit_expr_unsafe(self, node);
    }

    fn visit_item_fn(&mut self, node: &'ast syn::ItemFn) {
        if node.sig.unsafety.is_some() {
            self.found = true;
        }
        visit::visit_item_fn(self, node);
    }

    fn visit_item_trait(&mut self, node: &'ast syn::ItemTrait) {
        if node.unsafety.is_some() {
            self.found = true;
        }
        visit::visit_item_trait(self, node);
    }

    fn visit_item_impl(&mut self, node: &'ast syn::ItemImpl) {
        if node.unsafety.is_some() {
            self.found = true;
        }
        visit::visit_item_impl(self, node);
    }

    fn visit_item_foreign_mod(&mut self, node: &'ast syn::ItemForeignMod) {
        if node.unsafety.is_some() {
            self.found = true;
        }
        visit::visit_item_foreign_mod(self, node);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use slophammer_config::{RustConfig, RustUnsafe, UnsafeAllow};
    use slophammer_scan::{RepoFile, Snapshot};
    use std::collections::BTreeMap;
    use std::path::PathBuf;

    #[test]
    fn detects_unsafe_blocks() {
        assert!(file_contains_unsafe(
            "fn main() { unsafe { core::ptr::null::<u8>(); } }"
        ));
    }

    #[test]
    fn missing_policy_is_a_finding() {
        let findings = policy_findings(&unsafe_snapshot(), &Config::default());
        assert_eq!(findings.len(), 1);
        assert_eq!(findings[0].path, "slophammer.yml");
    }

    #[test]
    fn allow_entry_suppresses_forbidden_unsafe() {
        let findings = policy_findings(
            &unsafe_snapshot(),
            &Config {
                rust: Some(RustConfig {
                    unsafe_policy: Some(RustUnsafe {
                        policy: UnsafePolicy::Forbid,
                        allow: vec![UnsafeAllow {
                            path: "src/lib.rs".to_owned(),
                            reason: "ffi boundary".to_owned(),
                        }],
                    }),
                    targets: vec!["src".to_owned()],
                    ..RustConfig::default()
                }),
            },
        );
        assert!(findings.is_empty());
    }

    #[test]
    fn allow_documented_requires_allow_entry() {
        let findings = policy_findings(
            &unsafe_snapshot(),
            &Config {
                rust: Some(RustConfig {
                    unsafe_policy: Some(RustUnsafe {
                        policy: UnsafePolicy::AllowDocumented,
                        allow: Vec::new(),
                    }),
                    targets: vec!["src".to_owned()],
                    ..RustConfig::default()
                }),
            },
        );
        assert_eq!(findings.len(), 1);
        assert!(findings[0].message.contains("documented"));
    }

    fn unsafe_snapshot() -> Snapshot {
        Snapshot {
            root: PathBuf::from("."),
            files: BTreeMap::from([
                (
                    "Cargo.toml".to_owned(),
                    RepoFile {
                        path: "Cargo.toml".to_owned(),
                        content: "[package]\nname = \"demo\"\nrust-version = \"1.86\"\n".to_owned(),
                    },
                ),
                (
                    "src/lib.rs".to_owned(),
                    RepoFile {
                        path: "src/lib.rs".to_owned(),
                        content: "pub fn demo() { unsafe { core::ptr::null::<u8>(); } }".to_owned(),
                    },
                ),
            ]),
        }
    }
}
