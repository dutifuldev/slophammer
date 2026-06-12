"""Shared finding and report types matching specs/REPORT_FORMAT.md."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Literal

Severity = Literal["error", "warn"]


@dataclass(frozen=True)
class Finding:
    rule_id: str
    severity: Severity
    path: str
    message: str
    baselined: bool | None = None


@dataclass(frozen=True)
class ScopeCoverage:
    scanned: int
    production_files: int


@dataclass(frozen=True)
class Report:
    ok: bool
    findings: list[Finding] = field(default_factory=list)
    scope: ScopeCoverage | None = None


def new_report(findings: list[Finding], scope: ScopeCoverage | None = None) -> Report:
    ordered = sorted(findings, key=lambda finding: (finding.rule_id, finding.path))
    return Report(ok=len(ordered) == 0, findings=ordered, scope=scope)
