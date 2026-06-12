"""Repository snapshot and binding command evidence.

Ported function-by-function from ``typescript/src/repo/repo.ts`` so behavior
differences are diffable line-by-line. Workflows contribute only run scripts
from steps that can execute and can fail on integration branches; scripts,
runner files, and package scripts are credited only when binding workflow
evidence reaches them.
"""

from __future__ import annotations

import json
import re
from collections.abc import Mapping
from dataclasses import dataclass

import yaml

MAX_WORD_CHARACTER = re.compile(r"[A-Za-z0-9_-]")
SYNTHETIC_REPO_EVIDENCE_PREFIX = "scripts/__repo_"
RUNNER_FILE_NAMES = {"makefile", "taskfile.yml", "taskfile.yaml", "justfile"}
RUNNER_COMMANDS = {
    "makefile": "make",
    "taskfile.yml": "task",
    "taskfile.yaml": "task",
    "justfile": "just",
}


@dataclass(frozen=True)
class RepoFile:
    path: str
    content: str


@dataclass(frozen=True)
class Snapshot:
    root: str
    files: Mapping[str, RepoFile]


def new_snapshot(root: str, files: list[RepoFile]) -> Snapshot:
    by_path = {
        normalize_path(file.path): RepoFile(normalize_path(file.path), file.content)
        for file in files
    }
    return Snapshot(root=root, files=dict(sorted(by_path.items())))


def has_file(snapshot: Snapshot, path: str) -> bool:
    return normalize_path(path) in snapshot.files


def has_file_named(snapshot: Snapshot, *names: str) -> bool:
    wanted = {name.lower() for name in names}
    return any(file_path.rsplit("/", 1)[-1].lower() in wanted for file_path in snapshot.files)


def files_with_suffix(snapshot: Snapshot, suffix: str) -> list[RepoFile]:
    return [file for file in snapshot.files.values() if file.path.endswith(suffix)]


def files_named(snapshot: Snapshot, *names: str) -> list[RepoFile]:
    wanted = {name.lower() for name in names}
    return [
        file for file in snapshot.files.values() if file.path.rsplit("/", 1)[-1].lower() in wanted
    ]


def workflow_files(snapshot: Snapshot) -> list[RepoFile]:
    return [
        file
        for file in snapshot.files.values()
        if len(file.path.split("/")) == 3
        and file.path.startswith(".github/workflows/")
        and (file.path.endswith(".yml") or file.path.endswith(".yaml"))
    ]


def command_files(snapshot: Snapshot) -> list[RepoFile]:
    workflows = [
        file
        for file in (workflow_command_file(workflow) for workflow in workflow_files(snapshot))
        if not ignored_evidence_path(file.path) and file.content.strip() != ""
    ]
    synthetic = synthetic_repo_evidence_files(snapshot)
    root_evidence = joined_contents(workflows + synthetic)
    scripts = reachable_script_files(snapshot, root_evidence)
    runners = reachable_runner_files(snapshot, root_evidence)
    packages = reachable_package_script_files(snapshot, root_evidence + joined_contents(scripts))
    return [
        file
        for file in workflows + synthetic + scripts + runners + packages
        if not ignored_evidence_path(file.path) and file.content.strip() != ""
    ]


def synthetic_repo_evidence_path(file_path: str) -> bool:
    return file_path.startswith(SYNTHETIC_REPO_EVIDENCE_PREFIX)


def synthetic_repo_evidence_files(snapshot: Snapshot) -> list[RepoFile]:
    return [
        shell_command_file(file)
        for file in snapshot.files.values()
        if synthetic_repo_evidence_path(file.path)
    ]


def joined_contents(files: list[RepoFile]) -> str:
    return "".join(file.content + "\n" for file in files)


def reachable_script_files(snapshot: Snapshot, root_evidence: str) -> list[RepoFile]:
    candidates = [shell_command_file(file) for file in script_files(snapshot)]
    first_hop = [file for file in candidates if references_file(root_evidence, file.path)]
    extended = root_evidence + joined_contents(first_hop)
    return [file for file in candidates if references_file(extended, file.path)]


def reachable_runner_files(snapshot: Snapshot, root_evidence: str) -> list[RepoFile]:
    runners = [
        file
        for file in snapshot.files.values()
        if file.path.rsplit("/", 1)[-1].lower() in RUNNER_FILE_NAMES
    ]
    return [
        shell_command_file(file)
        for file in runners
        if contains_word(root_evidence, RUNNER_COMMANDS[file.path.rsplit("/", 1)[-1].lower()])
    ]


def references_file(evidence: str, file_path: str) -> bool:
    base_name = file_path.rsplit("/", 1)[-1]
    return contains_word(evidence, base_name)


