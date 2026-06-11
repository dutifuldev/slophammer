use crate::core::{Finding, Report};
use serde::{Deserialize, Serialize};
use std::collections::BTreeSet;
use std::path::Path;
use thiserror::Error;

pub const BASELINE_FILE: &str = "slophammer-baseline.json";

#[derive(Clone, Copy, Debug, Default, Eq, PartialEq)]
pub enum BaselineMode {
    #[default]
    Off,
    Check,
    Write,
}

#[derive(Debug, Error)]
pub enum BaselineError {
    #[error("baseline file {BASELINE_FILE} is missing")]
    Missing,
    #[error("baseline parse failed: {0}")]
    Parse(String),
    #[error("baseline version must be 1")]
    Version,
    #[error("baseline contains resolved findings; rewrite it: {0}")]
    Stale(String),
    #[error("baseline write would grow the baseline; fix the new findings instead: {0}")]
    Superset(String),
    #[error("baseline write failed: {0}")]
    Write(String),
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct BaselineFile {
    version: u32,
    findings: Vec<BaselineEntry>,
}

#[derive(Clone, Deserialize, Eq, Ord, PartialEq, PartialOrd, Serialize)]
#[serde(deny_unknown_fields)]
struct BaselineEntry {
    rule_id: String,
    path: String,
}

/// Applies a checked-in baseline to a report: matched findings are marked
/// baselined and stop affecting `ok`; stale entries are an error so the
/// ratchet can only shrink. Matching is on rule_id plus path, never message.
pub fn apply_check(root: &Path, report: &mut Report) -> Result<(), BaselineError> {
    let baseline = read_baseline(root)?;
    let mut unmatched = baseline.clone();
    for finding in &mut report.findings {
        if baseline.contains(&entry_of(finding)) {
            finding.baselined = Some(true);
            unmatched.remove(&entry_of(finding));
        }
    }
    if !unmatched.is_empty() {
        return Err(BaselineError::Stale(joined(&unmatched)));
    }
    report.ok = report
        .findings
        .iter()
        .all(|finding| finding.baselined == Some(true));
    Ok(())
}

/// Records current findings as the baseline. Refuses to write a superset of
/// an existing baseline and prints the added and removed entries, so debt is
/// recorded once, reviewed, and only ever reduced.
pub fn write(root: &Path, report: &Report) -> Result<String, BaselineError> {
    let current: BTreeSet<BaselineEntry> = report.findings.iter().map(entry_of).collect();
    let previous = read_baseline(root).unwrap_or_default();
    let added: BTreeSet<_> = current.difference(&previous).cloned().collect();
    let removed: BTreeSet<_> = previous.difference(&current).cloned().collect();
    if !previous.is_empty() && !added.is_empty() && removed.is_empty() {
        return Err(BaselineError::Superset(joined(&added)));
    }
    let file = BaselineFile {
        version: 1,
        findings: current.iter().cloned().collect(),
    };
    let serialized = serde_json::to_string_pretty(&file)
        .map_err(|error| BaselineError::Write(error.to_string()))?;
    std::fs::write(root.join(BASELINE_FILE), format!("{serialized}\n"))
        .map_err(|error| BaselineError::Write(error.to_string()))?;
    Ok(write_summary(current.len(), &added, &removed))
}

pub fn debt_line(report: &Report) -> String {
    let baselined = report
        .findings
        .iter()
        .filter(|finding| finding.baselined == Some(true))
        .count();
    let new = report.findings.len() - baselined;
    format!("{baselined} findings baselined; {new} new\n")
}

fn read_baseline(root: &Path) -> Result<BTreeSet<BaselineEntry>, BaselineError> {
    let content =
        std::fs::read_to_string(root.join(BASELINE_FILE)).map_err(|_| BaselineError::Missing)?;
    let file: BaselineFile =
        serde_json::from_str(&content).map_err(|error| BaselineError::Parse(error.to_string()))?;
    if file.version != 1 {
        return Err(BaselineError::Version);
    }
    Ok(file.findings.into_iter().collect())
}

fn entry_of(finding: &Finding) -> BaselineEntry {
    BaselineEntry {
        rule_id: finding.rule_id.clone(),
        path: finding.path.clone(),
    }
}

fn joined(entries: &BTreeSet<BaselineEntry>) -> String {
    entries
        .iter()
        .map(|entry| format!("{} at {}", entry.rule_id, entry.path))
        .collect::<Vec<_>>()
        .join(", ")
}

fn write_summary(
    total: usize,
    added: &BTreeSet<BaselineEntry>,
    removed: &BTreeSet<BaselineEntry>,
) -> String {
    let mut summary = format!("baseline written: {total} finding(s)\n");
    for entry in added {
        summary.push_str(&format!("added: {} at {}\n", entry.rule_id, entry.path));
    }
    for entry in removed {
        summary.push_str(&format!("removed: {} at {}\n", entry.rule_id, entry.path));
    }
    summary
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::core::Severity;
    use crate::report::new_report;

    fn finding(rule_id: &str, path: &str) -> Finding {
        Finding {
            rule_id: rule_id.to_owned(),
            severity: Severity::Error,
            path: path.to_owned(),
            message: "missing".to_owned(),
            baselined: None,
        }
    }

    fn write_baseline(root: &Path, findings: &[(&str, &str)]) {
        let entries = findings
            .iter()
            .map(|(rule_id, path)| BaselineEntry {
                rule_id: (*rule_id).to_owned(),
                path: (*path).to_owned(),
            })
            .collect();
        let file = BaselineFile {
            version: 1,
            findings: entries,
        };
        std::fs::write(
            root.join(BASELINE_FILE),
            serde_json::to_string(&file).expect("serialize"),
        )
        .expect("write baseline");
    }

    #[test]
    fn baselined_findings_stop_affecting_ok() {
        let dir = tempfile::tempdir().expect("tempdir");
        write_baseline(dir.path(), &[("repo.readme-required", "README.md")]);
        let mut report = new_report(vec![finding("repo.readme-required", "README.md")]);

        apply_check(dir.path(), &mut report).expect("apply baseline");

        assert!(report.ok);
        assert_eq!(report.findings[0].baselined, Some(true));
    }

    #[test]
    fn new_findings_keep_failing() {
        let dir = tempfile::tempdir().expect("tempdir");
        write_baseline(dir.path(), &[("repo.readme-required", "README.md")]);
        let mut report = new_report(vec![
            finding("repo.readme-required", "README.md"),
            finding("repo.agents-required", "AGENTS.md"),
        ]);

        apply_check(dir.path(), &mut report).expect("apply baseline");

        assert!(!report.ok);
        assert_eq!(debt_line(&report), "1 findings baselined; 1 new\n");
    }

    #[test]
    fn stale_entries_are_an_error() {
        let dir = tempfile::tempdir().expect("tempdir");
        write_baseline(dir.path(), &[("repo.readme-required", "README.md")]);
        let mut report = new_report(Vec::new());

        let error = apply_check(dir.path(), &mut report).expect_err("stale baseline");

        assert!(error.to_string().contains("resolved findings"));
    }

    #[test]
    fn write_refuses_supersets() {
        let dir = tempfile::tempdir().expect("tempdir");
        write_baseline(dir.path(), &[("repo.readme-required", "README.md")]);
        let report = new_report(vec![
            finding("repo.readme-required", "README.md"),
            finding("repo.agents-required", "AGENTS.md"),
        ]);

        let error = write(dir.path(), &report).expect_err("superset write");

        assert!(error.to_string().contains("grow the baseline"));
    }

    #[test]
    fn write_records_and_shrinks() {
        let dir = tempfile::tempdir().expect("tempdir");
        let first = new_report(vec![
            finding("repo.readme-required", "README.md"),
            finding("repo.agents-required", "AGENTS.md"),
        ]);
        let summary = write(dir.path(), &first).expect("initial write");
        assert!(summary.contains("baseline written: 2"));

        let second = new_report(vec![finding("repo.agents-required", "AGENTS.md")]);
        let summary = write(dir.path(), &second).expect("shrinking write");
        assert!(summary.contains("removed: repo.readme-required at README.md"));
    }
}
