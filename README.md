<p align="center">
  <img src="assets/unclebob_hammer.jpg" alt="Uncle Bob with a hammer" width="320">
  <br>
  <sub>*Not affiliated with Uncle Bob; only inspired by him.</sub>
</p>

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
- A released npm TypeScript implementation named `slophammer-ts`.
- A package-ready Rust implementation named `slophammer-rs`.
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
slophammer-rs
slophammer-py
```

`slophammer-go` is released. `slophammer-ts` is released to npm.
`slophammer-rs` is implemented and package-ready from the Rust workspace.
`slophammer-py` is the planned Python command; the current Python work is a
template, not a checker implementation.

The `slophammer` npm package name is reserved for a future umbrella package or
default installer. Language implementation packages should keep their
language-specific names.

Each implemented checker supports the same basic commands under its own
executable name:

```sh
slophammer-go check <path>
slophammer-go check <path> --format json
slophammer-go check <path> --format sarif
slophammer-go check <path> --execute
slophammer-go check <path> --only <rule-id>
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
slophammer-ts check <path> --only <rule-id>
slophammer-ts boundaries <path> [--format text|json|sarif]
slophammer-ts dry [path] [--max-findings n] [--show-report] [--format json|text]
```

The Rust implementation includes direct Rust quality checks:

```sh
slophammer-rs dry [path] [--max-findings n] [--format json|text]
slophammer-rs boundaries [path] [--format json|text|sarif]
slophammer-rs unsafe [path] [--format json|text|sarif]
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
- dependency imports that cross configured boundaries

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
├── rust/
│   ├── Cargo.toml
│   └── crates/
└── templates/
    ├── go/
    ├── python/
    └── typescript/
