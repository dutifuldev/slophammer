"""Repository walk producing a snapshot of small text files."""

from __future__ import annotations

import os
from pathlib import Path

from .repo import RepoFile, Snapshot, new_snapshot

MAX_FILE_BYTES = 1 << 20
SKIPPED_DIRECTORIES = {
    ".git",
    "node_modules",
    ".venv",
    ".mypy_cache",
    ".pytest_cache",
    ".ruff_cache",
    ".stryker-tmp",
    "mutants",
    "__pycache__",
    "dist",
    "coverage",
    "target",
}


def scan_repo(root: str) -> Snapshot:
    absolute_root = Path(root).resolve()
    files: list[RepoFile] = []
    for current, directories, names in os.walk(absolute_root):
        directories[:] = sorted(name for name in directories if name not in SKIPPED_DIRECTORIES)
        for name in sorted(names):
            file = read_small_text_file(absolute_root, Path(current) / name)
            if file is not None:
                files.append(file)
    return new_snapshot(str(absolute_root), files)


def read_small_text_file(root: Path, file_path: Path) -> RepoFile | None:
    try:
        if not file_path.is_file() or file_path.stat().st_size > MAX_FILE_BYTES:
            return None
        content = file_path.read_text(encoding="utf-8")
    except (OSError, UnicodeDecodeError):
        return None
    if "\0" in content:
        return None
    return RepoFile(path=file_path.relative_to(root).as_posix(), content=content)
