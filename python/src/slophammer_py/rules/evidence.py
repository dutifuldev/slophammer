"""Command evidence matchers over binding CI evidence.

Matchers are uv-native: tools count when invoked bare, via ``uv run``,
``uvx``, ``uv run --directory <dir>``, or ``python -m``. Segments split on
newlines, ``&&``, and ``;``; segments containing ``||`` are discredited the
same way the other implementations discredit fallback chains.
"""

from __future__ import annotations

import re

from ..repo import Snapshot, command_files

CHECK_INVOCATION_WINDOW = 160
CHECKER_BINARIES = ("slophammer-go", "slophammer-ts", "slophammer-rs", "slophammer-py")


def command_text(snapshot: Snapshot) -> str:
    return "\n".join(file.content for file in command_files(snapshot))


def command_segments(content: str) -> list[str]:
    segments = re.split(r"\n|&&|;", content.replace("\\\n", " "))
    normalized = (normalize_command_content(segment).strip() for segment in segments)
    return [segment for segment in normalized if segment and "||" not in segment]


def normalize_command_content(content: str) -> str:
    return re.sub(r"\s+", " ", content.replace("\\\n", " ")).lower()


def snapshot_segments(snapshot: Snapshot) -> list[str]:
    return [
        segment for file in command_files(snapshot) for segment in command_segments(file.content)
    ]


def any_segment(snapshot: Snapshot, pattern: str) -> bool:
    compiled = re.compile(pattern)
    return any(compiled.search(segment) for segment in snapshot_segments(snapshot))


# A tool invocation: bare, `uv run [--directory d] [--frozen|--locked] tool`,
# `uvx tool`, or `python -m tool`.
def tool_pattern(tool: str) -> str:
    runner = r"(?:uv run(?: -[\w-]+(?: [\w./-]+)?)* |uvx(?: -[\w-]+(?: [\w./-]+)?)* |python3? -m )?"
    return rf"\b{runner}{tool}\b"


def has_typecheck_command(snapshot: Snapshot) -> bool:
    return (
        any_segment(snapshot, tool_pattern("ty") + r"(?: [\w./=-]+)* check\b")
        or any_segment(snapshot, tool_pattern("mypy"))
        or any_segment(snapshot, tool_pattern("pyright"))
    )


def has_lint_command(snapshot: Snapshot) -> bool:
    return (
        any_segment(snapshot, tool_pattern("ruff") + r" check\b")
        or any_segment(snapshot, tool_pattern("flake8"))
        or any_segment(snapshot, tool_pattern("pylint"))
    )


def has_format_command(snapshot: Snapshot) -> bool:
    return any_segment(snapshot, tool_pattern("ruff") + r" format\b[^\n]*--check\b") or any_segment(
        snapshot, tool_pattern("black") + r"\b[^\n]*--check\b"
    )


def has_test_command(snapshot: Snapshot) -> bool:
    return (
        any_segment(snapshot, tool_pattern("pytest"))
        or any_segment(snapshot, r"\bpython3? -m unittest\b")
        or any_segment(snapshot, tool_pattern("tox"))
    )


def has_coverage_command(snapshot: Snapshot, threshold: int) -> bool:
    pattern = re.compile(r"--cov-fail-under[ =](\d+(?:\.\d+)?)")
    for segment in snapshot_segments(snapshot):
        match = pattern.search(segment)
        if match is not None and float(match.group(1)) >= threshold:
            return True
    return has_coverage_run(snapshot) and has_coverage_config_threshold(snapshot, threshold)


def has_coverage_run(snapshot: Snapshot) -> bool:
    return any_segment(snapshot, tool_pattern("pytest") + r"\b[^\n]*--cov\b") or any_segment(
        snapshot, tool_pattern("coverage") + r" (?:run|report)\b"
    )


def has_coverage_config_threshold(snapshot: Snapshot, threshold: int) -> bool:
    from .toolconfig import coverage_fail_under

    configured = coverage_fail_under(snapshot)
    return configured is not None and configured >= threshold


def has_complexity_command(snapshot: Snapshot) -> bool:
    return any_segment(snapshot, tool_pattern("radon") + r" cc\b") or any_segment(
        snapshot, tool_pattern("xenon")
    )


def has_dry_command(snapshot: Snapshot) -> bool:
    return (
        any_segment(snapshot, r"\bslophammer-py(?:@\S+)? dry\b")
        or any_segment(snapshot, r"\bslophammer(?:@\S+)? python dry\b")
        or any_segment(snapshot, tool_pattern("jscpd"))
        or any_segment(snapshot, tool_pattern("pylint") + r"\b[^\n]*duplicate-code")
    )


def has_mutation_command(snapshot: Snapshot) -> bool:
    return any_segment(snapshot, tool_pattern("mutmut")) or any_segment(
        snapshot, tool_pattern("cosmic-ray")
    )


def has_audit_command(snapshot: Snapshot) -> bool:
    return any_segment(snapshot, tool_pattern("pip-audit")) or any_segment(
        snapshot, r"\buv(?:x)? (?:tool run )?(?:pip-)?audit\b"
    )


def slophammer_invocation(evidence: str) -> bool:
    if "uses: dutifuldev/slophammer@" in evidence:
        return True
    return any(invocation_with_check(evidence, binary) for binary in CHECKER_BINARIES)


def invocation_with_check(evidence: str, binary: str) -> bool:
    index = evidence.find(binary)
    while index >= 0:
        if " check" in evidence[index : index + CHECK_INVOCATION_WINDOW]:
            return True
        index = evidence.find(binary, index + 1)
    return False
