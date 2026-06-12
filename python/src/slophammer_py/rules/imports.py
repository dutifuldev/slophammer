"""Absolute import discipline for production Python code.

Relative imports are spelled differently depending on where the importing
file sits, so they defeat grep, silently break on file moves, and read as
dot-counting at depth. Production imports must be absolute; the fix is
mechanical (``ruff check --select TID252 --fix``). Detection parses the
source, so commented or quoted import text is invisible. The scan reports
the first offense per file.
"""

from __future__ import annotations

import ast

from slophammer_py.core import Finding
from slophammer_py.repo import Snapshot
from slophammer_py.rules.definitions import Definition
from slophammer_py.rules.scope import conventional_path


def absolute_import_findings(definition: Definition, snapshot: Snapshot) -> list[Finding]:
    findings: list[Finding] = []
    for file in snapshot.files.values():
        if not file.path.endswith(".py") or conventional_path(file.path):
            continue
        line = first_relative_import_line(file.content)
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


def first_relative_import_line(content: str) -> int | None:
    try:
        tree = ast.parse(content)
    except SyntaxError:
        return None
    lines = [
        node.lineno
        for node in ast.walk(tree)
        if isinstance(node, ast.ImportFrom) and node.level > 0
    ]
    return min(lines) if lines else None
