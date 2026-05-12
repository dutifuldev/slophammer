# Slophammer

Reference implementations and engineering standards for AI-assisted software projects.

This repository is for projects that may be generated, extended, or maintained by coding agents, but still need to behave like serious software. It collects language-specific baselines that make the correct path explicit: strict typing, small units, fast tests, dependency hygiene, and CI checks that fail before weak code reaches production.

## What This Is

- A reference repo for common project shapes across languages.
- A set of guardrails for agent-built and vibecoded projects.
- A place to compare an existing project against practical defaults.
- A starting point when a new project needs sane constraints on day one.

## What This Is Not

- A framework.
- A package manager wrapper.
- A replacement for architecture review.
- A promise that generated code is correct because it passes lint.

## Guardrail Principles

1. Keep the core boring.
   Business rules should be ordinary code with direct tests. Frameworks, databases, queues, HTTP, file systems, and external APIs belong at the edges.

2. Make weak types fail early.
   Avoid `any`, untyped dictionaries, implicit nulls, unchecked casts, and broad escape hatches. If a boundary is dynamic, validate it and convert it to a typed shape.

3. Prefer fast local checks.
   Formatting, linting, type checking, and unit tests should be cheap enough to run constantly.

4. Separate policy from plumbing.
   The important rules should not depend on CLIs, web servers, ORMs, or cloud SDKs.

5. Require explicit dependencies.
   Every dependency should have a job. Avoid packages that replace a few lines of clear standard-library code.

6. Treat generated code as untrusted.
   Review it, test it, type-check it, and keep the architecture understandable to humans.

## Repository Layout

```text
.
├── AGENTS.md
├── docs/
│   ├── UNCLE_BOB_CONCEPTS.md
│   ├── 2026-05-12-guardrails.md
│   └── uncle-bob/
└── templates/
    ├── go/
    ├── python/
    └── typescript/
```

## Template Status

| Language   | Focus                                                                   |
| ---------- | ----------------------------------------------------------------------- |
| TypeScript | Strict compiler settings, no explicit `any`, ESLint flat config, Vitest |
| Python     | Ruff, mypy strict mode, pytest, `src` layout                            |
| Go         | Standard layout, tests, `go vet`, golangci-lint config                  |

## How To Use This Repo

- For a new project, copy the closest template and rename the package/module.
- For an existing project, compare its lint, typing, tests, and boundaries against the matching template.
- For agent workflows, point the agent at `AGENTS.md` and the relevant language template before implementation starts.

## Concept Docs

Start with [Uncle Bob Concepts](docs/UNCLE_BOB_CONCEPTS.md) for the wiki-style
notes behind the guardrails.
