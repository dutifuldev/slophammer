---
title: Unified DRY Engine Plan
author: Bob <dutifulbob@gmail.com>
date: 2026-05-15
---

# Unified DRY Engine Plan

Slophammer should own one DRY module instead of exposing separate `dry4go` and
CPD-style checks as unrelated tools.

The goal is not to invent a new duplicate-detection theory. The goal is to
combine two existing, useful approaches under one Slophammer DRY report model:

1. Structural function similarity, based on the `dry4go` model.
2. Copied-block detection, based on the CPD model.

The unified DRY module should make one answer visible:

```text
How much duplicate production code does this repository contain,
where is it, and what kind of duplication is it?
```

## Why This Exists

A real mixed production Go codebase comparison showed that the two approaches
catch different problems.

`dry4go` found many Go structural candidates, but it missed large copied blocks
inside bigger handlers and state builders. PMD CPD found those copied blocks.

The measured comparison was:

| Detector                     | Findings |
| ---------------------------- | -------- |
| Slophammer `go dry` / dry4go | 220      |
| PMD CPD Go production scan   | 34       |
| Intersection                 | 8        |
| CPD-only findings            | 26       |

That means the current Go DRY check is useful, but incomplete.

## Product Shape

Use one DRY command:

```sh
slophammer go dry <path>
```

The command should run both engines. The user-facing keyword stays `dry`.

## Detection Engines

### Structural Function Engine

This engine absorbs the `dry4go` behavior.

It should:

- parse Go files with the standard Go parser
- compare Go functions and methods
- normalize away incidental names and literal values
- keep structural syntax such as branches, loops, calls, returns, composite
  literals, operators, parameters, and result shape
- build normalized syntax fingerprints
- compare fingerprint sets with Jaccard similarity
- report candidates above a configured threshold

Default policy:

```yaml
go:
  dry:
    structural:
      enabled: true
      threshold: 0.82
      min_lines: 4
      min_nodes: 20
```

This engine should report whole-function or whole-method candidates.

### Copied Block Engine

This engine absorbs the CPD behavior.

It should:

- tokenize Go source using the standard Go scanner
- preserve identifiers and literal values for copied-block matching
- keep operator, keyword, delimiter, and punctuation shape
- build repeated token-window matches
- expand exact token-window matches to the largest useful copied range
- report copied blocks even when they live inside larger functions
- ignore comments and formatting

Default policy:

```yaml
go:
  dry:
    copied_blocks:
      enabled: true
      min_tokens: 100
```

This engine should report copied source ranges, not whole functions.

## Unified Finding Model

Both engines should return the same internal type:

```go
type DRYFinding struct {
    Kind       DRYFindingKind
    Left       SourceRange
    Right      SourceRange
    Score      float64
    Tokens     int
    Nodes      int
    Engine     string
}
```

The public report should use stable strings:

```text
structural-function
copied-block
```

The JSON shape should be explicit:

```json
{
  "kind": "copied-block",
  "left": {"path": "internal/api/handlers/dispute.go", "start_line": 1199, "end_line": 1234},
  "right": {"path": "internal/api/handlers/dispute.go", "start_line": 1359, "end_line": 1393},
  "tokens": 236,
  "engine": "token-window"
}
```

## Finding Groups

The two engines can report overlapping findings. Slophammer should keep both
signals internally, then group overlaps for human output.

Grouping rules:

- Two findings overlap when they touch the same files and overlapping line
  ranges on both sides.
- A group can contain structural and copied-block findings.
- Text output should show one group with all detector labels.
- JSON output should preserve each raw finding and include a stable group ID.

Example summary:

```text
DRY findings: 246
Groups: 238
Structural function findings: 220
Copied block findings: 34
Found by both: 8
```

## Configuration

Keep existing top-level Go DRY fields working:

```yaml
go:
  dry_max_candidates: 0
  dry_paths:
    - go/cmd
    - go/internal
  dry_exclude:
    - "**/*_test.go"
```

Add the newer nested DRY shape:

