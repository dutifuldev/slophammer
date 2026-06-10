---
title: TypeScript Implementation
author: Bob <dutifulbob@gmail.com>
date: 2026-05-16
status: completed
---

# TypeScript Implementation

The TypeScript implementation should mirror the Go implementation's product
contract and architecture, while using TypeScript-native tooling for type
checking, linting, coverage, complexity, mutation testing, and dependency
boundaries. Duplication detection should be native to Slophammer.

The goal is not a line-for-line port. The TypeScript implementation should be a
fully standalone implementation with the same Slophammer behavior: scan a
repository, evaluate enforceable quality rules, render stable reports, and
provide a clean reference implementation that agents can copy.

## Current Status

The first TypeScript implementation is now in `typescript/`.

It includes the shared checker, TypeScript static rules, config validation,
native copied-block DRY detection, execute-mode package checks, fixture
equivalence against the Go implementation, and CI coverage. The implementation
also self-checks the repository root so the reference repo proves its own
declared TypeScript guardrails.

## Product Contract

The TypeScript implementation should support the shared commands through the
`slophammer-ts` public command:

```sh
slophammer-ts check <path>
slophammer-ts check <path> --format json
slophammer-ts check <path> --format sarif
slophammer-ts check <path> --execute
slophammer-ts explain <rule-id>
```

The TypeScript implementation should also include direct language quality
commands where Slophammer owns the check:

```sh
slophammer-ts dry <path>
slophammer-ts dry <path> --max-findings n
slophammer-ts dry <path> --show-report
slophammer-ts dry <path> --format json
slophammer-ts dry <path> --format text
```

It should use the same exit-code model:

- `0`: no findings
- `1`: findings
- `2`: infrastructure or usage error

It should use the same finding shape:

```json
{
  "rule_id": "ts.strict-required",
  "severity": "error",
  "path": "tsconfig.json",
  "message": "TypeScript projects must enable strict mode"
}
```

## Package Layout

Use the same boundaries as Go:

```text
typescript/
├── src/
│   ├── app/
│   ├── cli/
│   ├── config/
│   ├── dry/
│   ├── report/
│   ├── repo/
│   ├── rules/
│   ├── scan/
│   └── toolchecks/
└── tests/
```

Responsibilities:

- `cli`: parse arguments and map them to app calls.
- `app`: coordinate scanning, config loading, rule execution, and reports.
- `config`: parse and validate `slophammer.yml`.
- `dry`: native copied-block duplicate detection.
- `report`: render text, JSON, and SARIF.
- `repo`: typed repository snapshot and file helpers.
- `rules`: pure rule checks with no filesystem or process side effects.
- `scan`: filesystem traversal and text-file loading.
- `toolchecks`: subprocess wrappers for TypeScript quality tools.

The rule engine should be ordinary typed code. It should not shell out, read the
filesystem, or know about GitHub Actions.

## Type Safety Policy

Strict TypeScript is non-negotiable.

`tsconfig.json` should require:

```json
{
  "compilerOptions": {
    "strict": true,
    "noImplicitAny": true,
    "noImplicitOverride": true,
    "noUncheckedIndexedAccess": true,
    "exactOptionalPropertyTypes": true,
    "noFallthroughCasesInSwitch": true,
    "noPropertyAccessFromIndexSignature": true,
    "useUnknownInCatchVariables": true,
    "noEmitOnError": true
  }
}
```

Rules:

- Do not use `any`.
- Do not allow `@ts-ignore`.
- Allow `@ts-expect-error` only with a short explanation.
- Use `unknown` for external input.
- Validate external input before converting it to domain types.
- Avoid unchecked casts. If a cast is required, keep it at a boundary and test
  the validator that protects it.
- Prefer discriminated unions over stringly typed status fields.
- Prefer `type` aliases for data shapes unless a class has behavior and state.

The closest TypeScript equivalent of Go's `interface{}` risk is `any`. It should
not appear in production source. `unknown` is the right boundary type.

## Lint Policy

Use ESLint with typed TypeScript rules.

Baseline rules:

- `@typescript-eslint/no-explicit-any`
- `@typescript-eslint/no-unsafe-assignment`
- `@typescript-eslint/no-unsafe-call`
- `@typescript-eslint/no-unsafe-member-access`
- `@typescript-eslint/no-unsafe-return`
- `@typescript-eslint/no-floating-promises`
- `@typescript-eslint/no-misused-promises`
- `@typescript-eslint/switch-exhaustiveness-check`
- `@typescript-eslint/consistent-type-definitions`
- `complexity`
- `max-lines`
- `max-lines-per-function`

Recommended targets:

