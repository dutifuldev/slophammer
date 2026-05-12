# Slophammer

Multi-language reference implementations of the same repository quality checker
for agent-assisted software projects.

`slophammer` checks whether a project has the basic constraints needed to keep
AI-generated code under control: agent instructions, CI, tests, strict typing,
linting, coverage, documentation conventions, and project structure that humans
can still review.

The point of this repository is not to ship one blessed implementation. The
point is to show the same small tool implemented cleanly in multiple languages
so agents can copy patterns from real, working code.

## What This Is

- A small product spec for a repo quality checker.
- Parallel Go, TypeScript, and Python implementations of that same product.
- A reference for project structure, testing, errors, config, reporting, and CI.
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
slophammer explain <rule-id>
```

The checker should scan a target repository and report findings such as:

- missing `README.md`
- missing `AGENTS.md`
- missing CI workflow
- missing test command
- weak language-specific typing setup
- missing linting setup
- missing coverage gate
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

## Target Repository Layout

```text
.
├── AGENTS.md
├── docs/
│   ├── UNCLE_BOB_CONCEPTS.md
│   ├── IMPLEMENTATION_MODEL.md
│   ├── 2026-05-12-guardrails.md
│   └── uncle-bob/
├── specs/
│   ├── PRODUCT.md
│   ├── RULES.md
│   ├── CONFIG.md
│   └── REPORT_FORMAT.md
├── fixtures/
├── go/
├── python/
└── typescript/
```

The repo currently contains transitional language template directories. New work
should move toward this top-level language layout so each implementation follows
the same product contract.

## Shared Rule Set

Start with a small common rule set:

| Rule ID                | Meaning                                       |
| ---------------------- | --------------------------------------------- |
| `repo.readme-required` | The target repo should have a `README.md`.    |
| `repo.agents-required` | The target repo should have an `AGENTS.md`.   |
| `repo.ci-required`     | The target repo should have a CI workflow.    |
| `go.tests-required`    | Go projects should run `go test ./...`.       |
| `go.vet-required`      | Go projects should run `go vet ./...`.        |
| `ts.strict-required`   | TypeScript projects should use strict mode.   |
| `ts.no-explicit-any`   | TypeScript projects should reject `any`.      |
| `python.mypy-required` | Python projects should run mypy.              |
| `python.ruff-required` | Python projects should run Ruff.              |
| `docs.simpledoc`       | Docs should follow the repository convention. |

The exact rule behavior belongs in `specs/RULES.md` so each implementation can
share the same contract.

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
architecture that Go, TypeScript, and Python should follow.
