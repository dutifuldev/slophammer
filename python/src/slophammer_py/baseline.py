"""Baseline ratchet per specs/BASELINE.md, ported from app/baseline.ts.

Matched findings are marked baselined and stop affecting ok; stale entries
are an error so the ratchet can only shrink. Matching is on rule_id plus
path, never message.
"""

from __future__ import annotations

import json
from dataclasses import replace
from pathlib import Path

from slophammer_py.core import Finding, Report

BASELINE_FILE_NAME = "slophammer-baseline.json"


class BaselineError(Exception):
    """Baseline failures exit with code 2."""


def apply_baseline_check(root: str, report: Report) -> Report:
    baseline = read_baseline(root)
    keys = {entry_key(entry) for entry in baseline}
    findings = [
        replace(finding, baselined=True) if entry_key(entry_of(finding)) in keys else finding
        for finding in report.findings
    ]
    matched = {entry_key(entry_of(finding)) for finding in findings if finding.baselined is True}
    stale = [entry for entry in baseline if entry_key(entry) not in matched]
    if stale:
        raise BaselineError(f"baseline contains resolved findings; rewrite it: {joined(stale)}")
    return replace(
        report,
        ok=all(finding.baselined is True for finding in findings),
        findings=findings,
    )


def write_baseline(root: str, report: Report) -> str:
    current = sorted_unique_entries([entry_of(finding) for finding in report.findings])
    previous = previous_baseline(root)
    previous_keys = {entry_key(entry) for entry in previous}
    current_keys = {entry_key(entry) for entry in current}
    added = [entry for entry in current if entry_key(entry) not in previous_keys]
    removed = [entry for entry in previous if entry_key(entry) not in current_keys]
    if previous and added and not removed:
        raise BaselineError(
            f"baseline write would grow the baseline; fix the new findings instead: {joined(added)}"
        )
    serialized = json.dumps(
        {"version": 1, "findings": [dict(entry) for entry in current]}, indent=2
    )
    (Path(root) / BASELINE_FILE_NAME).write_text(serialized + "\n", encoding="utf-8")
    return write_summary(len(current), added, removed)


def debt_line(report: Report) -> str:
    baselined = sum(1 for finding in report.findings if finding.baselined is True)
    fresh = len(report.findings) - baselined
    return f"{baselined} findings baselined; {fresh} new\n"


# Only a missing baseline file counts as the initial-write case; a present
# but malformed baseline propagates its parse error instead of being
# silently replaced.
def previous_baseline(root: str) -> list[dict[str, str]]:
    if not (Path(root) / BASELINE_FILE_NAME).exists():
        return []
    return read_baseline(root)


def read_baseline(root: str) -> list[dict[str, str]]:
    path = Path(root) / BASELINE_FILE_NAME
    try:
        content = path.read_text(encoding="utf-8")
    except OSError as error:
        raise BaselineError(f"baseline file {BASELINE_FILE_NAME} is missing") from error
    parsed = parse_baseline_root(content)
    if parsed.get("version") != 1:
        raise BaselineError("baseline version must be 1")
    return parse_baseline_entries(parsed.get("findings"))


def parse_baseline_root(content: str) -> dict[str, object]:
    try:
        parsed = json.loads(content)
    except ValueError as error:
        raise BaselineError(f"baseline parse failed: {error}") from error
    if not isinstance(parsed, dict) or not set(parsed) <= {"version", "findings"}:
        raise BaselineError(
            "baseline parse failed: baseline must be an object with version and findings"
        )
    return parsed


def parse_baseline_entries(value: object) -> list[dict[str, str]]:
    if not isinstance(value, list):
        raise BaselineError("baseline parse failed: findings must be a list")
    entries: list[dict[str, str]] = []
    for item in value:
        if not isinstance(item, dict) or not set(item) <= {"rule_id", "path"}:
            raise BaselineError(
                "baseline parse failed: findings entries need rule_id and path strings"
            )
        rule_id = item.get("rule_id")
        path = item.get("path")
        if not isinstance(rule_id, str) or not isinstance(path, str):
            raise BaselineError(
                "baseline parse failed: findings entries need rule_id and path strings"
            )
        entries.append({"rule_id": rule_id, "path": path})
    return entries


def entry_of(finding: Finding) -> dict[str, str]:
    return {"rule_id": finding.rule_id, "path": finding.path}


def entry_key(entry: dict[str, str]) -> str:
    return f"{entry['rule_id']}\x00{entry['path']}"


def sorted_unique_entries(entries: list[dict[str, str]]) -> list[dict[str, str]]:
    by_key = {entry_key(entry): entry for entry in entries}
    return sorted(by_key.values(), key=lambda entry: (entry["rule_id"], entry["path"]))


def joined(entries: list[dict[str, str]]) -> str:
    return ", ".join(
        f"{entry['rule_id']} at {entry['path']}" for entry in sorted_unique_entries(entries)
    )


def write_summary(total: int, added: list[dict[str, str]], removed: list[dict[str, str]]) -> str:
    lines = [f"baseline written: {total} finding(s)"]
    lines.extend(f"added: {entry['rule_id']} at {entry['path']}" for entry in added)
    lines.extend(f"removed: {entry['rule_id']} at {entry['path']}" for entry in removed)
    return "\n".join(lines) + "\n"
