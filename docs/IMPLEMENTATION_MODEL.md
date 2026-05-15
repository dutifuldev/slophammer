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
│   ├── REPORT_FORMAT.md
│   └── EXIT_CODES.md
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

## Go Production Plan

The Go implementation should become the production reference first.

The goal is not to invent a private quality system. The goal is to make normal
Go guardrails and Uncle Bob's Go tools easy for agents to discover, run, and
interpret.

Slophammer should own the policy and report format. It should not duplicate
existing tools unless direct use proves impractical.

## Go Rule Set

The production Go rule set should be:

| Rule ID                             | Source of truth                          | Slophammer responsibility                                |
| ----------------------------------- | ---------------------------------------- | -------------------------------------------------------- |
| `repo.readme-required`              | Repository files                         | Check that `README.md` exists.                           |
| `repo.agents-required`              | Repository files                         | Check that `AGENTS.md` exists.                           |
| `repo.ci-required`                  | GitHub Actions files                     | Check that a workflow exists.                            |
| `go.module-required`                | Go toolchain                             | Check that `go.mod` exists.                              |
| `go.tests-required`                 | `go test ./...`                          | Check that tests are declared in CI or scripts.          |
| `go.vet-required`                   | `go vet ./...`                           | Check that vet is declared in CI or scripts.             |
| `go.lint-required`                  | `golangci-lint`                          | Check that linting is configured and declared.           |
| `go.coverage-required`              | `go test -coverprofile`, `go tool cover` | Check that a coverage gate is configured.                |
| `go.complexity-required`            | `golangci-lint` complexity linters       | Check that complexity linting is configured.             |
| `go.dry-required`                   | Slophammer native DRY                    | Check that structural and copied-block duplicate detection is configured. |
| `go.crap-required`                  | `crap4go`                                | Check that CRAP analysis is configured.                  |
| `go.mutation-required`              | `mutate4go`                              | Check that mutation testing has a workflow slot.         |
| `go.dependency-boundaries-required` | Slophammer config plus Go imports        | Check declared import boundaries.                        |

The first group is standard Go practice. The second group should use Uncle
Bob's tools directly. The dependency-boundary rule is Slophammer policy because
the allowed architecture is project-specific.

## Tooling Policy

Slophammer should prefer existing tools in this order:

1. Use the normal Go command directly.
2. Use a stable existing Go tool directly.
3. Wrap the tool output into Slophammer findings.
4. Add a native implementation only when direct use blocks production use.

For Uncle Bob's Go tools, the default position is direct use unless there is a
clear production reason to absorb behavior:

- use `crap4go` for CRAP analysis
- use `mutate4go` for mutation testing

DRY is the exception. Slophammer owns a native `go dry` engine now because the
production target needs both `dry4go`-style structural similarity and CPD-style
copied-block detection in one report and one budget.

Do not copy those implementations into Slophammer just to reduce dependencies.
Absorb behavior only with evidence that direct use is not production-ready for
this repo.

These are not later optional ideas. They belong in the Go production rule set.
The first implementation should check that a Go repo declares these tools in its
quality workflow. Native reimplementation is the later fallback, not the plan.

Valid reasons to absorb a tool are:

- it cannot be installed reliably in CI
- it cannot produce output Slophammer can parse
- it is too slow for the intended check mode
- it cannot be configured for this repo's policy
- it has correctness issues that affect Slophammer's fixtures
- upstream maintenance stops and the tool becomes a release risk

Until one of those is true, Slophammer should call the tool. For DRY, that
condition has been met and the runtime dependency on `dry4go` has been removed.

## Go Lint Policy

The current Go implementation uses `golangci-lint` as the lint runner. That is
the right integration point because it lets the project compose normal Go
linters without turning Slophammer into another lint framework.

The current baseline stays focused on high-signal checks:

- `staticcheck`
- `unused`
- `errcheck`
- `ineffassign`
- `cyclop`
- `gocognit`
- `revive`
- `gosec` for security mistakes
- `errorlint` for error wrapping and matching mistakes
- `nilerr` and `nilnesserr` for nil-related mistakes
- `exhaustive` for missing enum-style cases
- `bodyclose` for leaked HTTP response bodies
- `noctx` for HTTP calls without context
- `unconvert` for unnecessary conversions
- `whitespace` for noisy formatting drift
- `nolintlint` for uncontrolled suppressions

`revive` enforces `file-length-limit`, because oversized source files are easy
for agents to keep growing and hard for humans to review. The Go implementation
uses an 800-line production limit and excludes test files from that specific
rule. Use the rule as refactoring pressure, not as a reason to hide logic in
worse places.

Add noisier linters only after the core signal is stable:

- `gocritic`
- `prealloc`
- `dupl`

`dupl` overlaps with the intent of native DRY, so it should be treated as a
cheap lint-time hint, not the source of truth for duplicate detection.

Formatting is separate from linting. Use Go's normal tools and
`golangci-lint` formatters for formatting:

