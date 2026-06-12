"""Native copied-block DRY engine: a token-window duplicate detector over
Python source, ported from typescript/src/dry. Identifiers and literals
keep their text in the token tag, so only genuinely identical code matches;
whitespace, comments, and layout tokens are invisible.
"""

from __future__ import annotations

import io
import itertools
import token as token_types
import tokenize as tokenizer
from dataclasses import dataclass

from slophammer_py.config import Config
from slophammer_py.repo import RepoFile, Snapshot
from slophammer_py.rules.scope import excluded, in_targets, production_python_files

SKIPPED_TOKEN_TYPES = {
    token_types.NEWLINE,
    token_types.NL,
    token_types.INDENT,
    token_types.DEDENT,
    token_types.COMMENT,
    token_types.ENCODING,
    token_types.ENDMARKER,
}


@dataclass(frozen=True)
class TokenEntry:
    tag: str
    file: str
    line: int


@dataclass(frozen=True)
class SourceRange:
    path: str
    start_line: int
    end_line: int


@dataclass(frozen=True)
class DryFinding:
    left: SourceRange
    right: SourceRange
    tokens: int

    def json_value(self) -> dict[str, object]:
        return {
            "kind": "copied-block",
            "left": range_json(self.left),
            "right": range_json(self.right),
            "tokens": self.tokens,
            "engine": "token-window",
        }


def range_json(value: SourceRange) -> dict[str, object]:
    return {"path": value.path, "start_line": value.start_line, "end_line": value.end_line}


def dry_findings(snapshot: Snapshot, config: Config) -> list[DryFinding]:
    if not copied_blocks_enabled(config):
        return []
    files = dry_source_files(snapshot, config)
    window = min_tokens(config)
    return find_copied_blocks(files, window)


def copied_blocks_enabled(config: Config) -> bool:
    dry = config.python.dry if config.python is not None else None
    return dry.copied_blocks.enabled if dry is not None else True


def dry_source_files(snapshot: Snapshot, config: Config) -> list[RepoFile]:
    dry = config.python.dry if config.python is not None else None
    paths = dry.paths if dry is not None else []
    patterns = [entry.pattern for entry in dry.exclude] if dry is not None else []
    production = set(production_python_files(snapshot))
    return [
        file
        for file in snapshot.files.values()
        if file.path in production
        and in_targets(file.path, paths)
        and not excluded(file.path, patterns)
    ]


def min_tokens(config: Config) -> int:
    dry = config.python.dry if config.python is not None else None
    return dry.copied_blocks.min_tokens if dry is not None else 100


def max_findings(config: Config) -> int:
    dry = config.python.dry if config.python is not None else None
    return dry.max_findings if dry is not None else 0


def tokenize_file(file: RepoFile) -> list[TokenEntry]:
    tokens: list[TokenEntry] = []
    try:
        for item in tokenizer.generate_tokens(io.StringIO(file.content).readline):
            if item.type in SKIPPED_TOKEN_TYPES:
                continue
            tokens.append(TokenEntry(tag=token_tag(item), file=file.path, line=item.start[0]))
    except (tokenizer.TokenError, SyntaxError, ValueError):
        return tokens
    return tokens


def token_tag(item: tokenizer.TokenInfo) -> str:
    if item.type == token_types.NAME:
        return f"ident/{item.string}"
    if item.type in (token_types.NUMBER, token_types.STRING, token_types.FSTRING_MIDDLE):
        return f"literal/{token_types.tok_name[item.type]}/{item.string}"
    return f"{token_types.tok_name[item.type]}/{item.string}"


def find_copied_blocks(files: list[RepoFile], window_size: int) -> list[DryFinding]:
    sequences = [tokenize_file(file) for file in sorted(files, key=lambda file: file.path)]
    matches = copied_block_matches(sequences, window_size)
    return collapse_overlapping(matches)


def copied_block_matches(sequences: list[list[TokenEntry]], window_size: int) -> list[DryFinding]:
    windows: dict[str, list[tuple[int, int]]] = {}
    for sequence_index, tokens in enumerate(sequences):
        for index in range(len(tokens) - window_size + 1):
            key = "\x00".join(entry.tag for entry in tokens[index : index + window_size])
            windows.setdefault(key, []).append((sequence_index, index))
    matches: list[DryFinding] = []
    for occurrences in windows.values():
        matches.extend(matches_for_occurrences(sequences, occurrences, window_size))
    return matches


def matches_for_occurrences(
    sequences: list[list[TokenEntry]], occurrences: list[tuple[int, int]], window_size: int
) -> list[DryFinding]:
    seen: set[str] = set()
    matches: list[DryFinding] = []
    compacted = compact_occurrences(occurrences, window_size)
    for left, right in itertools.pairwise(compacted):
        match = copied_block_match(sequences, left, right, window_size)
        if match is not None and finding_key(match) not in seen:
            seen.add(finding_key(match))
            matches.append(match)
    return matches


