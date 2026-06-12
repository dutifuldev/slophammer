"""Suppression discipline for production Python code.

``# noqa``, ``# type: ignore``, and ``# ty: ignore`` directives need a
stated reason: trailing ``-- reason`` text or a preceding explanatory
comment. Bare ``# type: ignore`` or ``# ty: ignore`` without an error code
is itself a finding. Directives only take effect inside comments, so the
scan tracks quote state and markers inside string literals stay invisible.
Test scope is exempt. The scan reports the first offense per file.
"""

from __future__ import annotations

import re

from ..core import Finding
from ..repo import Snapshot
from .definitions import Definition

NOQA = re.compile(r"#\s*noqa(?P<codes>:\s*[\w, ]+)?", re.IGNORECASE)
TYPE_IGNORE = re.compile(r"#\s*type:\s*ignore(?P<codes>\[[\w,\s-]+\])?")
TY_IGNORE = re.compile(r"#\s*ty:\s*ignore(?P<codes>\[[\w,\s-]+\])?")
REASON_TEXT = re.compile(r"--\s*\S")

EXEMPT_SEGMENTS = {"tests", "test", "fixtures", "templates", "testdata", "vendor", "scripts"}


def suppression_findings(definition: Definition, snapshot: Snapshot) -> list[Finding]:
    findings: list[Finding] = []
    for file in snapshot.files.values():
        if not production_python_path(file.path):
            continue
        line = bare_suppression_line(file.content)
        if line is not None:
            findings.append(
                Finding(
                    rule_id=definition.id,
                    severity=definition.severity,
                    path=file.path,
                    message=f"{definition.message} (line {line})",
                )
            )
    return findings


def production_python_path(path: str) -> bool:
    if not path.endswith(".py"):
        return False
    base = path.rsplit("/", 1)[-1]
    if base.startswith("test_") or base.endswith("_test.py") or base == "conftest.py":
        return False
    segments = set(path.split("/")[:-1])
    return not (segments & EXEMPT_SEGMENTS) and "migrations" not in segments


def bare_suppression_line(content: str) -> int | None:
    previous_is_comment = False
    for index, line in enumerate(content.split("\n")):
        comment = comment_text(line)
        if bare_suppression_directive(comment, previous_is_comment):
            return index + 1
        previous_is_comment = explanatory_comment(line)
    return None


def bare_suppression_directive(comment: str, previous_is_comment: bool) -> bool:
    for pattern, codes_required in ((TYPE_IGNORE, True), (TY_IGNORE, True), (NOQA, False)):
        match = pattern.search(comment)
        if match is None:
            continue
        if codes_required and match.group("codes") is None:
            return True
        rest = comment[match.end() :]
        return not previous_is_comment and not REASON_TEXT.search(rest)
    return False


def explanatory_comment(line: str) -> bool:
    stripped = line.lstrip()
    if not stripped.startswith("#"):
        return False
    return not any(pattern.search(stripped) for pattern in (NOQA, TYPE_IGNORE, TY_IGNORE))


def comment_text(line: str) -> str:
    """The line's comment, starting at the first # outside string literals."""
    quote = ""
    index = 0
    while index < len(line):
        character = line[index]
        if quote:
            if character == "\\":
                index += 2
                continue
            if character == quote:
                quote = ""
        elif character in {'"', "'"}:
            quote = character
        elif character == "#":
            return line[index:]
        index += 1
    return ""
