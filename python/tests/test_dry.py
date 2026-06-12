"""Copied-block DRY engine tests."""

import json
from pathlib import Path

from slophammer_py.cli import main
from slophammer_py.config import parse_config
from slophammer_py.dry import dry_findings, find_copied_blocks
from slophammer_py.repo import RepoFile, new_snapshot

BLOCK = (
    "def handler_{n}(payload: dict) -> dict:\n"
    "    cleaned = {{}}\n"
    "    for key, value in payload.items():\n"
    "        if value is None:\n"
    "            continue\n"
    "        if isinstance(value, str):\n"
    "            cleaned[key] = value.strip().lower()\n"
    "        elif isinstance(value, (int, float)):\n"
    "            cleaned[key] = value * 2 + 1\n"
    "        else:\n"
    "            cleaned[key] = repr(value)\n"
    "    return cleaned\n"
)


class TestCopiedBlocks:
    def test_identical_blocks_across_files_match(self):
        files = [
            RepoFile("src/a.py", BLOCK.format(n=1)),
            RepoFile("src/b.py", BLOCK.format(n=1)),
        ]
        findings = find_copied_blocks(files, 40)
        assert len(findings) == 1
        assert findings[0].left.path == "src/a.py"
        assert findings[0].right.path == "src/b.py"
        assert findings[0].tokens >= 40

    def test_different_identifiers_do_not_match(self):
        other = BLOCK.format(n=2).replace("cleaned", "result").replace("payload", "body")
        files = [RepoFile("src/a.py", BLOCK.format(n=1)), RepoFile("src/b.py", other)]
        assert find_copied_blocks(files, 40) == []

    def test_duplicate_within_one_file_matches(self):
        content = BLOCK.format(n=1) + "\n\n" + BLOCK.format(n=1).replace("handler_1", "handler_2")
        findings = find_copied_blocks([RepoFile("src/a.py", content)], 40)
        assert len(findings) == 1
        assert findings[0].left.start_line < findings[0].right.start_line

    def test_short_overlap_is_below_window(self):
        content = "x = 1\ny = 2\n"
        files = [RepoFile("src/a.py", content), RepoFile("src/b.py", content)]
        assert find_copied_blocks(files, 40) == []

    def test_unparseable_files_contribute_partial_tokens(self):
        files = [RepoFile("src/a.py", "def broken(:\n"), RepoFile("src/b.py", "x = 1\n")]
        assert find_copied_blocks(files, 40) == []


class TestDryScoping:
    def config(self, content: str):
        return parse_config(content)

    def test_dry_paths_limit_scope(self):
        snapshot = new_snapshot(
            "/repo",
            [
                RepoFile("src/a.py", BLOCK.format(n=1)),
                RepoFile("vendor_copy/b.py", BLOCK.format(n=1)),
            ],
        )
        config = self.config("python:\n  dry:\n    paths:\n      - src\n")
        assert dry_findings(snapshot, config) == []

    def test_tests_are_not_dry_scope(self):
        snapshot = new_snapshot(
            "/repo",
            [
                RepoFile("src/a.py", BLOCK.format(n=1)),
                RepoFile("tests/test_a.py", BLOCK.format(n=1)),
            ],
        )
        config = self.config("python:\n  dry:\n    copied_blocks:\n      min_tokens: 40\n")
        assert dry_findings(snapshot, config) == []

    def test_disabled_copied_blocks_skip_the_scan(self):
        snapshot = new_snapshot(
            "/repo",
            [RepoFile("src/a.py", BLOCK.format(n=1)), RepoFile("src/b.py", BLOCK.format(n=1))],
        )
        disabled = self.config(
            "python:\n  dry:\n    copied_blocks:\n      enabled: false\n      min_tokens: 40\n"
        )
        assert dry_findings(snapshot, disabled) == []

    def test_min_tokens_config_applies(self):
        snapshot = new_snapshot(
            "/repo",
            [RepoFile("src/a.py", BLOCK.format(n=1)), RepoFile("src/b.py", BLOCK.format(n=1))],
        )
        loose = self.config("python:\n  dry:\n    copied_blocks:\n      min_tokens: 500\n")
        assert dry_findings(snapshot, loose) == []
        tight = self.config("python:\n  dry:\n    copied_blocks:\n      min_tokens: 40\n")
        assert len(dry_findings(snapshot, tight)) == 1


class TestDryCommand:
    def test_findings_exit_one_with_json(self, tmp_path: Path, capsys):
        for name in ("a", "b"):
            target = tmp_path / "src" / f"{name}.py"
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text(BLOCK.format(n=1))
        (tmp_path / "slophammer.yml").write_text(
            "python:\n  dry:\n    copied_blocks:\n      min_tokens: 40\n"
        )
        assert main(["dry", str(tmp_path), "--format", "json"]) == 1
        parsed = json.loads(capsys.readouterr().out)
        assert parsed["ok"] is False
        assert parsed["findings"][0]["kind"] == "copied-block"
        assert parsed["findings"][0]["engine"] == "token-window"