def compact_occurrences(
    occurrences: list[tuple[int, int]], window_size: int
) -> list[tuple[int, int]]:
    last_index: dict[int, int] = {}
    compacted: list[tuple[int, int]] = []
    for sequence, index in sorted(occurrences):
        previous = last_index.get(sequence)
        if previous is not None and index < previous + window_size:
            continue
        compacted.append((sequence, index))
        last_index[sequence] = index
    return compacted


def copied_block_match(
    sequences: list[list[TokenEntry]],
    left: tuple[int, int],
    right: tuple[int, int],
    window_size: int,
) -> DryFinding | None:
    if right < left:
        left, right = right, left
    if left[0] == right[0] and ranges_touch(
        left[1], left[1] + window_size - 1, right[1], right[1] + window_size - 1
    ):
        return None
    return expand_token_match(sequences, left, right, window_size)


def expand_token_match(
    sequences: list[list[TokenEntry]],
    left: tuple[int, int],
    right: tuple[int, int],
    window_size: int,
) -> DryFinding | None:
    left_tokens = sequences[left[0]]
    right_tokens = sequences[right[0]]
    same_sequence = left[0] == right[0]
    left_start, right_start = expand_start(
        left_tokens, right_tokens, left, right, window_size, same_sequence
    )
    left_end, right_end = expand_end(
        left_tokens,
        right_tokens,
        left,
        right,
        (left_start, right_start),
        window_size,
        same_sequence,
    )
    if left_end - left_start < window_size:
        return None
    return DryFinding(
        left=token_range(left_tokens[left_start:left_end]),
        right=token_range(right_tokens[right_start:right_end]),
        tokens=left_end - left_start,
    )


def expand_start(
    left_tokens: list[TokenEntry],
    right_tokens: list[TokenEntry],
    left: tuple[int, int],
    right: tuple[int, int],
    window_size: int,
    same_sequence: bool,
) -> tuple[int, int]:
    left_start, right_start = left[1], right[1]
    while (
        left_start > 0
        and right_start > 0
        and left_tokens[left_start - 1].tag == right_tokens[right_start - 1].tag
    ):
        if same_sequence and ranges_touch(
            left_start - 1, left[1] + window_size - 1, right_start - 1, right[1] + window_size - 1
        ):
            break
        left_start -= 1
        right_start -= 1
    return left_start, right_start


def expand_end(
    left_tokens: list[TokenEntry],
    right_tokens: list[TokenEntry],
    left: tuple[int, int],
    right: tuple[int, int],
    starts: tuple[int, int],
    window_size: int,
    same_sequence: bool,
) -> tuple[int, int]:
    left_end = left[1] + window_size
    right_end = right[1] + window_size
    while (
        left_end < len(left_tokens)
        and right_end < len(right_tokens)
        and left_tokens[left_end].tag == right_tokens[right_end].tag
    ):
        if same_sequence and ranges_touch(starts[0], left_end, starts[1], right_end):
            break
        left_end += 1
        right_end += 1
    return left_end, right_end


def collapse_overlapping(matches: list[DryFinding]) -> list[DryFinding]:
    ordered = sorted(matches, key=lambda match: (-match.tokens, finding_key(match)))
    kept: list[DryFinding] = []
    for match in ordered:
        if not any(overlap_pair(match, existing) for existing in kept):
            kept.append(match)
    return sorted(kept, key=finding_key)


def overlap_pair(left: DryFinding, right: DryFinding) -> bool:
    return (ranges_overlap(left.left, right.left) and ranges_overlap(left.right, right.right)) or (
        ranges_overlap(left.left, right.right) and ranges_overlap(left.right, right.left)
    )


def ranges_overlap(left: SourceRange, right: SourceRange) -> bool:
    if left.path != right.path:
        return False
    return max(left.start_line, right.start_line) <= min(left.end_line, right.end_line)


def ranges_touch(left_start: int, left_end: int, right_start: int, right_end: int) -> bool:
    return max(left_start, right_start) <= min(left_end, right_end)


def token_range(tokens: list[TokenEntry]) -> SourceRange:
    if not tokens:
        return SourceRange(path="", start_line=0, end_line=0)
    return SourceRange(path=tokens[0].file, start_line=tokens[0].line, end_line=tokens[-1].line)


def finding_key(finding: DryFinding) -> str:
    return "|".join(
        str(part)
        for part in (
            finding.left.path,
            finding.left.start_line,
            finding.left.end_line,
            finding.right.path,
            finding.right.start_line,
            finding.right.end_line,
            finding.tokens,
        )
    )
