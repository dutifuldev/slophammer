use serde::Serialize;
use slophammer_core::{Finding, Report, Severity};
use thiserror::Error;

#[derive(Debug, Error)]
pub enum ReportError {
    #[error("json serialization failed: {0}")]
    Json(#[from] serde_json::Error),
}

pub fn new_report(mut findings: Vec<Finding>) -> Report {
    findings.sort_by(|left, right| {
        left.rule_id
            .cmp(&right.rule_id)
            .then_with(|| left.path.cmp(&right.path))
            .then_with(|| left.message.cmp(&right.message))
    });
    Report {
        ok: findings.is_empty(),
        findings,
    }
}

pub fn write_json(report: &Report) -> Result<String, ReportError> {
    Ok(format!("{}\n", serde_json::to_string_pretty(report)?))
}

pub fn write_text(report: &Report) -> String {
    if report.ok {
        return "OK: no findings\n".to_owned();
    }
    let mut output = String::new();
    for finding in &report.findings {
        output.push_str(&format!(
            "{} {} {}: {}\n",
            finding.severity, finding.rule_id, finding.path, finding.message
        ));
    }
    output.push_str(&format!("\n{} finding(s)\n", report.findings.len()));
    output
}

pub fn write_sarif(report: &Report) -> Result<String, ReportError> {
    Ok(format!(
        "{}\n",
        serde_json::to_string_pretty(&sarif_report(report))?
    ))
}

#[derive(Serialize)]
struct SarifLog {
    #[serde(rename = "$schema")]
    schema: &'static str,
    version: &'static str,
    runs: Vec<SarifRun>,
}

#[derive(Serialize)]
struct SarifRun {
    tool: SarifTool,
    results: Vec<SarifResult>,
}

#[derive(Serialize)]
struct SarifTool {
    driver: SarifDriver,
}

#[derive(Serialize)]
struct SarifDriver {
    name: &'static str,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    rules: Vec<SarifRule>,
}

#[derive(Serialize)]
struct SarifRule {
    id: String,
    #[serde(rename = "shortDescription")]
    short_description: SarifMessage,
}

#[derive(Serialize)]
struct SarifResult {
    #[serde(rename = "ruleId")]
    rule_id: String,
    level: &'static str,
    message: SarifMessage,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    locations: Vec<SarifLocation>,
}

#[derive(Serialize)]
struct SarifMessage {
    text: String,
}

#[derive(Serialize)]
struct SarifLocation {
    #[serde(rename = "physicalLocation")]
    physical_location: SarifPhysicalLocation,
}

#[derive(Serialize)]
struct SarifPhysicalLocation {
    #[serde(rename = "artifactLocation")]
    artifact_location: SarifArtifactLocation,
    region: SarifRegion,
}

#[derive(Serialize)]
struct SarifArtifactLocation {
    uri: String,
}

#[derive(Serialize)]
struct SarifRegion {
    #[serde(rename = "startLine")]
    start_line: u32,
}

fn sarif_report(report: &Report) -> SarifLog {
    SarifLog {
        schema: "https://json.schemastore.org/sarif-2.1.0.json",
        version: "2.1.0",
        runs: vec![SarifRun {
            tool: SarifTool {
                driver: SarifDriver {
                    name: "slophammer",
                    rules: sarif_rules(&report.findings),
                },
            },
            results: sarif_results(&report.findings),
        }],
    }
}

fn sarif_rules(findings: &[Finding]) -> Vec<SarifRule> {
    let mut seen = std::collections::BTreeSet::new();
    let mut rules = Vec::new();
    for finding in findings {
        if seen.insert(finding.rule_id.clone()) {
            rules.push(SarifRule {
                id: finding.rule_id.clone(),
                short_description: SarifMessage {
                    text: finding.message.clone(),
                },
            });
        }
    }
    rules
}

fn sarif_results(findings: &[Finding]) -> Vec<SarifResult> {
    findings
        .iter()
        .map(|finding| SarifResult {
            rule_id: finding.rule_id.clone(),
            level: sarif_level(finding.severity),
            message: SarifMessage {
                text: finding.message.clone(),
            },
            locations: sarif_locations(&finding.path),
        })
        .collect()
}

fn sarif_level(severity: Severity) -> &'static str {
    match severity {
        Severity::Warn => "warning",
        Severity::Error => "error",
    }
}

fn sarif_locations(path: &str) -> Vec<SarifLocation> {
    if path.is_empty() {
        return Vec::new();
    }
    vec![SarifLocation {
        physical_location: SarifPhysicalLocation {
            artifact_location: SarifArtifactLocation {
                uri: path.to_owned(),
            },
            region: SarifRegion { start_line: 1 },
        },
    }]
}

#[cfg(test)]
mod tests {
    use super::*;
    use slophammer_core::{Finding, Severity};

    #[test]
    fn sorts_findings() {
        let report = new_report(vec![
            Finding {
                rule_id: "rust.test-required".to_owned(),
                severity: Severity::Error,
                path: "b".to_owned(),
                message: "b".to_owned(),
            },
            Finding {
                rule_id: "repo.readme-required".to_owned(),
                severity: Severity::Error,
                path: "a".to_owned(),
                message: "a".to_owned(),
            },
        ]);
        assert_eq!(report.findings[0].rule_id, "repo.readme-required");
    }

    #[test]
    fn writes_text_reports() {
        assert_eq!(write_text(&new_report(Vec::new())), "OK: no findings\n");
        let report = new_report(vec![Finding {
            rule_id: "rust.check-required".to_owned(),
            severity: Severity::Error,
            path: ".github/workflows".to_owned(),
            message: "missing".to_owned(),
        }]);
        let text = write_text(&report);
        assert!(text.contains("error rust.check-required .github/workflows: missing"));
        assert!(text.contains("1 finding(s)"));
    }

    #[test]
    fn writes_json_reports() {
        let report = new_report(vec![Finding {
            rule_id: "rust.check-required".to_owned(),
            severity: Severity::Error,
            path: ".github/workflows".to_owned(),
            message: "missing".to_owned(),
        }]);
        let json = write_json(&report).expect("json report");
        assert!(json.contains("\"ok\": false"));
        assert!(json.contains("\"rule_id\": \"rust.check-required\""));
    }

    #[test]
    fn writes_sarif_reports() {
        let report = new_report(vec![
            Finding {
                rule_id: "rust.check-required".to_owned(),
                severity: Severity::Error,
                path: ".github/workflows".to_owned(),
                message: "missing".to_owned(),
            },
            Finding {
                rule_id: "rust.warn-example".to_owned(),
                severity: Severity::Warn,
                path: String::new(),
                message: "warning".to_owned(),
            },
        ]);
        let sarif = write_sarif(&report).expect("sarif report");
        assert!(sarif.contains("\"version\": \"2.1.0\""));
        assert!(sarif.contains("\"level\": \"warning\""));
        assert!(sarif.contains("\"uri\": \".github/workflows\""));
    }
}