def contains_word(evidence: str, word: str) -> bool:
    index = evidence.find(word)
    while index >= 0:
        before = evidence[index - 1] if index > 0 else ""
        after = evidence[index + len(word) : index + len(word) + 1]
        if not word_character(before) and not word_character(after):
            return True
        index = evidence.find(word, index + 1)
    return False


def word_character(value: str) -> bool:
    return bool(MAX_WORD_CHARACTER.fullmatch(value))


def reachable_package_script_files(snapshot: Snapshot, evidence: str) -> list[RepoFile]:
    return [
        shell_command_file(
            RepoFile(file.path, reachable_package_script_content(file.content, evidence))
        )
        for file in files_named(snapshot, "package.json")
    ]


def reachable_package_script_content(content: str, evidence: str) -> str:
    scripts = parsed_package_scripts(content)
    invoked = {name for name in scripts if script_invoked(evidence, name)}
    chained = {
        candidate
        for name in invoked
        for candidate in scripts
        if script_invoked(scripts[name], candidate)
    }
    invoked |= chained
    return "\n".join(f"{name}: {value}" for name, value in scripts.items() if name in invoked)


def parsed_package_scripts(content: str) -> dict[str, str]:
    try:
        parsed = json.loads(content)
    except ValueError:
        return {}
    scripts = parsed.get("scripts") if isinstance(parsed, dict) else None
    if not isinstance(scripts, dict):
        return {}
    return {name: value for name, value in scripts.items() if isinstance(value, str)}


def script_invoked(evidence: str, name: str) -> bool:
    escaped = re.escape(name)
    runner = re.compile(rf"\b(?:npm|pnpm|yarn|bun)(?:\s+run)?\s+(?:-[\w-]+\s+)*{escaped}(?![\w-])")
    if runner.search(evidence):
        return True
    return name == "test" and re.search(r"\b(?:npm|pnpm|yarn|bun)\s+test\b", evidence) is not None


def normalize_path(path: str) -> str:
    return re.sub(r"^\./+", "", path.replace("\\", "/"))


def script_files(snapshot: Snapshot) -> list[RepoFile]:
    return [
        file
        for file in snapshot.files.values()
        if not synthetic_repo_evidence_path(file.path)
        and (
            file.path.startswith("scripts/")
            or "/scripts/" in file.path
            or file.path.endswith(".sh")
        )
    ]


def workflow_command_file(file: RepoFile) -> RepoFile:
    return RepoFile(file.path, strip_comment_lines(extract_workflow_run_content(file.content)))


def shell_command_file(file: RepoFile) -> RepoFile:
    return RepoFile(file.path, strip_comment_lines(file.content.replace("\\\n", " ")))


def extract_workflow_run_content(content: str) -> str:
    workflow = workflow_record(content)
    if workflow is not None and as_record(workflow.get("jobs")):
        if binding_workflow_triggers(workflow.get("on", workflow.get(True))):
            return "\n".join(workflow_commands(workflow))
        return ""
    return "\n".join(line_based_run_commands(content))


def line_based_run_commands(content: str) -> list[str]:
    lines = content.replace("\r\n", "\n").split("\n")
    commands: list[str] = []
    index = 0
    while index < len(lines):
        entry = workflow_run_entry(lines[index])
        if entry is None:
            index += 1
            continue
        indent, value = entry
        if block_scalar(value):
            block, next_index = collect_indented_block(lines, index + 1, indent)
            commands.append(block)
            index = next_index
            continue
        command = inline_run_command(value)
        if command is not None:
            commands.append(command)
        index += 1
    return commands


MATRIX_COMMAND_EXPRESSION = re.compile(r"\$\{\{\s*matrix\.command\s*\}\}")


def workflow_commands(workflow: Mapping[str, object]) -> list[str]:
    return [
        command
        for job in as_record(workflow.get("jobs")).values()
        for command in job_workflow_commands(job)
    ]


def job_workflow_commands(job: object) -> list[str]:
    record = as_record(job)
    if neutralized_entry(record):
        return []
    matrix_commands = job_matrix_commands(record)
    return [
        command
        for step in array_values(record.get("steps"))
        for command in step_workflow_commands(step, matrix_commands)
    ]


def step_workflow_commands(step: object, matrix_commands: list[str]) -> list[str]:
    record = as_record(step)
    if neutralized_entry(record):
        return []
    evidence: list[str] = []
    uses = string_value(record.get("uses"))
    if uses != "":
        evidence.append(f"uses: {uses}")
    command = string_value(record.get("run"))
    if command == "":
        return evidence
    if not direct_matrix_command(command) or not matrix_commands:
        return [*evidence, command]
    return [*evidence, *matrix_commands]


