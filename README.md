# Slophammer

Reference implementations and templates for repository quality checks in
agent-assisted software projects.

`slophammer` checks whether a project has the basic constraints needed to keep
AI-generated code under control: agent instructions, CI, tests, strict typing,
linting, coverage, documentation conventions, and project structure that humans
can still review.

The point of this repository is to show a small tool implemented cleanly, with
language templates beside it, so agents can copy patterns from real, working
code.

## What This Is

- A small product spec for a repo quality checker.
- A production Go implementation of that product.
- Go, TypeScript, and Python project templates with strict local checks.
- A reference for project structure, testing, errors, reporting, and CI.
- A source of patterns for agents working in different language ecosystems.

## What This Is Not

- A generic starter template collection.
- A framework.
- A replacement for architecture review.
- A claim that code is good because generated checks pass.

## Product Shape

Each implementation should support the same basic commands:

```sh
slophammer check <path>
slophammer check <path> --format json
slophammer check <path> --format sarif
slophammer check <path> --execute
slophammer explain <rule-id>
```

The Go implementation also includes direct checks for Uncle Bob's Go tools:

```sh
slophammer go dry [path] [--max-candidates n] [--show-report]
slophammer go crap [path] [--max-score n]
slophammer go mutate [path] [--target file] [--scan]
```

When `slophammer.yml` defines Go policy values, the direct Go commands use
those values as defaults. Explicit CLI flags still win. `check --execute` runs
the configured Go tool checks and folds tool failures into the normal report.

The checker should scan a target repository and report findings such as:

- missing `README.md`
- missing `AGENTS.md`
- missing CI workflow
- missing test command
- weak language-specific typing setup
- missing linting setup
- missing coverage gate
- missing Go complexity check
- missing `dry4go`, gated `crap4go`, or `mutate4go` declaration
- documentation that does not follow the repo convention

The shared report model should stay simple:

```json
{
  "ok": false,
  "findings": [
    {
      "rule_id": "repo.agents-required",
      "severity": "error",
      "path": "AGENTS.md",
      "message": "AGENTS.md is required"
    }
  ]
}
```

## Repository Layout

```text
.
├── AGENTS.md
├── slophammer.yml
├── docs/
│   ├── UNCLE_BOB_CONCEPTS.md
│   ├── IMPLEMENTATION_MODEL.md
│   ├── 2026-05-12-guardrails.md
│   └── uncle-bob/
├── specs/
│   ├── PRODUCT.md
│   ├── RULES.md
│   ├── CONFIG.md
│   ├── REPORT_FORMAT.md
│   └── EXIT_CODES.md
├── fixtures/
│   ├── repos/
│   └── expected/
├── go/
│   ├── cmd/slophammer/
│   ├── internal/
│   └── scripts/
└── templates/
    ├── go/
    ├── python/
    └── typescript/
```

`go/` is the working Slophammer implementation.

`templates/` contains language project references that agents can copy from.
Those templates are not full Slophammer implementations yet.

## Implementation Status

| Language   | Status                                           |
| ---------- | ------------------------------------------------ |
| Go         | Implemented checker, CLI, tool checks, fixtures, CI |
| TypeScript | Template only; checker implementation planned       |
| Python     | Template only; checker implementation planned       |

The Go implementation currently provides:

- repo rules for `README.md`, `AGENTS.md`, and CI
- Go rules for module, tests, vet, lint, coverage, and complexity
- static declarations for `dry4go`, `crap4go`, and `mutate4go`
- direct commands that run `dry4go`, `crap4go`, and `mutate4go`
- `slophammer.yml` config parsing
- native Go dependency boundary checks
- text, JSON, and SARIF report output
- shared fixtures and expected reports for clean and failing repos
- CI gates for formatting, tests, vet, lint, coverage, tool checks, and
  Slophammer's own self-check

The Go implementation now tightens `golangci-lint` with `revive`, including an
800-line production file limit, and focused production linters for security,
errors, nil handling, exhaustiveness, HTTP cleanup, context use, conversions,
whitespace, and `nolint` discipline. See
[Implementation Model](docs/IMPLEMENTATION_MODEL.md) for the detailed policy.

## Current Go Quality Surface

The Go implementation now covers the policy and architecture items that make it
useful as a reference implementation:

1. `main` stays clean between tranches.
   Completed work is committed and pushed after green checks.

