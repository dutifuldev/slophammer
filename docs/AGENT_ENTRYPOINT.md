# Agent Entrypoint

Attention agent: start here when you are asked to apply Slophammer standards to
an existing repository.

Your job is not to make the repository look busy. Your job is to make quality
enforceable. Prefer small, reviewable changes that add real checks, wire them
into CI, and leave maintainers with commands they can run locally.

## Operating Rules

- Do not weaken existing tests, linters, type checks, or CI to make your work
  pass.
- Do not add broad type escapes such as `any`, `Any`, unchecked casts,
  reflection, or dynamic maps unless they are isolated at an external boundary.
- Do not hide failures behind warnings unless the repo owner explicitly asks for
  a staged rollout.
- Do not add a dependency when the language standard library or existing project
  tooling is enough.
- Keep domain behavior separate from IO, frameworks, databases, queues, clocks,
  random values, networks, and process state.
- Add or update tests for every behavior change.
- Run the checks before you finish.

## First Pass

Start by reading the repository before changing it:

```sh
find . -maxdepth 3 -type f | sort | head -200
```

Identify:

- the primary language or languages
- the package manager and build tool
- existing test, lint, type-check, and coverage commands
- CI workflow files
- existing agent instruction files
- whether generated, fixture, vendor, or template code should be excluded from
  production quality budgets

Then make the smallest plan that turns the existing project into an enforceable
project. Do not rewrite the project structure unless the current structure blocks
quality checks.

## Required Files

Every serious repository should have these files:

```text
README.md
AGENTS.md
slophammer.yml
.github/workflows/ci.yml
```

If the repo already has equivalent CI, update it instead of adding a parallel
workflow.

## Agent Instructions

Create or update `AGENTS.md` with repository-specific instructions. Keep it
short and enforceable.

At minimum, include:

- the local commands agents must run before finishing
- language-specific typing rules
- testing expectations
- dependency rules
- architecture boundaries that should not be crossed
- a pointer back to this entrypoint if Slophammer standards are being applied

Do not fill `AGENTS.md` with generic advice that no tool checks.

## Slophammer Config

Add `slophammer.yml` at the repository root.

For Go projects, start with this policy:

```yaml
go:
  coverage_threshold: 85
  dry:
    max_findings: 0
    paths:
      - go/cmd
      - go/internal
    exclude:
      - "**/*_test.go"
      - "fixtures/**"
      - "templates/**"
    structural:
      enabled: true
      threshold: 0.82
      min_lines: 4
      min_nodes: 20
    copied_blocks:
      enabled: true
      min_tokens: 100
  crap_max_score: 8
  mutation_targets:
    - go/internal/rules/rules.go
```

Adjust `go.dry.paths`, `go.dry.exclude`, and `mutation_targets` to match the
target repository.

These are hard quality targets:

- coverage must be at least `85`
- CRAP must be at most `8`
- production DRY candidates should be `0`

Use stricter values when the repo already supports them. Do not use weaker
values.

## CI Contract

CI must run the same checks a maintainer can run locally.

For a Go project, the CI job should include:

```sh
go vet ./...
go test ./...
golangci-lint run
go build ./cmd/<binary>
slophammer-go dry .
slophammer-go crap .
slophammer-go mutate . --scan
slophammer-go check .
slophammer-go check . --execute
```

Adapt the build path to the target repo. When working inside this repository's
source tree, `go run ./cmd/slophammer-go ...` is the local development
equivalent of the installed `slophammer-go` command.

If the repo does not implement Slophammer itself, install or run the matching
Slophammer tool, such as `slophammer-go`, instead of assuming
`./cmd/slophammer-go` exists.

Use the rule catalog command when a finding needs context:

```sh
slophammer-go rules
slophammer-ts rules
```

## Language Baselines

For TypeScript:

- enable `strict: true`
- reject explicit `any`
- run formatting, linting, type checking, and tests in CI
- validate unknown external input at the boundary before converting it to typed
  domain data

For Python:

- run Ruff
- run mypy
- use type annotations on public functions and meaningful helpers
- avoid `Any` except at narrow dynamic boundaries
- run tests in CI

For Go:

- run `go test ./...`
- run `go vet ./...`
- run `golangci-lint` v2
- enforce coverage
- enforce CRAP
- run DRY detection on production code
- run mutation checks on risky target files
- keep interfaces near consumers
- keep package APIs small and explicit

## Refactoring Rules

When a new quality gate fails, fix the code. Do not lower the gate.

Use this order:

1. Add missing tests for important behavior.
2. Split high-complexity functions by responsibility.
3. Move IO and framework code away from domain logic.
4. Replace repeated code with a small helper only when the helper has a clear
   name and stable purpose.
5. Remove dead or unused paths.
6. Rerun the failing quality gate.

Do not refactor unrelated areas only because you noticed them.

## Validation

Before finishing, run the relevant local suite.

For this repository's Go implementation, the expected validation set is:

```sh
go test ./...
go vet ./...
golangci-lint run
./scripts/check-go-coverage.sh
go run ./cmd/slophammer-go dry ..
go run ./cmd/slophammer-go crap ..
go run ./cmd/slophammer-go mutate .. --scan
go run ./cmd/slophammer-go check .. --execute
./scripts/check-go-install.sh
node ../scripts/check-conformance.mjs
npx -y @simpledoc/simpledoc check
git diff --check
```

If a command cannot run locally, say exactly why and what still needs CI,
staging, or production verification.

## Final Report

When you finish, report:

- files added or changed
- policy values enforced
- checks that passed
- checks that could not run
- any remaining risks

Do not call the repo ready until the changes are committed, pushed, and CI is
green.