- `gofmt`
- `gofumpt`
- `goimports`
- `gci`

Do not describe these formatters as lint rules. They can run in the same CI
workflow, but they solve a different problem.

The formatter gate keeps the existing `gofmt` check and adds:

```sh
golangci-lint fmt --diff
```

Start with `gofmt` through `golangci-lint` formatters. Add `gofumpt`,
`goimports`, or `gci` only when the expected import and formatting policy is
spelled out in this document and the config.

## Static And Execute Modes

The default mode should be static inspection.

Static inspection checks whether the target repo declares the right guardrails.
It reads files, workflows, scripts, and config. It does not run arbitrary target
repo commands.

Example static checks:

- `go.mod` exists
- `.github/workflows/*.yml` includes `go test ./...`
- `.github/workflows/*.yml` includes `go vet ./...`
- `.golangci.yml` exists
- `.golangci.yml` enables complexity linters
- a coverage script or CI command enforces coverage
- a `slophammer go dry` check is present in CI or scripts
- a `crap4go` check is present in CI or scripts
- a `mutate4go` command exists in a slow, nightly, or manual workflow

Execution mode should be explicit:

```sh
slophammer check <path> --execute
```

Execution mode runs configured Go checks, parses their output where needed, and
adds failures to the normal Slophammer report. It is opt-in because running
commands in an arbitrary repo has security and speed costs.

Go execution resolves real Go module roots before running tools. That lets the
repo-level `slophammer.yml` drive this repo's nested `go/` implementation
without running Go tools from the repository root. Embedded `fixtures/`,
`templates/`, and `vendor/` modules are skipped as execution targets.

## Existing Go Tooling

These checks should use existing Go tooling:

```sh
go test ./...
go vet ./...
golangci-lint run
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

For complexity, Slophammer should inspect `golangci-lint` config for linters
such as:

- `cyclop`
- `gocognit`
- `gocyclo`

The exact accepted set belongs in `specs/RULES.md` once the rule is implemented.

## Uncle Bob Tooling

The advanced quality checks should start as direct integrations with Uncle
Bob's Go tools.

### `slophammer go dry`

Use the native Slophammer DRY engine for Go duplicate detection.

Slophammer checks that the repo has a declared `slophammer go dry` command in
CI or a script. Existing `dry4go` declarations remain accepted as legacy
evidence, but the recommended command is the first-class Slophammer command. If
`slophammer.yml` sets `go.dry.max_findings` or `go.dry_max_candidates`, that
value is the default budget unless `--max-candidates` is passed.

The production target is strict: production Go code should have a zero-candidate
DRY budget. Tests should be reviewed separately, fixtures should be excluded,
and templates should run their own checks as reference projects. Slophammer
enforces that by expanding configured include paths and exclude globs before
running its native DRY engine.

The native engine combines:

- structural function and method similarity based on the `dry4go` model
- CPD-style copied-block detection that preserves identifiers and literals

The static `go.dry-required` rule is a hard requirement: the repo must declare a
DRY check somewhere Slophammer can inspect.

### `crap4go`

Use `crap4go` for CRAP scoring.

Slophammer checks that the repo has a declared CRAP command and a clear
threshold. The first-class `slophammer go crap` command runs `crap4go` directly,
parses its report, and fails when a function exceeds the configured maximum. If
`slophammer.yml` sets `go.crap_max_score`, that value is the default maximum
unless `--max-score` is passed.

Coverage and complexity already exist separately in Go tooling. CRAP is valuable
because it combines them into one risk signal.

The static `go.crap-required` rule is a hard requirement: the repo must declare
`crap4go` or `slophammer go crap` with a threshold.

### `mutate4go`

Use `mutate4go` for mutation testing.

Mutation testing is expensive, so the production policy should require a
declared workflow slot rather than requiring it on every pull request. Valid
slots are nightly, manual, release, or targeted execution for risky packages.

The first-class `slophammer go mutate` command runs `mutate4go` directly. If
`slophammer.yml` sets `go.mutation_targets`, those targets are used unless
`--target` is passed. Pull requests should use `--scan` against configured
targets; full mutation testing belongs on slower scheduled or manual paths.

The static `go.mutation-required` rule is a hard requirement: the repo must
declare `mutate4go` or `slophammer go mutate` in an inspectable workflow or
script.

## Native Slophammer Checks

Slophammer should own checks that are policy, not generic tooling.

Native checks should include:

- repo file requirements
- workflow presence
- Go command declaration in CI or scripts
- coverage gate declaration
- tool configuration declaration
- dependency boundary rules
- config parsing
- report rendering
- fixture comparison

Dependency boundaries should be native because the allowed import direction is
part of the project policy.

For Slophammer itself, the intended Go dependency direction is:

```text
cmd -> cli -> app -> scan/report/rules
rules -> repo
scan -> repo
report -> rules
repo -> no internal packages
```

No package should depend upward against that direction.

## Machine-Readable Rules

Markdown is for humans. Production rules need a machine-readable registry.

Add:

```text
specs/rules.json
```

It should define:

- rule ID
- title
- description
- category
- default severity
- default enabled state
- implementation status
- finding path
- finding message
- related tool, when a tool backs the rule

Go tests should assert that the Go rule registry matches `specs/rules.json`.

## Go Fixtures

Shared fixtures should be the acceptance suite for rule behavior.

Add Go fixtures in this shape:

```text
fixtures/repos/go-clean
fixtures/repos/go-missing-module
fixtures/repos/go-missing-tests
fixtures/repos/go-missing-vet
fixtures/repos/go-missing-lint
fixtures/repos/go-missing-coverage
fixtures/repos/go-missing-complexity
fixtures/repos/go-missing-dry
fixtures/repos/go-missing-crap
fixtures/repos/go-missing-mutation
fixtures/repos/go-bad-dependency
```

Each fixture needs a matching expected report:

```text
fixtures/expected/<fixture-name>.json
```

Every production rule should have:

- one passing fixture
- one failing fixture
- one edge-case fixture when the rule has meaningful edge cases

The next Go fixture tranche must cover both baseline Go tooling and Uncle Bob's
tools. The clean fixture should declare `go test`, `go vet`, `golangci-lint`, a
coverage gate, complexity linting, `slophammer go dry`, `crap4go`, and
`mutate4go`.

## Config

Production config should use:

```text
slophammer.yml
```

Config should tune policy. It should not make problems disappear silently.
The Go defaults are also floors and ceilings: coverage cannot be configured
below 85%, and CRAP cannot be configured above 8. Stricter values are allowed.

Example:

```yaml
rules:
  go.dry-required:
    severity: warn
    threshold: 0.82
  go.crap-required:
    severity: error
    max: 8
