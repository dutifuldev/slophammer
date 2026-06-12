#!/usr/bin/env python3
"""Extract ty's lint rules and default severities from a ty source checkout.

Writes specs/ty-rules.json and the Python checker's bundled copy. Refreshing
the table is part of bumping the supported ty version:

    python3 scripts/extract-ty-rules.py ~/repos/ty/ruff
"""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

LINT_FILES = (
    "crates/ty_python_semantic/src/types/diagnostic.rs",
    "crates/ty_python_semantic/src/suppression.rs",
    "crates/ty_python_semantic/src/types/string_annotation.rs",
    "crates/ty_python_semantic/src/lint.rs",
)
DECLARATION = re.compile(r"static\s+([A-Z_]+)\s*=\s*\{(.*?)\}", re.DOTALL)
LEVEL = re.compile(r"default_level:\s*Level::(\w+)")
STATUS = re.compile(r"status:\s*LintStatus::(\w+)")


def source_version(ty_source_root: Path) -> str:
    """The ty version at the source checkout, from the ty crate metadata."""
    import re as version_re
    import subprocess

    cargo = ty_source_root / "crates" / "ty" / "Cargo.toml"
    if cargo.exists():
        match = version_re.search(r'^version\s*=\s*"([^"]+)"', cargo.read_text(), version_re.M)
        if match and match.group(1) != "0.0.0":
            return match.group(1)
    described = subprocess.run(
        ["git", "-C", str(ty_source_root), "rev-parse", "--short", "HEAD"],
        capture_output=True,
        text=True,
        check=False,
    )
    return described.stdout.strip() or "unknown"


def extract(ty_source_root: Path) -> dict[str, dict[str, str]]:
    text = "".join(
        (ty_source_root / name).read_text(encoding="utf-8")
        for name in LINT_FILES
        if (ty_source_root / name).exists()
    )
    rules: dict[str, dict[str, str]] = {}
    for name, body in DECLARATION.findall(text):
        level = LEVEL.search(body)
        status = STATUS.search(body)
        if level is None or status is None:
            continue
        rule = name.lower().replace("_", "-")
        rules[rule] = {
            "default_level": level.group(1).lower(),
            "stability": status.group(1).lower(),
        }
    return dict(sorted(rules.items()))


def main() -> int:
    if len(sys.argv) != 2:
        print(__doc__, file=sys.stderr)
        return 2
    repo_root = Path(__file__).resolve().parent.parent
    source_root = Path(sys.argv[1]).expanduser()
    rules = extract(source_root)
    if not rules:
        print("no rules extracted; wrong source root?", file=sys.stderr)
        return 1
    document = {"ty_version": source_version(source_root), "rules": rules}
    serialized = json.dumps(document, indent=2) + "\n"
    for destination in (
        repo_root / "specs" / "ty-rules.json",
        repo_root / "python" / "src" / "slophammer" / "ty_rules.json",
    ):
        destination.write_text(serialized, encoding="utf-8")
        print(f"wrote {len(rules)} rules to {destination}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
