"""Tool configuration as evidence: ty, mypy, pyright, Ruff, and coverage.

Quiet weakening happens in tool config files and invocation flags, not in
workflow text, so these are parsed structurally. The ty strictness contract
is judged against the bundled default-severity table (ty_rules.json),
generated from ty's source by scripts/extract-ty-rules.py.
"""

from __future__ import annotations

import configparser
import json
import re
import tomllib
from collections.abc import Mapping
from dataclasses import dataclass, field
from importlib import resources

from ..config import Config
from ..repo import Snapshot
from .evidence import snapshot_segments

LEVELS = ("ignore", "warn", "error")
REQUIRED_PROMOTIONS = (
    "missing-type-argument",
    "possibly-missing-attribute",
    "possibly-unresolved-reference",
    "possibly-missing-import",
)


@dataclass(frozen=True)
class TyContract:
    """The effective ty configuration relevant to the strictness contract."""

    severities: Mapping[str, str] = field(default_factory=dict)
    error_on_warning: bool = False
    respects_type_ignore: bool = True


def load_ty_rules() -> Mapping[str, Mapping[str, str]]:
    content = resources.files("slophammer_py").joinpath("ty_rules.json").read_text("utf-8")
    parsed = json.loads(content)
    return parsed if isinstance(parsed, dict) else {}


def parse_toml(content: str) -> Mapping[str, object]:
    try:
        return tomllib.loads(content)
    except (tomllib.TOMLDecodeError, ValueError):
        return {}


def file_toml(snapshot: Snapshot, path: str) -> Mapping[str, object]:
    file = snapshot.files.get(path)
    return parse_toml(file.content) if file is not None else {}


IGNORED_PROJECT_SEGMENTS = {
    "fixtures",
    "templates",
    "testdata",
    "node_modules",
    "vendor",
    "dist",
    "build",
}


# Python projects can live in nested directories (the slophammer repo's own
# checker lives under python/). Tool config lookups search the repo root and
# every non-conventional directory carrying a pyproject.toml.
def project_dirs(snapshot: Snapshot) -> list[str]:
    dirs: list[str] = []
    for path in snapshot.files:
        if path.rsplit("/", 1)[-1] != "pyproject.toml":
            continue
        if set(path.split("/")[:-1]) & IGNORED_PROJECT_SEGMENTS:
            continue
        dirs.append(path.rsplit("/", 1)[0] if "/" in path else "")
    return sorted(dirs, key=len)


def project_file_toml(snapshot: Snapshot, name: str) -> Mapping[str, object]:
    for directory in ["", *project_dirs(snapshot)]:
        path = f"{directory}/{name}" if directory else name
        config = file_toml(snapshot, path)
        if config:
            return config
    return {}


def project_file(snapshot: Snapshot, name: str) -> str | None:
    for directory in ["", *project_dirs(snapshot)]:
        path = f"{directory}/{name}" if directory else name
        if path in snapshot.files:
            return path
    return None


def pyproject_tool(snapshot: Snapshot, tool: str) -> Mapping[str, object]:
    for directory in ["", *project_dirs(snapshot)]:
        path = f"{directory}/pyproject.toml" if directory else "pyproject.toml"
        section = as_mapping(as_mapping(file_toml(snapshot, path).get("tool")).get(tool))
        if section:
            return section
    return {}


def ty_contract(snapshot: Snapshot) -> TyContract:
    config = project_file_toml(snapshot, "ty.toml") or pyproject_tool(snapshot, "ty")
    severities = dict(flag_severities(snapshot))
    for rule, level in as_mapping(config.get("rules")).items():
        if isinstance(level, str):
            severities[str(rule)] = level
    terminal = as_mapping(config.get("terminal"))
    analysis = as_mapping(config.get("analysis"))
    return TyContract(
        severities=severities,
        error_on_warning=terminal.get("error-on-warning") is True
        or any("--error-on-warning" in segment for segment in ty_segments(snapshot)),
        respects_type_ignore=analysis.get("respect-type-ignore-comments") is not False,
    )


def ty_segments(snapshot: Snapshot) -> list[str]:
    return [
        segment
        for segment in snapshot_segments(snapshot)
        if re.search(r"\bty (?:check|server)\b", segment)
    ]


def flag_severities(snapshot: Snapshot) -> dict[str, str]:
    severities: dict[str, str] = {}
    for segment in ty_segments(snapshot):
        for flag, level in (("--ignore", "ignore"), ("--warn", "warn"), ("--error", "error")):
            for match in re.finditer(rf"{flag}[ =]([\w-]+)", segment):
                severities[match.group(1)] = level
    return severities


def demoted_stable_rules(contract: TyContract, config: Config) -> list[str]:
    table = load_ty_rules()
    allowed = {
        demotion.rule
        for demotion in (
            config.python.typecheck.demotions
            if config.python is not None and config.python.typecheck is not None
            else []
        )
    }
    demoted: list[str] = []
    for rule, level in contract.severities.items():
        entry = table.get(rule)
        if entry is None or entry.get("stability") != "stable":
            continue
        if entry.get("default_level") != "error":
            continue
        if level_rank(level) < level_rank("error") and rule not in allowed:
            demoted.append(rule)
    return sorted(demoted)


def missing_promotions(contract: TyContract) -> list[str]:
    return [rule for rule in REQUIRED_PROMOTIONS if contract.severities.get(rule) != "error"]


