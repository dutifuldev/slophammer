use super::definitions::{definition, rule_ids};
use crate::core::Finding;
use crate::scan::Snapshot;

/// A bare `#[allow(...)]` in production Rust code is a finding: suppressions
/// must carry an adjacent `// reason` comment or use `#[expect(...)]` with a
/// `reason` attribute. Detection is line-based; the scan stops at the first
/// `#[cfg(test)]` line since test modules sit at the end of a file by
/// convention and test scope is exempt.
pub fn suppression_findings(snapshot: &Snapshot) -> Vec<Finding> {
    let mut findings = Vec::new();
    for file in snapshot.files.values() {
        if !production_rust_path(&file.path) {
            continue;
        }
        if let Some(line) = bare_suppression_line(&file.content) {
            findings.push(suppression_finding(&file.path, line));
        }
    }
    findings
}

fn production_rust_path(path: &str) -> bool {
    path.ends_with(".rs")
        && !path.ends_with("_test.rs")
        && !path.starts_with("tests/")
        && !path.contains("/tests/")
        && !super::ignored_project_path(path)
}

fn bare_suppression_line(content: &str) -> Option<usize> {
    let mut previous_line_is_comment = false;
    for (index, line) in content.lines().enumerate() {
        let trimmed = line.trim_start();
        if trimmed.starts_with("#[cfg(test)") {
            return None;
        }
        if bare_allow_attribute(trimmed, previous_line_is_comment) {
            return Some(index + 1);
        }
        previous_line_is_comment = trimmed.starts_with("//");
    }
    None
}

fn bare_allow_attribute(trimmed: &str, previous_line_is_comment: bool) -> bool {
    if !trimmed.starts_with("#[allow(") && !trimmed.starts_with("#![allow(") {
        return false;
    }
    !previous_line_is_comment && !trimmed.contains("//") && !trimmed.contains("reason")
}

fn suppression_finding(path: &str, line: usize) -> Finding {
    let template = definition(rule_ids::RUST_SUPPRESSIONS_JUSTIFIED);
    Finding::with_message(
        template,
        path,
        format!("{} (line {line})", template.message),
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::scan::RepoFile;
    use std::collections::BTreeMap;
    use std::path::PathBuf;

    fn snapshot(path: &str, content: &str) -> Snapshot {
        Snapshot {
            root: PathBuf::from("/repo"),
            files: BTreeMap::from([(
                path.to_owned(),
                RepoFile {
                    path: path.to_owned(),
                    content: content.to_owned(),
                },
            )]),
        }
    }

    #[test]
    fn bare_allow_in_production_code_is_a_finding() {
        let findings = suppression_findings(&snapshot(
            "src/lib.rs",
            "#[allow(dead_code)]\nfn demo() {}\n",
        ));
        assert_eq!(findings.len(), 1);
        assert_eq!(findings[0].path, "src/lib.rs");
        assert!(findings[0].message.contains("line 1"));
    }

    #[test]
    fn justified_and_expect_suppressions_pass() {
        let content = "#[allow(dead_code)] // kept for ffi consumers\n\
                       // exercised through the public api only\n\
                       #[allow(unused)]\n\
                       #[expect(dead_code, reason = \"used behind feature flag\")]\n\
                       fn demo() {}\n";
        assert!(suppression_findings(&snapshot("src/lib.rs", content)).is_empty());
    }

    #[test]
    fn test_scope_is_exempt() {
        assert!(
            suppression_findings(&snapshot("tests/cli.rs", "#[allow(dead_code)]\n")).is_empty()
        );
        let trailing = "fn demo() {}\n#[cfg(test)]\nmod tests {\n    #[allow(dead_code)]\n    fn helper() {}\n}\n";
        assert!(suppression_findings(&snapshot("src/lib.rs", trailing)).is_empty());
    }
}
