"""Rule evaluation tests over synthetic repositories."""

from slophammer_py.config import Config, load_config, parse_config
from slophammer_py.repo import RepoFile, new_snapshot
from slophammer_py.rules import explain, run_rules

GATE_STEPS = """\
      - run: uv run ty check src --error-on-warning
      - run: uv run ruff check .
      - run: uv run ruff format --check .
      - run: uv run pytest --cov=src --cov-fail-under=85
      - run: uv run slophammer-py dry .
      - run: uv run mutmut run
      - run: uv run pip-audit
      - run: uvx slophammer-py check .
"""

STRICT_TY_TOML = """\
[terminal]
error-on-warning = true

[analysis]
respect-type-ignore-comments = false

[rules]
missing-type-argument = "error"
possibly-missing-attribute = "error"
possibly-unresolved-reference = "error"
possibly-missing-import = "error"
"""

STRICT_PYPROJECT = """\
[project]
name = "demo"
version = "0.0.0"

[tool.ruff.lint]
select = ["E", "F", "ANN", "C90"]

[tool.ruff.lint.mccabe]
max-complexity = 8
"""


def clean_python_repo(overrides: dict[str, str] | None = None) -> dict[str, str]:
    files = {
        "README.md": "# Demo\n",
        "AGENTS.md": "# Agents\n",
        ".github/workflows/ci.yml": (
            f"name: CI\non: [push]\njobs:\n  check:\n    steps:\n{GATE_STEPS}"
        ),
        "pyproject.toml": STRICT_PYPROJECT,
        "ty.toml": STRICT_TY_TOML,
        "src/demo/__init__.py": "",
        "src/demo/main.py": "def run() -> int:\n    return 0\n",
        "slophammer.yml": "python:\n  coverage:\n    threshold: 85\n",
    }
    files.update(overrides or {})
    return files


def report_for(files: dict[str, str], only: list[str] | None = None):
    snapshot = new_snapshot("/repo", [RepoFile(path, content) for path, content in files.items()])
    config = load_config(snapshot)
    return run_rules(snapshot, config, only)


def rule_ids(report) -> list[str]:
    return [finding.rule_id for finding in report.findings]


class TestCleanRepo:
    def test_clean_python_repo_passes(self):
        report = report_for(clean_python_repo())
        assert report.findings == []
        assert report.ok

    def test_repo_without_python_skips_python_rules(self):
        report = report_for(
            {
                "README.md": "# Demo\n",
                "AGENTS.md": "# Agents\n",
                ".github/workflows/ci.yml": (
                    "name: CI\non: [push]\njobs:\n  c:\n    steps:\n      - run: true\n"
                ),
            }
        )
        assert report.findings == []


class TestRepoRules:
    def test_missing_readme_and_agents(self):
        files = clean_python_repo()
        del files["README.md"]
        del files["AGENTS.md"]
        ids = rule_ids(report_for(files))
        assert "repo.readme-required" in ids
        assert "repo.agents-required" in ids

    def test_missing_ci(self):
        files = clean_python_repo()
        del files[".github/workflows/ci.yml"]
        ids = rule_ids(report_for(files))
        assert "repo.ci-required" in ids

    def test_unenforced_config(self):
        files = clean_python_repo(
            {
                ".github/workflows/ci.yml": (
                    "name: CI\non: [push]\njobs:\n  check:\n    steps:\n"
                    "      - run: uv run pytest\n"
                )
            }
        )
        ids = rule_ids(report_for(files, only=["repo.slophammer-ci-required"]))
        assert ids == ["repo.slophammer-ci-required"]

    def test_action_reference_satisfies_slophammer_ci(self):
        files = clean_python_repo(
            {
                ".github/workflows/ci.yml": (
                    "name: CI\non: [push]\njobs:\n  check:\n    steps:\n"
                    "      - uses: dutifuldev/slophammer@v0.3.0\n"
                )
            }
        )
        ids = rule_ids(report_for(files, only=["repo.slophammer-ci-required"]))
        assert ids == []


class TestGateRules:
    def test_each_missing_gate_fires_its_rule(self):
        cases = {
            "ty check": "py.typecheck-required",
            "ruff check .": "py.lint-required",
            "ruff format --check .": "py.format-required",
            "pytest --cov=src --cov-fail-under=85": "py.coverage-required",
            "slophammer-py dry .": "py.dry-required",
            "mutmut run": "py.mutation-required",
            "pip-audit": "py.dependency-audit-required",
        }
        for needle, rule in cases.items():
            steps = "\n".join(line for line in GATE_STEPS.split("\n") if needle not in line)
            files = clean_python_repo(
                {
                    ".github/workflows/ci.yml": (
                        f"name: CI\non: [push]\njobs:\n  check:\n    steps:\n{steps}\n"
                    )
                }
            )
            ids = rule_ids(report_for(files, only=[rule]))
            assert ids == [rule], f"{rule} should fire when {needle!r} is removed"

    def test_pytest_without_coverage_still_satisfies_tests(self):
        files = clean_python_repo()
        ids = rule_ids(report_for(files, only=["py.test-required"]))
        assert ids == []

    def test_missing_pyproject_fires_project_rule(self):
        files = clean_python_repo()
        del files["pyproject.toml"]
        ids = rule_ids(report_for(files, only=["py.project-required"]))
        assert ids == ["py.project-required"]

    def test_complexity_via_radon_command(self):
        files = clean_python_repo({"pyproject.toml": '[project]\nname = "demo"\nversion = "0"\n'})
        files[".github/workflows/ci.yml"] += "      - run: uv run radon cc --max B src\n"
        ids = rule_ids(report_for(files, only=["py.complexity-required"]))
        assert ids == []

    def test_complexity_missing_when_no_mccabe(self):
        files = clean_python_repo({"pyproject.toml": '[project]\nname = "demo"\nversion = "0"\n'})
        ids = rule_ids(report_for(files, only=["py.complexity-required"]))
        assert ids == ["py.complexity-required"]


