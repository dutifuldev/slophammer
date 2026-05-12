# Implementation Model

Slophammer should be built as one product with several language implementations.

The important part is not that Go, TypeScript, and Python share code. They
should not. The important part is that they share the same contract: rules,
fixtures, config shape, report shape, and command behavior.

## Target Shape

Keep language implementations at the repository top level:

```text
.
├── specs/
│   ├── PRODUCT.md
│   ├── RULES.md
│   ├── CONFIG.md
│   └── REPORT_FORMAT.md
├── fixtures/
│   ├── repos/
│   └── expected/
├── go/
├── typescript/
└── python/
```

This keeps the repo easy to scan. Agents should not need to infer whether the
real implementation is under `templates/`, `examples/`, or `implementations/`.

## Product Contract

Every implementation should support the same core commands:

```sh
slophammer check <path>
slophammer check <path> --format json
slophammer explain <rule-id>
```

Every implementation should use the same public concepts:

- rule IDs
- severities
- findings
- reports
- config fields
- fixture repos
- expected reports
- exit codes

Rule IDs and report fields are public API. Rename them only with intent.

## Holy Grail Loop

The implementation pattern is:

```text
scan once
-> build typed snapshot
-> run pure rules
-> render reports
```

That shape matters because it makes the checker deterministic, portable, and
easy to test.

## Boundary Rules

The rule engine should not touch the filesystem.

Rules should receive a typed snapshot of the target repository and return
findings. Filesystem walking, config loading, command parsing, and report
rendering belong outside the rules.

The clean boundary looks like this:

```text
CLI adapter
  -> app orchestration
    -> scanner adapter
      -> typed snapshot
    -> pure rule engine
    -> reporter adapter
```

Each language can express that boundary differently, but the dependency
direction should stay the same.

## Shared Fixtures

Fixtures are the cross-language test contract.

The repo should keep small target repositories under `fixtures/repos/` and the
expected reports under `fixtures/expected/`.

Example:

```text
fixtures/
├── repos/
│   ├── missing-agents/
│   ├── missing-ci/
│   └── go-no-vet/
└── expected/
    ├── missing-agents.json
    ├── missing-ci.json
    └── go-no-vet.json
```

Each implementation should run against the same fixtures and compare against the
same expected reports.

## Go Implementation

The Go version should emphasize explicit types and small packages:

```text
go/
├── cmd/slophammer/
├── internal/app/
├── internal/cli/
├── internal/config/
├── internal/report/
├── internal/repo/
├── internal/rules/
└── internal/scan/
```

The core rule package should be pure. The scanner builds the snapshot. The app
coordinates scanner, rules, config, and reporting. The CLI parses arguments and
maps results to exit codes.

## TypeScript Implementation

The TypeScript version should mirror the same boundaries:

```text
typescript/
├── src/app/
├── src/cli/
├── src/config/
├── src/report/
├── src/repo/
├── src/rules/
└── src/scan/
```

Strict TypeScript should be non-negotiable. External data should enter as
`unknown`, be validated, and then become typed domain data.

## Python Implementation

The Python version should keep the same contract with typed modules:

```text
python/
├── src/slophammer/
│   ├── app.py
│   ├── cli.py
│   ├── config.py
│   ├── report.py
│   ├── repo.py
│   ├── rules.py
│   └── scan.py
└── tests/
```

Use type annotations, mypy, and small dataclasses or typed models for public
data shapes.

## First Slice

The first useful slice should be small and complete:

1. Load default rule set.
2. Scan a target repo.
3. Build a typed snapshot.
4. Run these rules:
   - `repo.readme-required`
   - `repo.agents-required`
   - `repo.ci-required`
5. Render text and JSON reports.
6. Return stable exit codes.
7. Pass shared fixtures.

After that, add language-specific rules for Go, TypeScript, and Python.

## Done Means Portable

A feature is not fully done when one implementation works.

For shared behavior, done means:

- the spec describes it
- fixtures cover it
- expected reports define it
- each language implementation passes it

That is what makes Slophammer more than a single tool. It becomes a reference
for porting the same product across languages without losing the architecture.
