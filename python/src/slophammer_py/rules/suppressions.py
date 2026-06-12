"""Suppression discipline for production Python code.

Suppression directives (noqa, type-ignore, and ty-ignore comments) need a
stated reason: trailing ``-- reason`` text or a preceding explanatory
comment. A type-ignore or ty-ignore without an error code is itself a
finding. Comments are located with the stdlib tokenizer, so directive text
inside strings and docstrings stays invisible; files the tokenizer rejects
fall back to a quote-tracking line scan. Test scope is exempt. The scan
reports the first offense per file.
"""

from __future__ import annotations

import io
import re
import tokenize

from ..core import Finding
from ..repo import RepoFile, Snapshot
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
        line = bare_suppression_line(file)
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


def bare_suppression_line(file: RepoFile) -> int | None:
    comments = comment_lines(file.content)
    explanatory = {line for line, text in comments.items() if explanatory_comment(text)}
    for line in sorted(comments):
        if bare_suppression_directive(comments[line], line - 1 in explanatory):
            return line
    return None


# Comment text by line number, via the tokenizer when the file parses and a
# quote-tracking line scan when it does not.
def comment_lines(content: str) -> dict[int, str]:
    try:
        return {
            item.start[0]: item.string
            for item in tokenize.generate_tokens(io.StringIO(content).readline)
            if item.type == tokenize.COMMENT
        }
    except (tokenize.TokenError, SyntaxError, ValueError):
        return fallback_comment_lines(content)


def fallback_comment_lines(content: str) -> dict[int, str]:
    comments: dict[int, str] = {}
    for index, line in enumerate(content.split("\n")):
        text = comment_text(line)
        if text:
            comments[index + 1] = text
    return comments


def bare_suppression_directive(comment: str, has_explanation: bool) -> bool:
    for pattern, codes_required in ((TYPE_IGNORE, True), (TY_IGNORE, True), (NOQA, False)):
        match = pattern.search(comment)
        if match is None:
            continue
        if codes_required and match.group("codes") is None:
            return True
        rest = comment[match.end() :]
        return not has_explanation and not REASON_TEXT.search(rest)
    return False


def explanatory_comment(text: str) -> bool:
    return not any(pattern.search(text) for pattern in (NOQA, TYPE_IGNORE, TY_IGNORE))


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