class TestTypedMarker:
    def test_published_package_needs_marker(self):
        files = clean_python_repo(
            {
                "pyproject.toml": STRICT_PYPROJECT
                + '\n[build-system]\nrequires = ["hatchling"]\nbuild-backend = "hatchling.build"\n'
            }
        )
        ids = rule_ids(report_for(files, only=["py.typed-marker-required"]))
        assert ids == ["py.typed-marker-required"]
        files["src/demo/py.typed"] = ""
        ids = rule_ids(report_for(files, only=["py.typed-marker-required"]))
        assert ids == []

    def test_applications_are_exempt(self):
        ids = rule_ids(report_for(clean_python_repo(), only=["py.typed-marker-required"]))
        assert ids == []


class TestSeverityOverrides:
    def test_rule_severity_override_applies(self):
        files = clean_python_repo(
            {"slophammer.yml": "rules:\n  repo.readme-required:\n    severity: warn\n"}
        )
        del files["README.md"]
        report = report_for(files, only=["repo.readme-required"])
        assert report.findings[0].severity == "warn"


class TestExplain:
    def test_explains_known_rules(self):
        text = explain("py.types-strict-required")
        assert text is not None
        assert "py.types-strict-required" in text
        assert explain("no.such-rule") is None


class TestScopeCoverage:
    def test_report_carries_scope_counts(self):
        files = clean_python_repo(
            {
                "slophammer.yml": (
                    "python:\n  coverage:\n    threshold: 85\n    paths:\n      - src\n"
                ),
            }
        )
        report = report_for(files)
        assert report.scope is not None
        assert report.scope.scanned == report.scope.production_files

    def test_carved_scope_fires_scope_incomplete(self):
        files = clean_python_repo(
            {
                "slophammer.yml": (
                    "python:\n  coverage:\n    threshold: 85\n    paths:\n      - src/demo\n"
                ),
                "corner/extra.py": "def hidden() -> int:\n    return 1\n",
            }
        )
        ids = rule_ids(report_for(files, only=["py.scope-incomplete"]))
        assert ids == ["py.scope-incomplete"]

    def test_reasoned_exclude_covers_carved_scope(self):
        files = clean_python_repo(
            {
                "slophammer.yml": (
                    "python:\n"
                    "  coverage:\n"
                    "    threshold: 85\n"
                    "    paths:\n"
                    "      - src/demo\n"
                    "    exclude:\n"
                    "      - pattern: corner/**\n"
                    "        reason: prototype corner kept out of every gate\n"
                ),
                "corner/extra.py": "def hidden() -> int:\n    return 1\n",
            }
        )
        ids = rule_ids(report_for(files, only=["py.scope-incomplete"]))
        assert ids == []


class TestBoundaries:
    def test_boundary_violation_is_a_finding(self):
        files = clean_python_repo(
            {
                "slophammer.yml": (
                    "python:\n  dependency_boundaries:\n    - from: src/demo\n      allow: []\n"
                ),
                "src/demo/main.py": "from src.helpers import util\n",
                "src/helpers/__init__.py": "",
                "src/helpers/util.py": "VALUE = 1\n",
            }
        )
        ids = rule_ids(report_for(files, only=["py.dependency-boundaries-required"]))
        assert ids == ["py.dependency-boundaries-required"]

    def test_allowed_imports_pass(self):
        files = clean_python_repo(
            {
                "slophammer.yml": (
                    "python:\n"
                    "  dependency_boundaries:\n"
                    "    - from: src/demo\n"
                    "      allow:\n"
                    "        - src/helpers\n"
                ),
                "src/demo/main.py": "from src.helpers import util\n",
                "src/helpers/__init__.py": "",
                "src/helpers/util.py": "VALUE = 1\n",
            }
        )
        ids = rule_ids(report_for(files, only=["py.dependency-boundaries-required"]))
        assert ids == []

    def test_third_party_imports_are_ignored(self):
        files = clean_python_repo(
            {
                "slophammer.yml": (
                    "python:\n  dependency_boundaries:\n    - from: src/demo\n      allow: []\n"
                ),
                "src/demo/main.py": "import json\nimport yaml\n",
            }
        )
        ids = rule_ids(report_for(files, only=["py.dependency-boundaries-required"]))
        assert ids == []


def test_parse_config_rejects_unknown_keys():
    import pytest

    from slophammer_py.config import ConfigError

    with pytest.raises(ConfigError, match=r"python\.made_up is not supported"):
        parse_config("python:\n  made_up: true\n")


def test_default_config_when_missing():
    snapshot = new_snapshot("/repo", [])
    assert load_config(snapshot) == Config()
