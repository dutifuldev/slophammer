"""Check and dry entry points: scan, config, rules, baseline, output."""

from __future__ import annotations

import json
from dataclasses import dataclass

from .baseline import BaselineError, apply_baseline_check, debt_line, write_baseline
from .config import ConfigError, load_config
from .core import Report
from .dry import dry_findings, max_findings
from .report import write_json, write_sarif, write_text
from .rules import run_rules
from .scan import scan_repo


@dataclass(frozen=True)
class CommandResult:
    code: int
    stdout: str = ""
    stderr: str = ""


def check(
    root: str,
    output_format: str = "text",
    only_rule_ids: list[str] | None = None,
    baseline: str = "off",
) -> CommandResult:
    try:
        snapshot = scan_repo(root)
        config = load_config(snapshot)
        report = run_rules(snapshot, config, only_rule_ids)
        return finish_check(snapshot.root, report, output_format, baseline)
    except (ConfigError, BaselineError, OSError) as error:
        return CommandResult(code=2, stderr=f"check failed: {error}\n")


def finish_check(root: str, report: Report, output_format: str, baseline: str) -> CommandResult:
    extra = ""
    if baseline == "write":
        extra = write_baseline(root, report)
        report = apply_baseline_check(root, report)
    elif baseline == "check":
        report = apply_baseline_check(root, report)
    body = format_report(report, output_format)
    if baseline != "off" and output_format == "text":
        body += debt_line(report)
    return CommandResult(code=0 if report.ok else 1, stdout=extra + body)


def format_report(report: Report, output_format: str) -> str:
    if output_format == "json":
        return write_json(report)
    if output_format == "sarif":
        return write_sarif(report)
    return write_text(report)


def dry(root: str, output_format: str = "text") -> CommandResult:
    try:
        snapshot = scan_repo(root)
        config = load_config(snapshot)
    except (ConfigError, OSError) as error:
        return CommandResult(code=2, stderr=f"dry failed: {error}\n")
    findings = dry_findings(snapshot, config)
    allowed = max_findings(config)
    code = 0 if len(findings) <= allowed else 1
    if output_format == "json":
        body = (
            json.dumps(
                {
                    "ok": code == 0,
                    "findings": [finding.json_value() for finding in findings],
                },
                indent=2,
            )
            + "\n"
        )
    else:
        body = f"DRY candidates: {len(findings)}; maximum: {allowed}\n"
    return CommandResult(code=code, stdout=body)
