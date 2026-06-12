"""Report serialization: JSON, text, and SARIF, per specs/REPORT_FORMAT.md."""

from __future__ import annotations

import json

from .core import Finding, Report, Severity


def report_json_value(report: Report) -> dict[str, object]:
    value: dict[str, object] = {
        "ok": report.ok,
        "findings": [finding_json_value(finding) for finding in report.findings],
    }
    if report.scope is not None:
        value["scope"] = {
            "scanned": report.scope.scanned,
            "production_files": report.scope.production_files,
        }
    return value


def finding_json_value(finding: Finding) -> dict[str, object]:
    value: dict[str, object] = {
        "rule_id": finding.rule_id,
        "severity": finding.severity,
        "path": finding.path,
        "message": finding.message,
    }
    if finding.baselined is True:
        value["baselined"] = True
    return value


def write_json(report: Report) -> str:
    return json.dumps(report_json_value(report), indent=2) + "\n"


def write_text(report: Report) -> str:
    return report_body(report) + scope_line(report)


def report_body(report: Report) -> str:
    if report.ok:
        return "OK: no findings\n"
    lines = [
        f"{finding.severity} {finding.rule_id} {finding.path}: {finding.message}"
        for finding in report.findings
    ]
    return "\n".join(lines) + f"\n\n{len(report.findings)} finding(s)\n"


def scope_line(report: Report) -> str:
    if report.scope is None:
        return ""
    scanned, total = report.scope.scanned, report.scope.production_files
    return f"scope: scanned {scanned} of {total} production files\n"


def write_sarif(report: Report) -> str:
    return json.dumps(sarif_report(report), indent=2) + "\n"


def sarif_report(report: Report) -> dict[str, object]:
    return {
        "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
        "version": "2.1.0",
        "runs": [
            {
                "tool": {"driver": {"name": "slophammer", "rules": sarif_rules(report.findings)}},
                "results": sarif_results(report.findings),
            }
        ],
    }


def sarif_rules(findings: list[Finding]) -> list[dict[str, object]]:
    seen: set[str] = set()
    rules: list[dict[str, object]] = []
    for finding in findings:
        if finding.rule_id not in seen:
            seen.add(finding.rule_id)
            rules.append({"id": finding.rule_id, "shortDescription": {"text": finding.message}})
    return rules


def sarif_results(findings: list[Finding]) -> list[dict[str, object]]:
    results: list[dict[str, object]] = []
    for finding in findings:
        result: dict[str, object] = {
            "ruleId": finding.rule_id,
            "level": sarif_level(finding.severity),
            "message": {"text": finding.message},
        }
        locations = sarif_locations(finding.path)
        if locations is not None:
            result["locations"] = locations
        if finding.baselined is True:
            result["suppressions"] = [{"kind": "external"}]
        results.append(result)
    return results


def sarif_level(severity: Severity) -> str:
    return "warning" if severity == "warn" else "error"


def sarif_locations(file_path: str) -> list[dict[str, object]] | None:
    if file_path == "":
        return None
    return [
        {
            "physicalLocation": {
                "artifactLocation": {"uri": file_path},
                "region": {"startLine": 1},
            }
        }
    ]
