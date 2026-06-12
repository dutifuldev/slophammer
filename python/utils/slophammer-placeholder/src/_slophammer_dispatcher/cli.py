"""Bare slophammer command entry point."""

from __future__ import annotations

import sys
from importlib import metadata

PLACEHOLDER_MESSAGE = (
    "slophammer is reserved for the future umbrella dispatcher. "
    "Use slophammer-py for the current Python checker.\n"
)


def main(argv: list[str] | None = None) -> int:
    if argv is None:
        argv = sys.argv[1:]
    if argv == ["--version"]:
        return print_version()
    sys.stderr.write(PLACEHOLDER_MESSAGE)
    return 2


def print_version() -> int:
    try:
        version = metadata.version("slophammer")
    except metadata.PackageNotFoundError:
        version = "0.0.0"
    print(version)
    return 0