def level_rank(level: str) -> int:
    return LEVELS.index(level) if level in LEVELS else len(LEVELS)


def typechecker_in_use(snapshot: Snapshot) -> str | None:
    segments = snapshot_segments(snapshot)
    for tool, pattern in (
        ("ty", r"\bty (?:check)\b"),
        ("mypy", r"\bmypy\b"),
        ("pyright", r"\bpyright\b"),
    ):
        if any(re.search(pattern, segment) for segment in segments):
            return tool
    return None


MYPY_STRICT_FLAGS = ("disallow_untyped_defs", "disallow_incomplete_defs", "check_untyped_defs")


def mypy_strict(snapshot: Snapshot) -> bool:
    if any(
        re.search(r"\bmypy\b[^\n]* --strict\b", segment) for segment in snapshot_segments(snapshot)
    ):
        return True
    section = pyproject_tool(snapshot, "mypy")
    if section.get("strict") is True:
        return True
    if all(section.get(flag) is True for flag in MYPY_STRICT_FLAGS) and section:
        return True
    return mypy_ini_strict(snapshot)


def mypy_ini_strict(snapshot: Snapshot) -> bool:
    for name in ("mypy.ini", "setup.cfg"):
        path = project_file(snapshot, name)
        file = snapshot.files.get(path) if path is not None else None
        if file is None:
            continue
        parser = configparser.ConfigParser()
        try:
            parser.read_string(file.content)
        except configparser.Error:
            continue
        if not parser.has_section("mypy"):
            continue
        if parser.getboolean("mypy", "strict", fallback=False):
            return True
        if all(parser.getboolean("mypy", flag, fallback=False) for flag in MYPY_STRICT_FLAGS):
            return True
    return False


def mypy_pydantic_plugin(snapshot: Snapshot) -> bool:
    plugins = pyproject_tool(snapshot, "mypy").get("plugins")
    if isinstance(plugins, list) and any("pydantic" in str(plugin) for plugin in plugins):
        return True
    if isinstance(plugins, str) and "pydantic" in plugins:
        return True
    for name in ("mypy.ini", "setup.cfg"):
        path = project_file(snapshot, name)
        file = snapshot.files.get(path) if path is not None else None
        if file is not None and re.search(r"plugins\s*=.*pydantic", file.content):
            return True
    return False


def pydantic_dependency(snapshot: Snapshot) -> bool:
    for pyproject in project_pyprojects(snapshot):
        project = as_mapping(pyproject.get("project"))
        dependencies = project.get("dependencies")
        if isinstance(dependencies, list) and any(
            str(dependency).strip().lower().startswith("pydantic") for dependency in dependencies
        ):
            return True
    return False


def project_pyprojects(snapshot: Snapshot) -> list[Mapping[str, object]]:
    configs: list[Mapping[str, object]] = []
    for directory in ["", *project_dirs(snapshot)]:
        path = f"{directory}/pyproject.toml" if directory else "pyproject.toml"
        config = file_toml(snapshot, path)
        if config:
            configs.append(config)
    return configs


def pyright_strict(snapshot: Snapshot) -> bool:
    config_path = project_file(snapshot, "pyrightconfig.json")
    file = snapshot.files.get(config_path) if config_path is not None else None
    if file is not None:
        try:
            parsed = json.loads(file.content)
        except ValueError:
            parsed = {}
        if isinstance(parsed, dict) and parsed.get("typeCheckingMode") == "strict":
            return True
    return pyproject_tool(snapshot, "pyright").get("typeCheckingMode") == "strict"


def ruff_lint_config(snapshot: Snapshot) -> Mapping[str, object]:
    for name in ("ruff.toml", ".ruff.toml"):
        config = project_file_toml(snapshot, name)
        if config:
            return as_mapping(config.get("lint")) or config
    return as_mapping(pyproject_tool(snapshot, "ruff").get("lint"))


def ruff_selects(snapshot: Snapshot, code: str) -> bool:
    select = ruff_lint_config(snapshot).get("select")
    if not isinstance(select, list):
        return False
    return any(str(item) in ("ALL", code) for item in select)


def ruff_complexity_limit(snapshot: Snapshot) -> int | None:
    mccabe = as_mapping(ruff_lint_config(snapshot).get("mccabe"))
    limit = mccabe.get("max-complexity")
    return limit if isinstance(limit, int) and not isinstance(limit, bool) else None


def coverage_fail_under(snapshot: Snapshot) -> float | None:
    report = as_mapping(pyproject_tool(snapshot, "coverage").get("report"))
    value = report.get("fail_under")
    if isinstance(value, (int, float)) and not isinstance(value, bool):
        return float(value)
    coveragerc = project_file(snapshot, ".coveragerc")
    file = snapshot.files.get(coveragerc) if coveragerc is not None else None
    if file is not None:
        match = re.search(r"fail_under\s*=\s*(\d+(?:\.\d+)?)", file.content)
        if match is not None:
            return float(match.group(1))
    return None


def builds_package(snapshot: Snapshot) -> bool:
    return any(
        "build-system" in pyproject and "project" in pyproject
        for pyproject in project_pyprojects(snapshot)
    )


def has_typed_marker(snapshot: Snapshot) -> bool:
    return any(path.endswith("/py.typed") or path == "py.typed" for path in snapshot.files)


def as_mapping(value: object) -> Mapping[str, object]:
    if isinstance(value, dict):
        return {str(key): item for key, item in value.items()}
    return {}
