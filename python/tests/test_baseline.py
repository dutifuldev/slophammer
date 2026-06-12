"""Baseline ratchet tests, ported from typescript/tests/baseline.test.ts."""

import json
from pathlib import Path

import pytest

from slophammer_py.app import check
from slophammer_py.baseline import (
    BaselineError,
    apply_baseline_check,
    debt_line,
    write_baseline,
)
from slophammer_py.core import Finding, new_report

ENTRY = {"rule_id": "repo.readme-required", "path": "README.md"}


def finding(rule_id: str, path: str) -> Finding:
    return Finding(rule_id=rule_id, severity="error", path=path, message="missing")


def write_baseline_file(root: Path, findings: list[dict[str, str]], version: int = 1) -> None:
    (root / "slophammer-baseline.json").write_text(
        json.dumps({"version": version, "findings": findings}) + "\n"
    )


class TestApply:
    def test_baselined_findings_stop_affecting_ok(self, tmp_path: Path):
        write_baseline_file(tmp_path, [ENTRY])
        report = apply_baseline_check(
            str(tmp_path), new_report([finding("repo.readme-required", "README.md")])
        )
        assert report.ok
        assert report.findings[0].baselined is True

    def test_new_findings_keep_failing(self, tmp_path: Path):
        write_baseline_file(tmp_path, [ENTRY])
        report = apply_baseline_check(
            str(tmp_path),
            new_report(
                [
                    finding("repo.readme-required", "README.md"),
                    finding("repo.agents-required", "AGENTS.md"),
                ]
            ),
        )
        assert not report.ok
        assert debt_line(report) == "1 findings baselined; 1 new\n"

    def test_stale_entries_are_an_error(self, tmp_path: Path):
        write_baseline_file(tmp_path, [ENTRY])
        with pytest.raises(BaselineError, match="resolved findings"):
            apply_baseline_check(str(tmp_path), new_report([]))

    def test_missing_baseline_is_an_error(self, tmp_path: Path):
        with pytest.raises(BaselineError, match="missing"):
            apply_baseline_check(str(tmp_path), new_report([]))

    def test_invalid_baselines_are_errors(self, tmp_path: Path):
        cases = [
            ("not json", "parse failed"),
            ('{"version": 2, "findings": []}', "version must be 1"),
            ('{"version": 1, "findings": [], "surprise": true}', "parse failed"),
            ('{"version": 1, "findings": [{"rule_id": "a"}]}', "parse failed"),
        ]
        for content, expected in cases:
            (tmp_path / "slophammer-baseline.json").write_text(content)
            with pytest.raises(BaselineError, match=expected):
                apply_baseline_check(str(tmp_path), new_report([]))


class TestWrite:
    def test_writes_sorted_unique_findings(self, tmp_path: Path):
        report = new_report(
            [
                finding("repo.readme-required", "README.md"),
                finding("repo.agents-required", "AGENTS.md"),
                finding("repo.agents-required", "AGENTS.md"),
            ]
        )
        summary = write_baseline(str(tmp_path), report)
        assert "baseline written: 2 finding(s)" in summary
        content = (tmp_path / "slophammer-baseline.json").read_text()
        assert content.endswith("\n")
        parsed = json.loads(content)
        assert [entry["rule_id"] for entry in parsed["findings"]] == [
            "repo.agents-required",
            "repo.readme-required",
        ]

    def test_refuses_to_grow_the_baseline(self, tmp_path: Path):
        write_baseline_file(tmp_path, [ENTRY])
        report = new_report(
            [
                finding("repo.readme-required", "README.md"),
                finding("repo.agents-required", "AGENTS.md"),
            ]
        )
        with pytest.raises(BaselineError, match="grow the baseline"):
            write_baseline(str(tmp_path), report)

    def test_records_removals_when_shrinking(self, tmp_path: Path):
        write_baseline_file(
            tmp_path, [ENTRY, {"rule_id": "repo.agents-required", "path": "AGENTS.md"}]
        )
        summary = write_baseline(
            str(tmp_path), new_report([finding("repo.agents-required", "AGENTS.md")])
        )
        assert "removed: repo.readme-required at README.md" in summary

    def test_refuses_to_replace_a_malformed_baseline(self, tmp_path: Path):
        (tmp_path / "slophammer-baseline.json").write_text("not json\n")
        with pytest.raises(BaselineError, match="parse failed"):
            write_baseline(str(tmp_path), new_report([finding("a.rule", "x")]))


class TestCheckIntegration:
    def empty_repo(self, tmp_path: Path) -> Path:
        return tmp_path

    def test_baseline_check_exit_codes(self, tmp_path: Path):
        assert check(str(tmp_path), baseline="check").code == 2
        write_baseline_file(
            tmp_path,
            [
                {"rule_id": "repo.agents-required", "path": "AGENTS.md"},
                {"rule_id": "repo.ci-required", "path": ".github/workflows"},
                {"rule_id": "repo.readme-required", "path": "README.md"},
            ],
        )
        result = check(str(tmp_path), baseline="check")
        assert result.code == 0
        assert "3 findings baselined; 0 new" in result.stdout

    def test_baseline_write_then_check(self, tmp_path: Path):
        result = check(str(tmp_path), baseline="write")
        assert result.code == 0
        assert "baseline written: 3 finding(s)" in result.stdout
        assert check(str(tmp_path), baseline="check").code == 0

    def test_sarif_marks_baselined_findings_suppressed(self, tmp_path: Path):
        check(str(tmp_path), baseline="write")
        result = check(str(tmp_path), output_format="sarif", baseline="check")
        parsed = json.loads(result.stdout)
        results = parsed["runs"][0]["results"]
        assert results and all(entry["suppressions"] == [{"kind": "external"}] for entry in results)
