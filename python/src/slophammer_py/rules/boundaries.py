"""Dependency boundary enforcement over parsed imports.

Config-gated: when ``python.dependency_boundaries`` is set, imports from
files under a boundary's ``from`` path must resolve inside that path or one
of its ``allow`` paths. Only imports that resolve to repository paths are
judged; third-party and standard-library imports stay out of scope.
"""

from __future__ import annotations

import ast

from ..config import Config, DependencyBoundary
from ..core import Finding
from ..repo import RepoFile, Snapshot
from .definitions import Definition


def boundary_findings(definition: Definition, snapshot: Snapshot, config: Config) -> list[Finding]:
    boundaries = config.python.dependency_boundaries if config.python is not None else []
    if not boundaries:
        return []
    roots = source_roots(snapshot, boundaries)
    findings: list[Finding] = []
    for boundary in boundaries:
        for file in python_files_under(snapshot, boundary.source):
            target = first_violation(snapshot, file.path, file.content, boundary, roots)
            if target is not None:
                findings.append(
                    Finding(
                        rule_id=definition.id,
                        severity=definition.severity,
                        path=file.path,
                        message=f"{definition.message}: imports {target}",
                    )
                )
    return findings


def python_files_under(snapshot: Snapshot, root: str) -> list[RepoFile]:
    prefix = root.rstrip("/") + "/"
    return [
        file
        for file in snapshot.files.values()
        if file.path.endswith(".py") and (file.path.startswith(prefix) or file.path == root)
    ]


# Source roots map module names to repository paths: each boundary path and
# its ancestors that look like package roots (src layouts included).
def source_roots(snapshot: Snapshot, boundaries: list[DependencyBoundary]) -> list[str]:
    roots: set[str] = {""}
    for boundary in boundaries:
        parts = boundary.source.split("/")
        for index in range(len(parts)):
            roots.add("/".join(parts[:index]))
    for path in snapshot.files:
        if path.endswith("/__init__.py"):
            package_parent = path.rsplit("/", 2)[0] if path.count("/") >= 2 else ""
            roots.add(package_parent)
    return sorted(roots, key=len, reverse=True)


def first_violation(
    snapshot: Snapshot,
    file_path: str,
    content: str,
    boundary: DependencyBoundary,
    roots: list[str],
) -> str | None:
    for module in imported_modules(content, file_path):
        resolved = resolve_module(snapshot, module, roots)
        if resolved is None:
            continue
        if not allowed_target(resolved, boundary):
            return resolved
    return None


def imported_modules(content: str, file_path: str) -> list[str]:
    try:
        tree = ast.parse(content)
    except SyntaxError:
        return []
    modules: list[str] = []
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            modules.extend(alias.name for alias in node.names)
        elif isinstance(node, ast.ImportFrom):
            modules.append(resolve_relative(node, file_path))
    return [module for module in modules if module]


def resolve_relative(node: ast.ImportFrom, file_path: str) -> str:
    if node.level == 0:
        return node.module or ""
    parts = file_path.split("/")[:-1]
    if node.level > 1:
        parts = parts[: -(node.level - 1)] if node.level - 1 <= len(parts) else []
    base = ".".join(parts)
    return f"{base}.{node.module}" if node.module else base


def resolve_module(snapshot: Snapshot, module: str, roots: list[str]) -> str | None:
    relative = module.replace(".", "/")
    for root in roots:
        candidate = f"{root}/{relative}" if root else relative
        if candidate + ".py" in snapshot.files or candidate + "/__init__.py" in snapshot.files:
            return candidate
        if any(path.startswith(candidate + "/") for path in snapshot.files):
            return candidate
    return None


def allowed_target(resolved: str, boundary: DependencyBoundary) -> bool:
    allowed_prefixes = [boundary.source, *boundary.allow]
    return any(
        resolved == prefix.rstrip("/") or resolved.startswith(prefix.rstrip("/") + "/")
        for prefix in allowed_prefixes
    )