2. `slophammer.yml` config is parsed.
   Config covers repo-specific policy such as coverage thresholds, DRY
   candidate budgets, CRAP score limits, mutation targets, and dependency
   boundary declarations.

3. Dependency boundary rules are native.
   This is Slophammer-owned policy. Existing tools cover lint, tests, coverage,
   duplication, CRAP, and mutation. Import direction belongs in this repo.

4. Go fixture coverage is organized by concern.
   Shared fixtures remain the acceptance contract, while tests separate
   command parsing, workflow scoping, `golangci-lint` config parsing, Go tool
   declarations, and coverage gates.

5. Formatter checks run through `golangci-lint` v2.
   CI keeps the direct `gofmt` check and adds `golangci-lint fmt --diff`.

6. SARIF output is available.
   JSON stays the stable internal report contract. SARIF lets GitHub code
   scanning consume Slophammer findings.

## Shared Rule Set

The current shared rule set is:

| Rule ID                             | Meaning                                             |
| ----------------------------------- | --------------------------------------------------- |
| `repo.readme-required`              | The target repo should have a `README.md`.          |
| `repo.agents-required`              | The target repo should have an `AGENTS.md`.         |
| `repo.ci-required`                  | The target repo should have a CI workflow.          |
| `go.module-required`                | Go projects should include a `go.mod`.              |
| `go.tests-required`                 | Go projects should run `go test ./...`.             |
| `go.vet-required`                   | Go projects should run `go vet ./...`.              |
| `go.lint-required`                  | Go projects should run `golangci-lint`.             |
| `go.coverage-required`              | Go projects should enforce coverage.                |
| `go.complexity-required`            | Go projects should check complexity.                |
| `go.dry-required`                   | Go projects should declare `dry4go`.                |
| `go.crap-required`                  | Go projects should gate `crap4go`.                  |
| `go.mutation-required`              | Go projects should declare `mutate4go`.             |
| `go.dependency-boundaries-required` | Go projects should obey configured import boundaries. |
| `ts.strict-required`                | TypeScript projects should use strict mode.         |
| `ts.no-explicit-any`                | TypeScript projects should reject `any`.            |
| `python.mypy-required`              | Python projects should run mypy.                    |
| `python.ruff-required`              | Python projects should run Ruff.                    |
| `docs.simpledoc`                    | Docs should follow the repository convention.       |

The exact shared rule behavior belongs in [Rules](specs/RULES.md) so each
implementation can share the same contract.

## Implementation Expectations

Each language implementation should demonstrate the same boundaries:

- core rule evaluation without filesystem side effects
- filesystem scanning isolated behind a small boundary
- config parsing isolated from rule logic
- text and JSON reporting
- typed findings, severities, and reports
- focused unit tests for rules
- integration tests for CLI behavior
- CI checks for formatting, linting, type checking, and tests

The implementations should be boring on purpose. Agents should be able to copy
the shape without copying accidental complexity.

The Go implementation does not have config parsing yet. That belongs in a later
slice.

## Guardrail Principles

1. Keep the core boring.
   Business rules should be ordinary code with direct tests. Frameworks,
   databases, queues, HTTP, file systems, and external APIs belong at the edges.

2. Make weak types fail early.
   Avoid broad escape hatches. If a boundary is dynamic, validate it and convert
   it to a typed shape.

3. Prefer fast local checks.
   Formatting, linting, type checking, and unit tests should be cheap enough to
   run constantly.

4. Separate policy from plumbing.
   The important rules should not depend on CLIs, web servers, ORMs, or cloud
   SDKs.

5. Treat generated code as untrusted.
   Review it, test it, type-check it, and keep the architecture understandable to
   humans.

## Concept Docs

Start with [Uncle Bob Concepts](docs/UNCLE_BOB_CONCEPTS.md) for the wiki-style
notes behind the guardrails.

See [Implementation Model](docs/IMPLEMENTATION_MODEL.md) for the shared
architecture and the Go production plan.

See [Product](specs/PRODUCT.md), [Report Format](specs/REPORT_FORMAT.md), and
[Exit Codes](specs/EXIT_CODES.md) for the shared compatibility contract.

See [DRY](docs/DRY.md) for the duplication policy informed by Uncle Bob's
structural `dry4clj`, `dry4java`, and `dry4go` tools.

The intended DRY policy is production-only: implementation code should trend
toward a zero-candidate budget, while tests are reviewed selectively, fixtures
are excluded, and templates run their own checks.
