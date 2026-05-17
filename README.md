# Slophammer

Reference implementations and templates for repository quality checks in
agent-assisted software projects.

Slophammer checks whether a project has the basic constraints needed to keep
AI-generated code under control: agent instructions, CI, tests, strict typing,
linting, coverage, documentation conventions, and project structure that humans
can still review.

This repository keeps the product contract, implementations, fixtures,
templates, and CI in one place so agents can copy tested patterns from working
code.

## Quick setup: tell your agent about Slophammer

If you are an agent applying Slophammer standards to an existing repository,
start with [Agent Entrypoint](docs/AGENT_ENTRYPOINT.md). It explains how to
choose the right implementation, what to add, which targets to enforce, how to
wire CI, and how to report the result.

Copy the block below and paste it into your coding agent when you want it to
apply Slophammer standards to a repository.

```text
Use Slophammer to enforce repository quality for this project.

Attention agent: start with this file before changing code:
https://raw.githubusercontent.com/dutifuldev/slophammer/refs/heads/main/docs/AGENT_ENTRYPOINT.md

Follow it exactly. Detect the target repo's language, use the matching Slophammer
implementation, and say clearly if no matching implementation exists.
```

## What This Is

- A small product spec for a repo quality checker.
- Separate user-facing implementations of that product.
- A released Go implementation named `slophammer-go`.
- A private, package-checked TypeScript implementation named `slophammer-ts`.
- Go, TypeScript, and Python project templates with strict local checks.
- A reference for project structure, testing, errors, reporting, and CI.
- A source of patterns for agents working in different language ecosystems.

## What This Is Not

- A generic starter template collection.
- A framework.
- A replacement for architecture review.
- A claim that code is good because generated checks pass.

## Product Shape

Slophammer is the standard. Implementations use short, language-specific names:

```sh
slophammer-go
slophammer-ts
slophammer-py
```

`slophammer-go` is released. `slophammer-ts` is implemented and package-checked
but not published to npm yet. `slophammer-py` is the planned Python command; the
current Python work is a template, not a checker implementation.

Each implemented checker supports the same basic commands under its own
executable name:

```sh
slophammer-go check <path>
slophammer-go check <path> --format json
slophammer-go check <path> --format sarif
slophammer-go check <path> --execute
slophammer-go explain <rule-id>
slophammer-go rules [--format text|json]
```

The language suffix names the implementation and packaging target, not a hard
limit on what the checker inspects. Each implementation is best at its native
ecosystem first, and it can carry selected checks for other languages when
those checks reuse the shared Slophammer contract without turning the tool into
a pile of duplicated code.

The public names are the documented interface. Older local commands remain
compatibility forms during the transition.

The Go implementation also includes direct Go quality checks:

```sh
slophammer-go dry [path] [--max-candidates n] [--show-report] [--format json|text]
slophammer-go crap [path] [--max-score n]
slophammer-go mutate [path] [--target file] [--scan]
```

When `slophammer.yml` defines Go policy values, the direct Go commands use
those values as defaults. Explicit CLI flags still win. `check --execute` runs
the configured Go tool checks and folds tool failures into the normal report.

The TypeScript implementation also includes native copied-block detection:

```sh
slophammer-ts dry [path] [--max-findings n] [--show-report] [--format json|text]
```

Both working implementations expose a `rules` command with text and JSON output
so agents can inspect the implemented rule catalog without reading source
files.

The checker scans a target repository and reports findings such as:

- missing `README.md`
- missing `AGENTS.md`
- missing CI workflow
- missing test command
- weak language-specific typing setup
- missing linting setup
- missing coverage gate
- missing Go complexity check
- missing TypeScript strictness, unsafe-type, complexity, or mutation setup
- missing DRY, gated `crap4go`, or `mutate4go` declaration
- documentation that does not follow the repo convention

The shared report model stays simple:

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
│   ├── cmd/slophammer-go/
│   ├── internal/
│   └── scripts/
├── typescript/
│   ├── src/
│   └── tests/
└── templates/
    ├── go/
    ├── python/
    └── typescript/
