"""Absolute import discipline tests."""

from test_rules import clean_python_repo, report_for, rule_ids

ONLY = ["py.absolute-imports-required"]


class TestAbsoluteImports:
    def test_absolute_imports_pass(self):
        files = clean_python_repo(
            {"src/demo/main.py": "from demo.helpers import util\nimport json\n"}
        )
        assert rule_ids(report_for(files, only=ONLY)) == []

    def test_relative_import_is_a_finding(self):
        files = clean_python_repo({"src/demo/main.py": "from .helpers import util\n"})
        report = report_for(files, only=ONLY)
        assert rule_ids(report) == ["py.absolute-imports-required"]
        assert "(line 1)" in report.findings[0].message
        assert "TID252" in report.findings[0].message

    def test_parent_relative_import_is_a_finding(self):
        files = clean_python_repo({"src/demo/main.py": "from ..config import load\n"})
        assert rule_ids(report_for(files, only=ONLY)) == ["py.absolute-imports-required"]

    def test_first_offense_per_file(self):
        files = clean_python_repo(
            {"src/demo/main.py": "import json\nfrom . import a\nfrom . import b\n"}
        )
        report = report_for(files, only=ONLY)
        assert len(report.findings) == 1
        assert "(line 2)" in report.findings[0].message

    def test_tests_and_conventional_paths_are_exempt(self):
        files = clean_python_repo(
            {
                "tests/test_demo.py": "from .helpers import util\n",
                "scripts/tool.py": "from .common import x\n",
                "app/migrations/0001.py": "from .base import m\n",
            }
        )
        assert rule_ids(report_for(files, only=ONLY)) == []

    def test_quoted_import_text_is_invisible(self):
        files = clean_python_repo({"src/demo/main.py": 'EXAMPLE = "from .helpers import util"\n'})
        assert rule_ids(report_for(files, only=ONLY)) == []

    def test_unparseable_files_are_skipped(self):
        files = clean_python_repo({"src/demo/broken.py": "def broken(:\n"})
        assert rule_ids(report_for(files, only=ONLY)) == []