def neutralized_entry(record: Mapping[str, object]) -> bool:
    continue_on_error = record.get("continue-on-error")
    if (
        continue_on_error is True
        or literal_expression_value(string_value(continue_on_error)) == "true"
    ):
        return True
    condition = record.get("if")
    if condition is False:
        return True
    return literal_expression_value(string_value(condition)) == "false"


def literal_expression_value(value: str) -> str:
    trimmed = value.strip()
    if trimmed.startswith("${{") and trimmed.endswith("}}"):
        trimmed = trimmed[3:-2].strip()
    return trimmed


def binding_workflow_triggers(on: object) -> bool:
    if isinstance(on, str):
        return binding_trigger_name(on)
    if isinstance(on, list):
        return any(binding_trigger_name(string_value(name)) for name in on)
    record = as_record(on)
    return any(binding_trigger_entry(str(name), value) for name, value in record.items())


def binding_trigger_name(name: str) -> bool:
    return name in {"push", "pull_request", "pull_request_target", "merge_group", "schedule"}


def binding_trigger_entry(name: str, value: object) -> bool:
    if name in {"pull_request", "pull_request_target", "merge_group", "schedule"}:
        return True
    if name != "push":
        return False
    record = as_record(value)
    branches = record.get("branches")
    if branches is None:
        # Defining only tags or tags-ignore stops the workflow from firing
        # for branch pushes entirely, so it is a release trigger, not
        # integration CI; a branches-ignore filter still fires for branches.
        if "branches-ignore" in record:
            return True
        return "tags" not in record and "tags-ignore" not in record
    patterns = (
        [string_value(branch) for branch in branches]
        if isinstance(branches, list)
        else [string_value(branches)]
    )
    return any(integration_branch_pattern(pattern) for pattern in patterns)


def integration_branch_pattern(pattern: str) -> bool:
    if "*" in pattern:
        return True
    return pattern in {"main", "master", "trunk", "develop"}


def direct_matrix_command(command: str) -> bool:
    return (
        MATRIX_COMMAND_EXPRESSION.search(command) is not None
        and MATRIX_COMMAND_EXPRESSION.sub("", command).strip() == ""
    )


RUN_ENTRY_PATTERN = re.compile(r"^(\s*)(?:-\s*)?run:\s*(.*)$")


def workflow_run_entry(line: str) -> tuple[int, str] | None:
    match = RUN_ENTRY_PATTERN.match(line)
    if match is None:
        return None
    return len(match.group(1)), match.group(2).rstrip()


def workflow_record(content: str) -> Mapping[str, object] | None:
    try:
        parsed = yaml.safe_load(content)
    except yaml.YAMLError:
        return None
    if not isinstance(parsed, dict):
        return None
    return parsed


def job_matrix_commands(job: Mapping[str, object]) -> list[str]:
    matrix = as_record(as_record(job.get("strategy")).get("matrix"))
    include_commands = [
        command
        for item in array_values(matrix.get("include"))
        if (command := string_value(as_record(item).get("command"))) != ""
    ]
    direct_commands = [
        command
        for value in array_values(matrix.get("command"))
        if (command := string_value(value)) != ""
    ]
    return include_commands + direct_commands


def inline_run_command(value: str) -> str | None:
    command = value.strip()
    return None if command == "" else unquote(command)


def block_scalar(value: str) -> bool:
    trimmed = value.strip()
    return trimmed.startswith("|") or trimmed.startswith(">")


def collect_indented_block(
    lines: list[str], start_index: int, parent_indent: int
) -> tuple[str, int]:
    kept: list[str] = []
    block_indent: int | None = None
    index = start_index
    while index < len(lines):
        line = lines[index]
        if line.strip() == "":
            kept.append("")
            index += 1
            continue
        indent = leading_spaces(line)
        if indent <= parent_indent:
            break
        if block_indent is None:
            block_indent = indent
        if indent < block_indent:
            break
        kept.append(line[block_indent:])
        index += 1
    return "\n".join(kept), index


def leading_spaces(line: str) -> int:
    return len(line) - len(line.lstrip(" \t"))


def unquote(value: str) -> str:
    if (value.startswith('"') and value.endswith('"')) or (
        value.startswith("'") and value.endswith("'")
    ):
        return value[1:-1]
    return value


def strip_comment_lines(content: str) -> str:
    lines = (line.split("#", 1)[0] for line in content.split("\n"))
    return "\n".join(line for line in lines if line.strip() != "")


def ignored_evidence_path(file_path: str) -> bool:
    return file_path.startswith("fixtures/") or file_path.startswith("templates/")


def as_record(value: object) -> Mapping[str, object]:
    if isinstance(value, dict):
        return {str(key): item for key, item in value.items()}
    return {}


def array_values(value: object) -> list[object]:
    return list(value) if isinstance(value, list) else []


def string_value(value: object) -> str:
    return value if isinstance(value, str) else ""