```yaml
go:
  dry:
    max_findings: 0
    paths:
      - go/cmd
      - go/internal
    exclude:
      - "**/*_test.go"
      - "fixtures/**"
      - "templates/**"
    structural:
      enabled: true
      threshold: 0.82
      min_lines: 4
      min_nodes: 20
    copied_blocks:
      enabled: true
      min_tokens: 100
```

Mapping rules:

- `dry_max_candidates` maps to `dry.max_findings`.
- `dry_paths` maps to `dry.paths`.
- `dry_exclude` maps to `dry.exclude`.
- The new shape wins when both are present.

## Package Layout

Add a focused internal package:

```text
go/internal/dry/
├── finding.go
├── config.go
├── scan.go
├── structural.go
├── structural_normalize.go
├── copied_blocks.go
├── copied_blocks_tokens.go
├── groups.go
└── report.go
```

Keep the package pure:

- no shell commands
- no filesystem walking beyond the input files it is given
- no GitHub-specific behavior
- no rule-engine dependency

`internal/app` should coordinate repository scanning, config loading, and
DRY execution.

## Compatibility Plan

Keep these commands working:

```sh
slophammer go dry <path>
slophammer go dry <path> --max-candidates 0
slophammer go dry <path> --show-report
```

Add output flags to the same command:

```sh
slophammer go dry <path> --format json
slophammer go dry <path> --format text
```

`go dry` should print the unified summary and keep the existing exit-code
behavior.

## Implementation Steps

- [x] Add `internal/dry` types and config.
- [x] Implement native structural function detection.
- [x] Add tests using small Go fixtures derived from the behavior description,
      not from unlicensed source.
- [x] Implement native copied-block detection.
- [x] Add fixtures for copied blocks inside larger functions.
- [x] Add grouping and overlap detection.
- [x] Add JSON and text report rendering.
- [x] Change `slophammer go dry` to call the unified engine.
- [x] Update `go.dry-required` docs to describe the unified DRY check.
- [x] Add small regression fixtures that capture missed CPD-style cases in a
      repo-safe form.
- [x] Remove direct runtime dependency on `dry4go` after parity checks pass.

## Non-Goals

This plan does not include semantic duplicate detection.

The first version should not try to prove that two different implementations
encode the same business rule. That requires type information, call graphs,
data-flow analysis, and domain knowledge. Add that later only after the
syntax-level engines are stable.

This plan also does not include copying PMD or dry4go source code directly.
If source code is reused, licensing must be explicit first. Otherwise,
Slophammer should reimplement the documented behavior.

## Acceptance Criteria

- `slophammer go dry` reports structural function findings.
- `slophammer go dry` reports copied-block findings.
- `slophammer go dry` remains compatible.
- JSON output distinguishes `structural-function` and `copied-block`.
- Text output groups overlaps instead of spamming duplicate duplicates.
- Production defaults are strict:
  - max findings: `0`
  - structural threshold: `0.82`
  - copied block minimum: `100` tokens
- Tests cover:
  - exact copied blocks
  - copied blocks inside larger functions
  - structurally similar functions with renamed identifiers
  - non-duplicates with shared small syntax
  - excluded tests, fixtures, templates, and generated paths

## Open Questions

- Should the copied-block detector report overlapping clones separately, or
  collapse them aggressively into the largest range?
- Should test files be excluded by default, or should that stay entirely in
  `slophammer.yml`?

## Implementation Notes

The copied-block detector keeps identifiers and literal values as tokens. The
structural engine already handles renamed identifiers and changed literals. The
copied-block engine should behave closer to CPD, which is why it preserves
tokens and collapses overlapping sliding-window matches into the largest
representative copied block.

Parity checks showed:

| Baseline                 | Findings | Covered by native `go dry` |
| ------------------------ | -------- | -------------------------- |
| Existing structural pass | 220      | 220                        |
| PMD CPD production scan  | 34       | 34                         |

The native command also passes Slophammer's own zero-finding production DRY
budget.
