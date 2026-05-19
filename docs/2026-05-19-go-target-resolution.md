---
title: Go Target Resolution
author: Bob <dutifulbob@gmail.com>
date: 2026-05-19
---

# Go Target Resolution

## Problem

Several Slophammer checks need the same basic answer: which Go source files are
in scope?

Mutation scanning is one example, but the same problem can apply to formatting,
linting, coverage, generated-code checks, import checks, or future Go-specific
rules. Repositories should not need to encode source discovery separately for
each check.

The desired repository-level policy is simple:

```yaml
go:
  targets:
    - internal
    - cmd
  exclude:
    - "**/*_test.go"
    - "**/testdata/**"
    - "**/fixtures/**"
```

That says: these are the Go source trees Slophammer should consider by default.

Today, a check can become too file-oriented and force a repository to write a
wrapper script that discovers files, filters them, sorts them, and loops over
tool invocations. Static validation can then fail to recognize that wrapper
unless it sees a direct tool command such as:

```bash
mutate4go internal/foo.go --scan
```

That pushes Slophammer policy mechanics into repository scripts. The cleaner
boundary is for repositories to declare source scope and for Slophammer to
resolve and enforce that scope consistently.

## Goal

Slophammer should provide a shared Go target resolver.

The desired repository contract is:

```yaml
go:
  targets:
    - internal
    - cmd
  exclude:
    - "**/*_test.go"
    - "**/testdata/**"
    - "**/fixtures/**"
```

Checks should consume that resolved target set by default:

```bash
slophammer go mutate --scan
```

Mutation scanning would discover the concrete files, run `mutate4go` on each
file, and make the static mutation-required rule accept the Slophammer command
as valid coverage.

A check may still override the shared target set when it has a legitimate
different scope:

```yaml
go:
  targets:
    - .
  mutation:
    targets:
      - internal/rules
    exclude:
      - "internal/rules/generated/**"
```

The shared target resolver remains the primitive. Check-specific targets are an
override, not a separate discovery mechanism.

## Semantics

Each target entry is either:

- a `.go` file
- a directory containing Go files

File targets resolve directly.

Directory targets are expanded recursively to production `.go` files.

The default production filter should exclude:

- `*_test.go`
- any path segment named `testdata`
- any path segment named `fixtures`
- hidden VCS/build directories such as `.git`, `target`, and `node_modules`

Configured exclude patterns are applied after the defaults.

The final target list must be sorted before execution so output is stable across
machines.

If expansion produces zero files, the command must fail. Silent success would be
misleading.

## Configuration Shape

Shared Go target configuration:

```yaml
go:
  targets:
    - internal
    - cmd/server/main.go
  exclude:
    - "internal/generated/**"
```

Check-specific override:

```yaml
go:
  targets:
    - .
  mutation:
    targets:
      - internal/rules
    exclude:
      - "internal/rules/generated/**"
```

Resolution order:

1. If the check has explicit targets, use those targets plus check-specific
   excludes.
2. Otherwise, use `go.targets` plus `go.exclude`.
3. Always apply default production Go excludes.
4. Sort the final result.
5. Fail if the result is empty.

The resolver should expose the resolved list to commands and rules without
duplicating filesystem traversal.

## Mutation Behavior

For:

```bash
slophammer go mutate --scan
```

Slophammer should:

1. Load `slophammer.yml`.
2. Resolve Go targets using the shared resolver.
3. Print the resolved count.
4. Run `mutate4go <file> --scan` for each resolved file.
5. Return non-zero if any scan fails.

Example output:

```text
Go mutation targets: 143 production files
Mutation scan: internal/rules/rules.go
Total mutation sites: 42
Changed mutation sites: 7
Manifest exists: true
```

## Static Rule Behavior

The `go.mutation-required` rule should accept either of these as valid:

- a direct `mutate4go <file.go>` command
- a config-backed `slophammer go mutate` command where the shared or
  mutation-specific Go targets expand to at least one file

It should not require a direct `mutate4go some-file.go` line when Slophammer
itself is doing target expansion.

This removes the need for placeholder one-file mutation scans in repository
scripts.

## Implementation Notes

The implementation should live in Slophammer's Go support code, not inside
repository fixtures or generated scripts.

Suggested pieces:

- Add shared Go config fields for `targets` and `exclude`.
- Add optional check-specific target/exclude overrides for mutation.
- Add a resolver that takes repository root, target strings, and excludes.
- Unit-test expansion for:
  - single file target
  - directory target
  - nested directory target
  - `*_test.go` exclusion
  - `testdata` and `fixtures` exclusion
  - configured exclude
  - deterministic sorting
  - zero-match failure
- Update `slophammer go mutate` to call the resolver before invoking
  `mutate4go`.
- Update `go.mutation-required` so config-backed directory targets satisfy the
  rule.

The resolver should be pure and independently tested. The command layer should
only handle config loading, printing, and process execution.

## Non-Goals

This does not require full mutation execution on every pull request. Scan mode is
still acceptable for normal CI.

This does not require a global mutation score threshold. The first step is
correct target declaration and reliable discovery.

This does not force every Go check to use exactly the same scope. Check-specific
overrides remain valid when the difference is explicit and justified.

## Acceptance Criteria

- A repo can declare shared Go source targets in `slophammer.yml`.
- Targets may be files or directories.
- Default excludes prevent test files and fixture data from being scanned.
- Configured excludes work.
- The resolved target list is deterministic.
- Commands fail on zero resolved targets.
- Mutation scan uses the shared Go target resolver by default.
- Mutation scan can override the shared target set when configured.
- `go.mutation-required` passes for a config-backed Slophammer mutation command.
- No repository needs a fake or placeholder one-file mutation scan solely to
  satisfy the static rule.
