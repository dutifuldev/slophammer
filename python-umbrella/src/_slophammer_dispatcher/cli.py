"""Bare slophammer command entry point."""

from __future__ import annotations

import sys
from importlib import metadata

from slophammer.cli import main as run_python_checker


def main(argv: list[str] | None = None) -> int:
    if argv is None:
        argv = sys.argv[1:]
    if argv == ["--version"]:
        return print_version()
    return run_python_checker(argv)


def print_version() -> int:
    try:
        version = metadata.version("slophammer")
    except metadata.PackageNotFoundError:
        version = "0.0.0"
    print(version)
    return 0
