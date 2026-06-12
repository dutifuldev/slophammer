"""Report serialization branches."""

import json

from slophammer_py.core import Finding, ScopeCoverage, new_report
from slophammer_py.report import write_json, write_sarif, write_text


def finding(rule_id: str = "repo.readme-required", path: str = "README.md") -> Finding:
    return Finding(rule_id=rule_id, severity="error", path=path, message="missing")


class TestText:
    def test_findings_with_scope_line(self):
        report = new_report([finding()], scope=ScopeCoverage(scanned=3, production_files=4))
        text = write_text(report)
        assert "error repo.readme-required README.md: missing" in text
        assert "1 finding(s)" in text
        assert "scope: scanned 3 of 4 production files" in text

    def test_ok_without_scope(self):
        assert write_text(new_report([])) == "OK: no findings\n"


class TestJSON:
    def test_scope_block_serialized(self):
        report = new_report([], scope=ScopeCoverage(scanned=1, production_files=1))
        parsed = json.loads(write_json(report))
        assert parsed["scope"] == {"scanned": 1, "production_files": 1}

    def test_findings_sorted_by_rule_then_path(self):
        report = new_report([finding("z.rule", "a"), finding("a.rule", "b")])
        parsed = json.loads(write_json(report))
        assert [item["rule_id"] for item in parsed["findings"]] == ["a.rule", "z.rule"]


class TestSARIF:
    def test_warn_severity_maps_to_warning(self):
        warn = Finding(rule_id="x.rule", severity="warn", path="x", message="m")
        parsed = json.loads(write_sarif(new_report([warn])))
        assert parsed["runs"][0]["results"][0]["level"] == "warning"

    def test_empty_path_has_no_location(self):
        pathless = Finding(rule_id="x.rule", severity="error", path="", message="m")
        parsed = json.loads(write_sarif(new_report([pathless])))
        assert "locations" not in parsed["runs"][0]["results"][0]

    def test_rules_deduplicated(self):
        report = new_report([finding(path="a"), finding(path="b")])
        parsed = json.loads(write_sarif(report))
        assert len(parsed["runs"][0]["tool"]["driver"]["rules"]) == 1
