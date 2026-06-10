---
title: Tool-Agnostic TypeScript Validation
author: Bob <dutifulbob@gmail.com>
date: 2026-05-18
status: completed
---

# Tool-Agnostic TypeScript Validation

The TypeScript checker should validate quality requirements, not one preferred
tool stack.

The production target is simple:

```sh
slophammer-ts check .
```

That command should pass or fail honestly for a real TypeScript repository. It
should not force a repository to add fake ESLint, Prettier, Vitest, or config
files just to satisfy Slophammer. It should recognize equivalent tools when
they enforce the same quality contract.

## Problem

The current TypeScript checker recognizes a narrow default stack:

- ESLint for lint rules and complexity
- Prettier for formatting
- Vitest or Jest-style coverage signals
- conventional strict `tsconfig` fields

That is useful for greenfield projects, but it is not enough for production
adoption. Real repositories may use other serious tools:

- Oxlint instead of ESLint
- Oxfmt, Biome, or Dprint instead of Prettier
- Node's built-in test runner instead of Vitest
- `c8` or `nyc` for coverage
- `tsgo` or another TypeScript-compatible typecheck command
- nested TypeScript projects that have different local gates

When Slophammer does not recognize those tools, `slophammer-ts check .` can fail
even though the repository has real gates. That creates pressure to add
performative config instead of improving the checker.

## Product Goal

Slophammer should evaluate the concern first and the tool second.

| Concern               | Acceptable Evidence                                      |
| --------------------- | -------------------------------------------------------- |
| Type checking         | `tsc --noEmit`, `tsgo --noEmit`, or equivalent strict TS |
| Linting               | ESLint, Oxlint, Biome, or another typed lint gate        |
| Unsafe TypeScript     | Tool rules that reject `any` and unsafe operations       |
| Formatting            | Prettier, Oxfmt, Biome, Dprint, or equivalent check      |
| Tests                 | Node test runner, Vitest, Jest, or equivalent CI test    |
| Coverage              | c8, nyc, Vitest, Jest, or equivalent threshold gate      |
| Complexity            | ESLint, Oxlint, Biome, or equivalent complexity rule     |
| Duplication           | Slophammer native DRY check                              |
| Mutation testing      | Stryker or equivalent mutation gate                      |
| Dependency boundaries | Slophammer native import-boundary check                  |

The report should still use stable Slophammer rule IDs. The implementation can
accept many tools, but the user should see one product contract.

## Boundary Enforcement

Dependency boundaries should be first-class Slophammer behavior.

The checker already parses `typescript.dependency_boundaries`. The user should
not need to run a full static report and filter JSON locally just to fail on
boundary violations.

Add one or both of these interfaces:

```sh
slophammer-ts check . --only ts.dependency-boundaries-required
slophammer-ts boundaries .
```

Requirements:

- Read `typescript.dependency_boundaries` from `slophammer.yml`.
- Scan TypeScript imports, exports, dynamic imports, and import types.
- Resolve relative imports to normalized repo paths.
- Treat each boundary as deny-by-default outside `from` and `allow`.
- Print file-level findings with the forbidden target path.
- Exit `1` when violations exist.
- Exit `0` when all configured boundaries pass.
- Work independently of ESLint, Prettier, Vitest, or coverage recognition.

This keeps architectural rules executable even while broader tool recognition
continues to improve.

## Coverage Scope

Coverage threshold and coverage scope must be represented together.

Today `typescript.coverage_threshold` is only a number. That is not expressive
enough for mature repositories that are raising coverage over time. If a repo
honestly gates a high-value subset, the scope should live in `slophammer.yml`,
not only inside a package script and prose.

Preferred config shape:

```yaml
typescript:
  coverage:
    threshold: 85
    paths:
      - src/runtime
      - src/flows
    exclude:
      - "**/*.test.ts"
      - "dist/**"
      - "dist-test/**"
```

Compatibility:

- Keep `typescript.coverage_threshold` as a legacy shorthand.
- Treat the legacy shorthand as whole-project unless a new scoped coverage
  shape is present.
- Do not let docs claim whole-project coverage when the configured gate is
  scoped.

The checker should report both the threshold and the scope it found.

## Tool Evidence Model

Each TypeScript rule should be backed by a small evidence detector.

An evidence detector should answer:

1. What quality concern is being checked?
2. Which tool or command provides evidence?
3. Is the command in CI, package scripts, or both?
4. Does it enforce the configured threshold or rule value?
5. Is the evidence scoped? If so, where is that scope declared?

The rule engine should not rely on raw substring checks alone when structured
parsing is available. Package scripts can be parsed into command segments, but
tool-specific config files should use structured parsers where practical.

## Nested Projects

Nested TypeScript projects should be intentional.

Slophammer should distinguish:

- the root project
- nested packages that are part of the product
- examples and fixtures that have their own local gates
- generated output

The report should not blindly fail a root repo because an example directory has
a local `tsconfig.json`. The config should allow explicit project scopes.

Example:

```yaml
typescript:
  projects:
    - root: .
      role: product
    - root: examples/flows/replay-viewer
      role: example
      inherit_root_gates: true
```

This avoids false failures while still making project boundaries visible.

## Command Design

The long-term command should remain:

```sh
slophammer-ts check .
```

Useful narrower commands are still valuable:

```sh
slophammer-ts check . --only ts.coverage-required
slophammer-ts check . --only ts.dependency-boundaries-required
slophammer-ts boundaries .
slophammer-ts dry .
slophammer-ts rules --format json
```

`--only` should run the named rule against the normal repository snapshot and
configuration. It should not require every other rule to pass first.

Direct commands such as `boundaries` are useful when Slophammer owns the check
itself. They should produce the same finding IDs and exit-code model as
`check`.

## Done Means

This work is done when a mature TypeScript repo can run:

```sh
slophammer-ts check .
```

and get a truthful result without local glue.

Specifically:

- Oxlint can satisfy lint, unsafe-type, and complexity rules when configured.
- Oxfmt can satisfy formatting rules when configured.
- Node test runner plus `c8` can satisfy tests and coverage rules.
- Scoped coverage is declared in `slophammer.yml` and reported clearly.
- Dependency boundaries are enforced directly by Slophammer.
- Nested projects do not produce false failures by default.
- Repositories do not need fake config files for tools they do not use.
- Narrow rule execution works for adoption and CI migration.

The goal is not to bless every tool forever. The goal is to make Slophammer's
quality contract portable across serious TypeScript toolchains.

## Implemented Shape

The TypeScript implementation recognizes the common production stack variants
without requiring fake config:

- `tsc --noEmit` and `tsgo --noEmit` satisfy typecheck evidence.
- ESLint and Oxlint satisfy explicit-`any`, unsafe-type, lint, and complexity
  evidence when the relevant rules are configured.
- Prettier, Oxfmt, Biome, and Dprint satisfy formatter evidence when run as
  checks.
- Node's built-in test runner, Vitest, Jest, Mocha, Ava, Uvu, Tap, `tsx
  --test`, and Playwright satisfy test evidence.
- `c8`, `nyc`, and Vitest threshold flags satisfy coverage evidence when all
  line, branch, function, and statement thresholds meet the configured target.
- Workflow matrix commands are inspected when a workflow executes
  `${{ matrix.command }}`.
- Package-less nested TypeScript examples are treated as root-owned by default
  instead of independent packages.
- `slophammer-ts check . --only <rule-id>` runs a single rule.
- `slophammer-ts boundaries .` runs the dependency-boundary rule directly.

Scoped coverage config is accepted through:

```yaml
typescript:
  coverage:
    threshold: 85
    paths:
      - src/runtime
    exclude:
      - dist/**
```

The legacy `typescript.coverage_threshold` shorthand remains supported.