```

`go/` and `typescript/` are working Slophammer implementations. Their public
product names are `slophammer-go` and `slophammer-ts`. Source-tree development
commands remain available through the local entrypoints.

`templates/` contains language project references that agents can copy from.
Those templates are not full Slophammer implementations yet.

`scripts/check-conformance.mjs` runs the shared fixture contract against the Go
and TypeScript implementations. It verifies JSON report shape, findings, and
exit codes for the fixture sets each implementation supports.

The Go checker is released from the
[Slophammer releases](https://github.com/dutifuldev/slophammer/releases). Install
the current release with:

```sh
go install github.com/dutifuldev/slophammer/go/cmd/slophammer-go@latest
```

[Required Next Work](docs/2026-05-17-required-next-work.md) records the release
hardening tasks that were completed for that Go release. TypeScript remains
private and package-checked in CI, but npm publishing is intentionally deferred.

## Implementation Status

| Language   | Product name    | Status                                                     |
| ---------- | --------------- | ---------------------------------------------------------- |
| Go         | `slophammer-go` | Released checker, CLI, tool checks, fixtures, CI           |
| TypeScript | `slophammer-ts` | Implemented private checker, CLI, native DRY, fixtures, CI |
| Python     | `slophammer-py` | Template only; checker implementation planned              |

An implementation can check more than one language. For example,
`slophammer-go` can operate as a strong single-binary checker for Go,
TypeScript, and Python repos when those cross-language rules are cleanly
implemented and covered by shared fixtures. Every implementation does not need
the same language coverage or the same depth.

The Go implementation currently provides:

- repo rules for `README.md`, `AGENTS.md`, and CI
- Go rules for module, tests, vet, lint, coverage, and complexity
- static declarations for DRY, `crap4go`, and `mutate4go`
- direct commands for native DRY, `crap4go`, and `mutate4go`
- an installed-binary release check for `slophammer-go`
- `slophammer.yml` config parsing
- strict `slophammer.yml` key validation
- native Go dependency boundary checks
- text, JSON, and SARIF report output
- JSON rule catalog output
- shared fixtures and expected reports for clean and failing repos
- CI gates for formatting, tests, vet, lint, coverage, tool checks, and
  Slophammer's own self-check

The Go implementation now tightens `golangci-lint` with `revive`, including an
800-line production file limit, and focused production linters for security,
errors, nil handling, exhaustiveness, HTTP cleanup, context use, conversions,
whitespace, and `nolint` discipline. See
[Implementation Model](docs/IMPLEMENTATION_MODEL.md) for the detailed policy.

The TypeScript implementation currently provides:

- shared repo rules for `README.md`, `AGENTS.md`, and CI
- TypeScript rules for package setup, strict compiler options, unsafe-type lint
  rules, formatting, linting, tests, coverage, complexity, DRY, mutation
  declaration, and dependency boundaries
- native CPD-style copied-block detection through `slophammer-ts dry`
- a narrowed npm package artifact with `slophammer-ts` and legacy `slophammer`
  bin verification
- CI package checks, without npm publishing for now
- `slophammer.yml` config parsing with hard targets for coverage, complexity,
  and duplication budgets
- strict `slophammer.yml` key validation
- text, JSON, and SARIF report output
- JSON rule catalog output
- shared fixture equivalence tests against the Go implementation
- CI gates for formatting, linting, type checking, tests, coverage, build,
  native DRY, package installation, and fixture conformance

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
   CRAP, and mutation. DRY is native because Slophammer combines structural
   function similarity with CPD-style copied-block detection. Import direction
   belongs in this repo.

4. Go fixture coverage is organized by concern.
   Shared fixtures remain the acceptance contract, while tests separate
   command parsing, workflow scoping, `golangci-lint` config parsing, Go tool
   declarations, and coverage gates.

5. Formatter checks run through `golangci-lint` v2.
   CI keeps the direct `gofmt` check and adds `golangci-lint fmt --diff`.

6. SARIF output is available.
   JSON stays the stable internal report contract. SARIF lets GitHub code
   scanning consume Slophammer findings.

7. Release checks exercise installed artifacts.
   CI installs `slophammer-go` into a temporary `GOBIN`, packs and installs
   `@dutifuldev/slophammer-ts`, checks both public command names, and runs the
   shared conformance script. The Go implementation is the release target;
   TypeScript package publishing is deferred. The Go release dry-run workflow
   validates release tags and verifies tagged `go install` on release tag
   pushes.

## Shared Rule Set

The current implemented rule registry contains 26 rules: 3 shared repository
rules, 10 Go rules, and 13 TypeScript rules.

The implemented rule set is:

| Rule ID                             | Meaning                                                       |
| ----------------------------------- | ------------------------------------------------------------- |
| `repo.readme-required`              | The target repo should have a `README.md`.                    |
| `repo.agents-required`              | The target repo should have an `AGENTS.md`.                   |
| `repo.ci-required`                  | The target repo should have a CI workflow.                    |
| `go.module-required`                | Go projects should include a `go.mod`.                        |
| `go.tests-required`                 | Go projects should run `go test ./...`.                       |
| `go.vet-required`                   | Go projects should run `go vet ./...`.                        |
| `go.lint-required`                  | Go projects should run `golangci-lint`.                       |
| `go.coverage-required`              | Go projects should enforce coverage.                          |
| `go.complexity-required`            | Go projects should check complexity.                          |
| `go.dry-required`                   | Go projects should declare a DRY check.                       |
| `go.crap-required`                  | Go projects should gate `crap4go`.                            |
| `go.mutation-required`              | Go projects should declare `mutate4go`.                       |
| `go.dependency-boundaries-required` | Go projects should obey configured import boundaries.         |
| `ts.package-required`               | TypeScript projects should include `package.json`.            |
| `ts.typecheck-required`             | TypeScript projects should run `tsc --noEmit`.                |
| `ts.strict-required`                | TypeScript projects should use strict mode.                   |
| `ts.no-explicit-any`                | TypeScript projects should reject `any`.                      |
| `ts.no-unsafe-types`                | TypeScript projects should reject unsafe type operations.     |
| `ts.lint-required`                  | TypeScript projects should run ESLint.                        |
| `ts.format-required`                | TypeScript projects should run a formatter check.             |
| `ts.test-required`                  | TypeScript projects should run tests.                         |
| `ts.coverage-required`              | TypeScript projects should enforce coverage.                  |
| `ts.complexity-required`            | TypeScript projects should enforce complexity limits.         |
| `ts.dry-required`                   | TypeScript projects should run duplication detection.         |
| `ts.mutation-required`              | TypeScript projects should declare mutation testing.          |
| `ts.dependency-boundaries-required` | TypeScript projects should obey configured import boundaries. |

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

The implementations should keep rule logic separate from CLI parsing,
filesystem scanning, config parsing, and report rendering.

## Guardrail Principles

1. Keep the core isolated.
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

5. Apply checks to generated code.
   Generated code must pass the same formatter, linter, type, test, and boundary
   checks unless it is explicitly excluded in config.

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
