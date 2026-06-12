"""Rule definitions for the shared repo rules and the Python rule set."""

from __future__ import annotations

from dataclasses import dataclass

from slophammer.core import Severity


@dataclass(frozen=True)
class Definition:
    id: str
    title: str
    severity: Severity
    path: str
    message: str
    description: str


REPO_README = "repo.readme-required"
REPO_AGENTS = "repo.agents-required"
REPO_CI = "repo.ci-required"
REPO_SLOPHAMMER_CI = "repo.slophammer-ci-required"
PY_PROJECT = "py.project-required"
PY_TYPECHECK = "py.typecheck-required"
PY_TYPES_STRICT = "py.types-strict-required"
PY_LINT = "py.lint-required"
PY_FORMAT = "py.format-required"
PY_TEST = "py.test-required"
PY_COVERAGE = "py.coverage-required"
PY_COMPLEXITY = "py.complexity-required"
PY_DRY = "py.dry-required"
PY_MUTATION = "py.mutation-required"
PY_SUPPRESSIONS = "py.suppressions-justified"
PY_AUDIT = "py.dependency-audit-required"
PY_BOUNDARIES = "py.dependency-boundaries-required"
PY_TYPED_MARKER = "py.typed-marker-required"
PY_ABSOLUTE_IMPORTS = "py.absolute-imports-required"
PY_SCOPE = "py.scope-incomplete"

