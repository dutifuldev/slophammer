"""slophammer-py command line interface."""

from __future__ import annotations

import argparse
import sys
from importlib import metadata

from .app import CommandResult, check, dry
from .rules import explain
from .rules.definitions import DEFAULT_DEFINITIONS

FORMATS = ("text", "json", "sarif")


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    arguments = parser.parse_args(argv)
    result = dispatch(arguments)
    if result.stdout:
        sys.stdout.write(result.stdout)
    if result.stderr:
        sys.stderr.write(result.stderr)
    return result.code


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="slophammer-py")
    parser.add_argument("--version", action="version", version=package_version())
    commands = parser.add_subparsers(dest="command", required=True)

    check_parser = commands.add_parser("check", help="Check a repository against the rules")
    check_parser.add_argument("path")
    check_parser.add_argument("--format", choices=FORMATS, default="text")
    check_parser.add_argument("--only", help="Comma-separated rule ids to run")
    baseline = check_parser.add_mutually_exclusive_group()
    baseline.add_argument("--baseline", action="store_true", help="Apply the checked-in baseline")
    baseline.add_argument(
        "--baseline-write", action="store_true", help="Record current findings as the baseline"
    )

    dry_parser = commands.add_parser("dry", help="Run the copied-block DRY check")
    dry_parser.add_argument("path")
    dry_parser.add_argument("--format", choices=("text", "json"), default="text")

    explain_parser = commands.add_parser("explain", help="Explain a rule")
    explain_parser.add_argument("rule_id")

    commands.add_parser("rules", help="List rule ids")
    return parser


def dispatch(arguments: argparse.Namespace) -> CommandResult:
    if arguments.command == "check":
        return run_check(arguments)
    if arguments.command == "dry":
        return dry(arguments.path, output_format=arguments.format)
    if arguments.command == "explain":
        return run_explain(arguments.rule_id)
    return CommandResult(
        code=0, stdout="".join(f"{definition.id}\n" for definition in DEFAULT_DEFINITIONS)
    )


def run_check(arguments: argparse.Namespace) -> CommandResult:
    only, error = parse_only(arguments.only)
    if error is not None:
        return CommandResult(code=2, stderr=error)
    baseline = "write" if arguments.baseline_write else "check" if arguments.baseline else "off"
    return check(
        arguments.path, output_format=arguments.format, only_rule_ids=only, baseline=baseline
    )


def parse_only(value: str | None) -> tuple[list[str] | None, str | None]:
    if value is None:
        return None, None
    known = {definition.id for definition in DEFAULT_DEFINITIONS}
    rule_ids = [rule_id.strip() for rule_id in value.split(",")]
    if not any(rule_ids) or "" in rule_ids:
        return None, "check failed: --only requires rule ids\n"
    unknown = [rule_id for rule_id in rule_ids if rule_id not in known]
    if unknown:
        return None, f"check failed: unknown rule ids: {', '.join(unknown)}\n"
    return rule_ids, None


def package_version() -> str:
    try:
        return metadata.version("slophammer-py")
    except metadata.PackageNotFoundError:
        return "0.0.0"


if __name__ == "__main__":
    raise SystemExit(main())