go:
  dependency_boundaries:
    - from: internal/rules
      allow:
        - internal/repo
    - from: internal/repo
      allow: []
```

Disabling a rule should eventually require a reason.

## Go CLI Target

The Go CLI should grow to:

```sh
slophammer check <path>
slophammer check <path> --format json
slophammer check <path> --format text
slophammer check <path> --format sarif
slophammer check <path> --execute
slophammer explain <rule-id>
slophammer rules
slophammer fixtures
```

JSON stays the stable internal report contract. SARIF is an output adapter so
GitHub code scanning can consume Slophammer findings without forcing SARIF
concerns into the rule engine.

## Go CI Target

The final CI for this repo should run:

```sh
gofmt
golangci-lint fmt --diff
go vet
go test
coverage gate
golangci-lint
slophammer check .
slophammer check . --execute
slophammer go dry ..
slophammer go crap ..
slophammer go mutate .. --scan
dependency boundary check
```

Full mutation testing should run on a slower schedule or targeted path:

- nightly
- release branches
- risky packages
- manual workflow dispatch

The pull request path may run `mutate4go --scan` because it inspects mutation
sites without running the full mutation suite.

## Go Implementation Order

The completed baseline is:

1. `specs/rules.json` exists and the Go registry is verified against it.
2. Go baseline rules cover module, tests, vet, lint, coverage, and complexity.
3. Direct Go quality checks cover native DRY, `crap4go`, and `mutate4go`.
4. Shared fixtures and expected reports cover clean and failing Go repos.
5. CI runs Slophammer's own self-check.
6. The Go lint stack includes the stricter production linters and an 800-line
   production file limit.
7. `slophammer.yml` config parsing is implemented.
8. Native dependency boundary checking is implemented.
9. CI runs `golangci-lint fmt --diff`.
10. SARIF output is implemented as a report adapter.

The implemented Go quality surface is:

1. Keep `main` clean between tranches.
   Commit and push completed work before starting the next slice so each change
   can be reviewed against a known-green baseline.

2. Use `slophammer.yml` config parsing.
   Config tunes policy without hiding problems. It parses coverage
   thresholds, DRY candidate budgets, CRAP score limits, mutation targets, and
   dependency boundary declarations.

   The long-term DRY config should support production-only include and exclude
   paths. That lets this repo enforce a zero-candidate budget over `go/cmd` and
   `go/internal` production code while excluding `_test.go`, `fixtures/`, and
   `templates/`.

3. Use native dependency boundary checking.
   This is Slophammer-owned Go policy because import direction is project
   policy, not generic Go tooling.

4. Keep Go rule fixture tests split by concern.
   Keep the shared fixture contract and organize tests around command parsing,
   workflow scoping, `golangci-lint` config parsing, Go tool declarations, and
   coverage gates.

5. Run formatter checks through `golangci-lint` v2.
   Keep the direct `gofmt` check and run `golangci-lint fmt --diff`.

6. Keep SARIF as a report adapter.
   SARIF is an output adapter over the existing findings model, not a
   second rule model.

Do not add more linters right now. The next quality gain should come from
deeper config semantics, richer dependency boundary fixtures, and report output
hardening.

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