DEFAULT_DEFINITIONS: tuple[Definition, ...] = (
    Definition(
        id=REPO_README,
        title="README required",
        severity="error",
        path="README.md",
        message="README.md is required",
        description="The target repo should have a README.md.",
    ),
    Definition(
        id=REPO_AGENTS,
        title="Agent instructions required",
        severity="error",
        path="AGENTS.md",
        message="AGENTS.md is required",
        description="The target repo should have an AGENTS.md.",
    ),
    Definition(
        id=REPO_CI,
        title="CI workflow required",
        severity="error",
        path=".github/workflows",
        message=".github/workflows must contain at least one .yml or .yaml workflow",
        description="The target repo should have a CI workflow under .github/workflows.",
    ),
    Definition(
        id=REPO_SLOPHAMMER_CI,
        title="Slophammer enforcement required",
        severity="error",
        path=".github/workflows",
        message="CI must run a Slophammer checker when slophammer.yml is present",
        description=(
            "A repository that carries slophammer.yml must execute a Slophammer checker "
            "from binding CI evidence; config without enforcement is decoration."
        ),
    ),
    Definition(
        id=PY_PROJECT,
        title="Python project metadata required",
        severity="error",
        path="pyproject.toml",
        message="Python projects must declare a pyproject.toml",
        description="Production Python code needs project metadata in pyproject.toml.",
    ),
    Definition(
        id=PY_TYPECHECK,
        title="Python typecheck required",
        severity="error",
        path=".github/workflows",
        message="Python projects must run a typechecker (ty, mypy, or pyright) in CI",
        description="Binding CI evidence must invoke a Python typechecker.",
    ),
    Definition(
        id=PY_TYPES_STRICT,
        title="Strict Python typing required",
        severity="error",
        path="pyproject.toml",
        message="Python typechecking must be strict",
        description=(
            "The typechecker configuration must make annotations mandatory and must not "
            "be quietly weakened: no unreasoned demotion of stable default-error ty "
            "rules, error-on-warning enabled, the ignore-default correctness rules "
            "promoted, coded suppressions only, and Ruff ANN annotation coverage."
        ),
    ),
    Definition(
        id=PY_LINT,
        title="Python lint required",
        severity="error",
        path=".github/workflows",
        message="Python projects must run a linter (ruff check) in CI",
        description="Binding CI evidence must invoke a Python linter.",
    ),
    Definition(
        id=PY_FORMAT,
        title="Python format check required",
        severity="error",
        path=".github/workflows",
        message=(
            "Python projects must verify formatting (ruff format --check or black --check) in CI"
        ),
        description="Binding CI evidence must verify formatting without mutating files.",
    ),
    Definition(
        id=PY_TEST,
        title="Python tests required",
        severity="error",
        path=".github/workflows",
        message="Python projects must run tests (pytest) in CI",
        description="Binding CI evidence must run the Python test suite.",
    ),
    Definition(
        id=PY_COVERAGE,
        title="Python coverage gate required",
        severity="error",
        path=".github/workflows",
        message="Python projects must enforce a coverage gate of at least 85",
        description=(
            "Binding CI evidence must enforce coverage, via --cov-fail-under or a "
            "fail_under coverage configuration of at least the configured threshold."
        ),
    ),
    Definition(
        id=PY_COMPLEXITY,
        title="Python complexity gate required",
        severity="error",
        path="pyproject.toml",
        message="Python projects must enforce complexity at most 8 (Ruff C901 or radon)",
        description="Complexity must be capped at the configured maximum.",
    ),
    Definition(
        id=PY_DRY,
        title="Python DRY check required",
        severity="error",
        path=".github/workflows",
        message="Python projects must declare a DRY check",
        description="Binding CI evidence must run a duplication check (slophammer-py dry).",
    ),
    Definition(
        id=PY_MUTATION,
        title="Python mutation testing required",
        severity="error",
        path=".github/workflows",
        message="Python projects must declare mutation testing (mutmut or cosmic-ray)",
        description="Binding CI evidence must declare a mutation testing tool.",
    ),
    Definition(
        id=PY_SUPPRESSIONS,
        title="Justified Python suppressions required",
        severity="error",
        path="",
        message=(
            "Python suppressions must carry a reason: bare # noqa, # type: ignore "
            "without an error code, or uncommented # ty: ignore"
        ),
        description=(
            "Suppression directives in production Python code need a stated reason; "
            "bare # type: ignore without an error code is itself a finding."
        ),
    ),
    Definition(
        id=PY_AUDIT,
        title="Python dependency audit required",
        severity="error",
        path=".github/workflows",
        message="Python projects must audit dependencies (pip-audit or uv audit) in CI",
        description="Binding CI evidence must audit Python dependencies.",
    ),
    Definition(
        id=PY_BOUNDARIES,
        title="Python dependency boundaries required",
        severity="error",
        path="",
        message="Python imports must respect the configured dependency boundaries",
        description=(
            "When python.dependency_boundaries is configured, imports crossing a "
            "boundary outside its allow list are findings."
        ),
    ),
    Definition(
        id=PY_TYPED_MARKER,
        title="py.typed marker required",
        severity="error",
        path="",
        message="Published Python packages must ship a py.typed marker",
        description=(
            "A project that builds a published package must ship the py.typed marker, "
            "or its checked types degrade to Any for every consumer."
        ),
    ),
    Definition(
        id=PY_ABSOLUTE_IMPORTS,
        title="Absolute Python imports required",
        severity="error",
        path="",
        message=(
            "Python imports must be absolute; replace relative imports "
            "(ruff check --select TID252 --fix)"
        ),
        description=(
            "Relative imports defeat grep, break on file moves, and read as "
            "dot-counting at depth; production imports must name the package. "
            "Ruff's TID252 autofix converts a whole repository in one command."
        ),
    ),
    Definition(
        id=PY_SCOPE,
        title="Complete Python scope required",
        severity="error",
        path="slophammer.yml",
        message=(
            "Configured Python scope must cover all production files or exclude them with reasons"
        ),
        description=(
            "Every production Python file must be inside a configured scope or covered "
            "by a conventional or reasoned exclude, so narrowing scope cannot hide code."
        ),
    ),
)


def definition(rule_id: str) -> Definition:
    for item in DEFAULT_DEFINITIONS:
        if item.id == rule_id:
            return item
    raise KeyError(rule_id)
