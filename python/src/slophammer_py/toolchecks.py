"""Executable checks: run the real gates and report failures as findings.

``check --execute`` runs each gate in the target root and converts non-zero
exits into findings under the gate's rule id, mirroring the TypeScript
implementation's tool checks. Commands run through ``uv run`` when the
target is uv-managed and directly otherwise; a tool that cannot be launched
is an infrastructure error, not a finding.
"""

from __future__ import annotations

import subprocess
from collections.abc import Callable
from dataclasses import dataclass

from .config import Config
from .core import Finding
from .dry import dry_findings, max_findings
from .repo import Snapshot, has_file
from .rules import coverage_threshold, python_project_present
from .rules.definitions import (
    PY_COVERAGE,
    PY_DRY,
    PY_FORMAT,
    PY_LINT,
    PY_TEST,
    PY_TYPECHECK,
    definition,
)
from .rules.toolconfig import project_dirs, typechecker_in_use

Runner = Callable[[str, list[str]], "CommandOutput"]


@dataclass(frozen=True)
class CommandOutput:
    code: int
    output: str = ""
    missing: bool = False


def subprocess_runner(root: str, command: list[str]) -> CommandOutput:
    try:
        completed = subprocess.run(
            command,
            cwd=root,
            capture_output=True,
            text=True,
            check=False,
            timeout=1800,
        )
    except FileNotFoundError:
        return CommandOutput(code=127, missing=True)
    except subprocess.TimeoutExpired:
        return CommandOutput(code=124, output="timed out")
    return CommandOutput(code=completed.returncode, output=first_output(completed))


def first_output(completed: subprocess.CompletedProcess[str]) -> str:
    for stream in (completed.stderr, completed.stdout):
        for line in stream.splitlines():
            if line.strip():
                return line.strip()
    return ""


def execute_python_checks(
    snapshot: Snapshot,
    config: Config,
    runner: Runner = subprocess_runner,
    only_rule_ids: list[str] | None = None,
) -> list[Finding]:
    if not python_project_present(snapshot):
        return []
    wanted = set(only_rule_ids or [])
    working_directory = execution_directory(snapshot)
    findings: list[Finding] = []
    for rule_id, label, command in selected_gate_commands(snapshot, config, wanted):
        result = runner(working_directory, run_prefix(snapshot) + command)
        if result.missing or result.code == 0:
            continue
        findings.append(executed_finding(rule_id, label, result))
    if not wanted or PY_DRY in wanted:
        findings.extend(executed_dry_findings(snapshot, config))
    return findings


# The coverage command runs the test suite, so a full run skips the bare
# test gate; a --only selection that names just the test rule still runs it.
def selected_gate_commands(
    snapshot: Snapshot, config: Config, wanted: set[str]
) -> list[tuple[str, str, list[str]]]:
    commands = [
        (rule_id, label, command)
        for rule_id, label, command in gate_commands(snapshot, config)
        if not wanted or rule_id in wanted
    ]
    rule_ids = {rule_id for rule_id, _, _ in commands}
    if PY_COVERAGE in rule_ids:
        commands = [entry for entry in commands if entry[0] != PY_TEST]
    return commands


# Gates run in the project directory, where pyproject.toml scopes the tools,
# rather than at the scan root: a monorepo's templates and fixtures are not
# this project's gate inputs.
def execution_directory(snapshot: Snapshot) -> str:
    directories = project_dirs(snapshot)
    if not directories or directories[0] == "":
        return snapshot.root
    return f"{snapshot.root}/{directories[0]}"


def gate_commands(snapshot: Snapshot, config: Config) -> list[tuple[str, str, list[str]]]:
    commands = [
        (PY_FORMAT, "formatter check failed", ["ruff", "format", "--check", "."]),
        (PY_LINT, "lint failed", ["ruff", "check", "."]),
        (PY_TEST, "tests failed", ["pytest"]),
        (
            PY_COVERAGE,
            "coverage gate failed",
            ["pytest", f"--cov-fail-under={coverage_threshold(config)}", "--cov", "."],
        ),
    ]
    typecheck = typecheck_command(snapshot)
    if typecheck is not None:
        commands.insert(2, (PY_TYPECHECK, "typecheck failed", typecheck))
    return commands


def typecheck_command(snapshot: Snapshot) -> list[str] | None:
    checker = typechecker_in_use(snapshot)
    if checker == "ty":
        return ["ty", "check"]
    if checker == "mypy":
        return ["mypy", "."]
    if checker == "pyright":
        return ["pyright"]
    return None


def run_prefix(snapshot: Snapshot) -> list[str]:
    directories = project_dirs(snapshot)
    project = directories[0] if directories else ""
    lock = f"{project}/uv.lock" if project else "uv.lock"
    return ["uv", "run", "--no-sync"] if has_file(snapshot, lock) else []


def executed_finding(rule_id: str, label: str, result: CommandOutput) -> Finding:
    template = definition(rule_id)
    detail = f": {result.output}" if result.output else ""
    return Finding(
        rule_id=rule_id,
        severity=template.severity,
        path=template.path,
        message=f"{label} (exit {result.code}){detail}",
    )


def executed_dry_findings(snapshot: Snapshot, config: Config) -> list[Finding]:
    found = dry_findings(snapshot, config)
    if len(found) <= max_findings(config):
        return []
    template = definition(PY_DRY)
    return [
        Finding(
            rule_id=PY_DRY,
            severity=template.severity,
            path=template.path,
            message=f"DRY candidates: {len(found)}; maximum: {max_findings(config)}",
        )
    ]