```

`go/`, `typescript/`, and `rust/` are working Slophammer implementations. Their
public product names are `slophammer-go`, `slophammer-ts`, and
`slophammer-rs`. Source-tree development commands remain available through the
local entrypoints.

Both implementations use the same internal shape:

```text
CLI
-> app orchestration
-> scanner and typed repository snapshot
-> config parser
-> rule evaluation
-> report renderer
```

Process execution is isolated behind tool-check adapters. Static `check` reads
the target repo and reports missing guardrails. `check --execute` is opt-in and
runs configured local tool commands.

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

The TypeScript checker is released to npm:

```sh
npm install -g slophammer-ts
slophammer-ts check .
slophammer-ts rules
slophammer-ts dry .
```

The Rust checker is package-ready from the Rust workspace:

```sh
cargo install --path rust/crates/slophammer-cli --locked
slophammer-rs check .
slophammer-rs rules
slophammer-rs dry .
```

[Required Next Work](docs/2026-05-17-required-next-work.md) records the release
hardening tasks that were completed for the first Go release. TypeScript remains
package-checked in CI and is published from the `typescript/` package.

## Policy Targets

`slophammer.yml` is the project policy file. This repo sets these targets:

| Policy                | Target                                     |
| --------------------- | ------------------------------------------ |
| Coverage              | at least `85`                              |
| Go CRAP               | at most `8`                                |
| TypeScript complexity | at most `8`                                |
| Production DRY        | `0` findings                               |
| Copied-block tokens   | `100` minimum token window                 |
| Go structural DRY     | `0.82` similarity, `4` lines, `20` nodes   |
| Dependency rules      | declared in `go` and `typescript` sections |

Config validation is strict. Unknown keys fail instead of being ignored, and
projects may use stricter thresholds but not weaker ones.

## Implementation Status

| Language   | Product name    | Status                                                          |
| ---------- | --------------- | --------------------------------------------------------------- |
| Go         | `slophammer-go` | Released checker, CLI, tool checks, fixtures, CI                |
| TypeScript | `slophammer-ts` | Released npm checker, CLI, native DRY, boundaries, fixtures, CI |
| Rust       | `slophammer-rs` | Package-ready checker, CLI, native DRY, boundaries, unsafe, fixtures, CI |
| Python     | `slophammer-py` | Template only; checker implementation planned                   |

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
- tool-agnostic TypeScript evidence for `tsc` or `tsgo` type checks,
  ESLint/Oxlint/Biome linting, Prettier/Oxfmt/Dprint/Biome formatting, common
  Node test runners, and `c8`/`nyc`/Vitest/Jest coverage gates
- direct `slophammer-ts boundaries` and `check --only <rule-id>` commands for
  focused adoption checks
- native CPD-style copied-block detection through `slophammer-ts dry`
- a narrowed npm package artifact with `slophammer-ts` bin verification
- CI package checks and npm publishing for the TypeScript package
- `slophammer.yml` config parsing with hard targets for coverage, complexity,
  and duplication budgets
- strict `slophammer.yml` key validation
- text, JSON, and SARIF report output
- JSON rule catalog output
- shared fixture equivalence tests against the Go implementation
- CI gates for formatting, linting, type checking, tests, coverage, build,
  native DRY, package installation, and fixture conformance

The Rust implementation currently provides:

- shared repo rules for `README.md`, `AGENTS.md`, and CI
- Rust rules for Cargo manifest, MSRV, `cargo check`, formatting, Clippy,
  tests, coverage, complexity, DRY, mutation, unsafe policy, dependency audit,
  and dependency boundaries
- direct `slophammer-rs dry`, `slophammer-rs boundaries`, and
  `slophammer-rs unsafe` commands
- strict `slophammer.yml` config parsing with hard targets for coverage,
  complexity, production DRY, unsafe policy, mutation, and boundaries
- text, JSON, and SARIF report output
- JSON rule catalog output
- shared fixture conformance with Rust success, failure, and config-error
  fixtures
- CI gates for formatting, Clippy, tests, package install, native commands, and
  shared conformance

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
   `slophammer-ts`, checks the public command name, and runs the shared
   conformance script. The Go release dry-run workflow validates release tags
   and verifies tagged `go install` on release tag pushes.

## Shared Rule Set

The shared rule registry contains 39 implemented rules: 3 shared repository
rules, 10 Go rules, 13 TypeScript rules, and 13 Rust rules. Each runtime command
prints the rules implemented by that executable: `slophammer-go rules` prints
repo plus Go rules, `slophammer-ts rules` prints repo plus TypeScript rules, and
`slophammer-rs rules` prints repo plus Rust rules.

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
| `ts.typecheck-required`             | TypeScript projects should run a no-emit typecheck.           |
| `ts.strict-required`                | TypeScript projects should use strict mode.                   |
| `ts.no-explicit-any`                | TypeScript projects should reject `any`.                      |
| `ts.no-unsafe-types`                | TypeScript projects should reject unsafe type operations.     |
| `ts.lint-required`                  | TypeScript projects should run a configured linter.           |
| `ts.format-required`                | TypeScript projects should run a formatter check.             |
| `ts.test-required`                  | TypeScript projects should run tests.                         |
| `ts.coverage-required`              | TypeScript projects should enforce coverage.                  |
| `ts.complexity-required`            | TypeScript projects should enforce complexity limits.         |
| `ts.dry-required`                   | TypeScript projects should run duplication detection.         |
| `ts.mutation-required`              | TypeScript projects should declare mutation testing.          |
| `ts.dependency-boundaries-required` | TypeScript projects should obey configured import boundaries. |
| `rust.manifest-required`            | Rust projects should include `Cargo.toml`.                    |
| `rust.msrv-required`                | Rust projects should declare an MSRV.                         |
| `rust.check-required`               | Rust projects should run `cargo check`.                       |
| `rust.fmt-required`                 | Rust projects should run `cargo fmt --check`.                 |
| `rust.clippy-required`              | Rust projects should run `cargo clippy` with warnings denied. |
| `rust.test-required`                | Rust projects should run `cargo test`.                        |
| `rust.coverage-required`            | Rust projects should enforce coverage.                        |
| `rust.complexity-required`          | Rust projects should enforce complexity limits.               |
| `rust.dry-required`                 | Rust projects should run duplication detection.               |
| `rust.mutation-required`            | Rust projects should declare mutation testing.                |
| `rust.unsafe-policy-required`       | Rust projects should declare and respect unsafe policy.       |
| `rust.dependency-audit-required`    | Rust projects should run dependency audit checks.             |
| `rust.dependency-boundaries-required` | Rust projects should obey configured dependency boundaries. |

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