- production file length: `800` lines
- production function length: `80` lines
- cyclomatic complexity: `8`

Test files may have separate limits for fixture-heavy cases, but tests should
not become dumping grounds for untyped helpers.

## TypeScript Rule Set

Add TypeScript-specific rules with stable IDs:

| Rule ID                               | Meaning                                            |
| ------------------------------------- | -------------------------------------------------- |
| `ts.package-required`                 | A TypeScript project should have `package.json`.   |
| `ts.typecheck-required`               | CI should run `tsc --noEmit`.                      |
| `ts.strict-required`                  | `tsconfig.json` should enable strict mode.         |
| `ts.no-explicit-any`                  | ESLint should reject explicit `any`.               |
| `ts.no-unsafe-types`                  | ESLint should reject unsafe type operations.        |
| `ts.lint-required`                    | CI should run ESLint.                              |
| `ts.format-required`                  | CI should run a formatter check.                   |
| `ts.test-required`                    | CI should run the test suite.                      |
| `ts.coverage-required`                | CI should enforce coverage.                        |
| `ts.complexity-required`              | CI should enforce complexity limits.               |
| `ts.dry-required`                     | CI should run duplication detection.               |
| `ts.mutation-required`                | CI should run or schedule mutation testing.         |
| `ts.dependency-boundaries-required`   | Imports should obey declared boundaries.           |

The first implementation slice should add the shared repo rules first, then the
TypeScript rules.

## Tooling Choices

Use existing tools before adding native algorithms, except where Slophammer has
decided to own the behavior.

Recommended tool mapping:

| Concern               | Tooling                                        |
| --------------------- | ---------------------------------------------- |
| Type checking         | `tsc --noEmit`                                 |
| Linting               | ESLint with `typescript-eslint`                |
| Formatting            | Prettier                                      |
| Tests                 | Vitest                                        |
| Coverage              | Vitest coverage with V8 provider              |
| Complexity            | ESLint `complexity`, `max-lines`, function size |
| Duplication           | Slophammer native CPD-style detector          |
| Mutation testing      | StrykerJS                                     |
| Dependency boundaries | dependency-cruiser, or native Slophammer checks |

Slophammer should orchestrate these tools and verify their configuration. It
should not rewrite type checking, linting, coverage, or mutation testing.

Duplication is the exception. Slophammer should own the TypeScript copied-block
detector so the Go and TypeScript implementations can expose one native DRY
report model.

## Native DRY Engine

The TypeScript DRY engine should absorb the CPD idea, not copy PMD source code.
The implementation should be clean TypeScript that follows the same algorithmic
shape:

1. Scan configured TypeScript and JavaScript source files.
2. Tokenize source code.
3. Ignore comments and formatting.
4. Preserve identifiers and literal values.
5. Build fixed-size token windows.
6. Find repeated token windows.
7. Expand matches to the largest useful copied range.
8. Collapse overlapping duplicate reports.
9. Return stable `copied-block` findings.

The report shape should match the Go DRY report:

```json
{
  "findings": [
    {
      "kind": "copied-block",
      "left": {"path": "src/a.ts", "start_line": 12, "end_line": 40},
      "right": {"path": "src/b.ts", "start_line": 18, "end_line": 46},
      "tokens": 120,
      "engine": "token-window"
    }
  ],
  "groups": []
}
```

Default policy:

```yaml
typescript:
  dry:
    max_findings: 0
    copied_blocks:
      enabled: true
      min_tokens: 100
```

Do not vendor PMD CPD or `jscpd` source into Slophammer unless there is a
separate license review and attribution plan. The normal path is to implement
the CPD-style algorithm directly.

## Config Shape

Extend `slophammer.yml` with a TypeScript section:

```yaml
typescript:
  coverage_threshold: 85
  complexity_max: 8
  dry:
    max_findings: 0
    paths:
      - src
    exclude:
      - "**/*.test.ts"
      - "**/*.spec.ts"
      - "fixtures/**"
      - "dist/**"
      - "coverage/**"
    copied_blocks:
      enabled: true
      min_tokens: 100
  mutation_targets:
    - src/rules/rules.ts
  dependency_boundaries:
    - from: src/rules
      allow:
        - src/repo
        - src/config
```

Hard defaults:

- coverage threshold: at least `85`
- complexity maximum: at most `8`
- production duplication findings: `0`

Projects may choose stricter values. They should not choose weaker values.

## Static Checks

Static rules should inspect repository files and command declarations.

Examples:

- `ts.strict-required` reads `tsconfig.json` and requires strict mode.
- `ts.no-explicit-any` reads ESLint config and requires `no-explicit-any`.
- `ts.typecheck-required` checks CI/scripts for `tsc --noEmit`.
- `ts.lint-required` checks CI/scripts for ESLint.
- `ts.test-required` checks CI/scripts for Vitest or another test command.
- `ts.coverage-required` checks CI/scripts for coverage with a threshold.
- `ts.dry-required` checks CI/scripts for `slophammer-ts dry`.
- `ts.mutation-required` checks CI/scripts for StrykerJS or an accepted
  scheduled/manual mutation workflow.

Static checks should accept commands in package scripts, CI workflows, and
repo-local scripts.

## Execute Mode

`check --execute` should run configured tool checks and fold failures into the
normal report.

For TypeScript, execute mode should run:

```sh
npm run format
npm run lint
npm run typecheck
npm test
npm run coverage
npm run complexity
slophammer-ts dry .
npm run mutate -- --dryRunOnly
```

The exact commands should come from config or package scripts. Slophammer should
not assume every TypeScript repo uses the same package manager.

Support package managers in this order:

1. use the lockfile if exactly one exists
2. `pnpm-lock.yaml` -> `pnpm`
3. `yarn.lock` -> `yarn`
4. `package-lock.json` -> `npm`
5. no lockfile -> `npm`

## Implementation Slices

### Slice 1: Shared Checker

- Add `typescript/src` structure.
- Implement typed repo snapshots.
- Implement scanner.
- Implement shared rules:
  - `repo.readme-required`
  - `repo.agents-required`
  - `repo.ci-required`
- Implement text and JSON reports.
- Implement CLI parsing and exit codes.
- Pass shared fixtures.

### Slice 2: TypeScript Static Rules

- Detect TypeScript projects.
- Add `ts.package-required`.
- Add `ts.strict-required`.
- Add `ts.typecheck-required`.
- Add `ts.no-explicit-any`.
- Add `ts.no-unsafe-types`.
- Add `ts.lint-required`.
- Add `ts.test-required`.
- Add fixtures for clean and failing TypeScript repos.

### Slice 3: Quality Budgets

- Parse `typescript` config from `slophammer.yml`.
- Validate hard defaults.
- Add coverage, complexity, DRY, mutation, and dependency-boundary rules.
- Implement the native CPD-style DRY engine.
- Add direct `slophammer-ts dry` command support.
- Add expected reports for each missing gate.

### Slice 4: Execute Mode

- Add package-manager detection.
- Add typed subprocess runner boundary.
- Run configured TypeScript tool checks.
- Convert tool failures into Slophammer findings.
- Keep infrastructure failures as exit code `2`.

### Slice 5: CI And Parity

- Add TypeScript implementation CI.
- Run formatter, lint, typecheck, tests, coverage, and self-checks.
- Verify the TypeScript implementation matches shared fixtures.
- Verify direct TypeScript DRY findings on copied-block fixtures.
- Keep the Go implementation green.

### Slice 6: Cross-Implementation Equivalence

- Add shared fixtures that every implementation must run.
- Add a verifier that runs Go and TypeScript implementations against the same
  shared fixtures.
- Compare JSON reports after stable sorting.
- Require matching rule IDs, severities, paths, and messages for shared rules.
- Keep language-specific expected reports separate.

Shared fixtures should cover:

- clean repository
- missing `README.md`
- missing `AGENTS.md`
- missing CI
- invalid config
- report format behavior
- exit-code behavior

TypeScript-specific fixtures should cover:

- clean TypeScript project
- missing `package.json`
- missing `tsconfig.json`
- weak `tsconfig.json`
- missing typecheck command
- missing no-`any` lint rule
- missing unsafe-type lint rules
- missing tests
- missing coverage gate
- missing complexity gate
- missing DRY check
- copied TypeScript blocks

## Acceptance Criteria

The TypeScript implementation is ready when:

- shared commands work
- text, JSON, and SARIF reports work
- shared repo fixtures pass
- TypeScript clean and failing fixtures pass
- native TypeScript copied-block detection works
- strict TypeScript is enforced
- explicit `any` and unsafe operations are rejected
- coverage threshold cannot be configured below `85`
- complexity maximum cannot be configured above `8`
- duplication budget defaults to `0`
- execute mode can run configured TypeScript tool checks
- shared fixtures prove Go and TypeScript report equivalence
- CI runs the TypeScript implementation checks
- Slophammer's full repo CI remains green

## Local Validation

Expected local commands:

```sh
cd typescript
npm install
npm run format
npm run lint
npm run typecheck
npm test
npm run coverage
npm run check
npm run self-check
```

From the repository root:

```sh
npx -y @simpledoc/simpledoc check
```

The Go implementation should remain green while the TypeScript implementation is
added.
