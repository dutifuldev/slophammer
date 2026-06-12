"""Scope completeness: configured scope must account for every production
Python file, so narrowing scope cannot hide code from checking. Coverage
paths, DRY paths, and mutation targets all participate in the scope union.
"""

from __future__ import annotations

import fnmatch

from ..config import Config, ExcludeEntry, PythonConfig
from ..core import Finding, ScopeCoverage
from ..repo import Snapshot
from .definitions import Definition

CONVENTIONAL_DIRS = {
    "tests",
    "test",
    "fixtures",
    "templates",
    "testdata",
    "dist",
    "build",
    "coverage",
    "target",
    "node_modules",
    "vendor",
    "scripts",
    "migrations",
}


def scope_findings(definition: Definition, snapshot: Snapshot, config: Config) -> list[Finding]:
    scopes = configured_scopes(config.python)
    if not scopes or config.python is None:
        return []
    uncovered = uncovered_production_dirs(snapshot, config.python, scopes)
    if not uncovered:
        return []
    return [
        Finding(
            rule_id=definition.id,
            severity=definition.severity,
            path=definition.path,
            message=f"{definition.message}: {', '.join(uncovered)}",
        )
    ]


def scope_counts(snapshot: Snapshot, config: Config) -> ScopeCoverage | None:
    scopes = configured_scopes(config.python)
    if not scopes:
        return None
    production = production_python_files(snapshot)
    scanned = sum(1 for path in production if in_targets(path, scopes))
    return ScopeCoverage(scanned=scanned, production_files=len(production))


def configured_scopes(python: PythonConfig | None) -> list[str]:
    if python is None:
        return []
    scopes: list[str] = []
    if python.coverage is not None:
        scopes.extend(python.coverage.paths)
    if python.dry is not None:
        scopes.extend(python.dry.paths)
    if python.mutation is not None:
        scopes.extend(python.mutation.targets)
    return scopes


def uncovered_production_dirs(
    snapshot: Snapshot, python: PythonConfig, scopes: list[str]
) -> list[str]:
    patterns = all_exclude_patterns(python)
    directories = {
        parent_dir(path)
        for path in production_python_files(snapshot)
        if not in_targets(path, scopes) and not excluded(path, patterns)
    }
    return sorted(directories)


def all_exclude_patterns(python: PythonConfig) -> list[str]:
    sections: list[list[ExcludeEntry]] = []
    if python.coverage is not None:
        sections.append(python.coverage.exclude)
    if python.dry is not None:
        sections.append(python.dry.exclude)
    if python.mutation is not None:
        sections.append(python.mutation.exclude)
    return [entry.pattern for entries in sections for entry in entries]


def production_python_files(snapshot: Snapshot) -> list[str]:
    return [path for path in snapshot.files if path.endswith(".py") and not conventional_path(path)]


def conventional_path(path: str) -> bool:
    base = path.rsplit("/", 1)[-1]
    if base.startswith("test_") or base.endswith("_test.py") or base == "conftest.py":
        return True
    if "generated" in path:
        return True
    return any(segment in CONVENTIONAL_DIRS for segment in path.split("/")[:-1])


def parent_dir(path: str) -> str:
    return path.rsplit("/", 1)[0] if "/" in path else "."


def in_targets(path: str, targets: list[str]) -> bool:
    if not targets:
        return True
    for target in targets:
        normalized = target.rstrip("/")
        if normalized in (".", "") or path == normalized or path.startswith(normalized + "/"):
            return True
    return False


def excluded(path: str, patterns: list[str]) -> bool:
    return any(matches_pattern(path, pattern) for pattern in patterns)


def matches_pattern(path: str, pattern: str) -> bool:
    if fnmatch.fnmatch(path, pattern):
        return True
    # `dir/**` covers everything under dir, including dir itself.
    if pattern.endswith("/**"):
        return path.startswith(pattern[:-3] + "/") or fnmatch.fnmatch(path, pattern[:-3])
    return False
