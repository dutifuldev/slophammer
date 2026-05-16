# Rules

This file defines the shared rule contract. Every implementation should use the
same rule IDs, severities, finding paths, messages, and descriptions for shared
rules.

## Shared Repository Rules

| Rule ID                | Severity | Finding path        | Finding message                                                      |
| ---------------------- | -------- | ------------------- | -------------------------------------------------------------------- |
| `repo.readme-required` | `error`  | `README.md`         | `README.md is required`                                              |
| `repo.agents-required` | `error`  | `AGENTS.md`         | `AGENTS.md is required`                                              |
| `repo.ci-required`     | `error`  | `.github/workflows` | `.github/workflows must contain at least one .yml or .yaml workflow` |

## Go Rules

Go rules apply only when the target appears to be a Go project. A repo appears
to be a Go project when it contains Go source, a `go.mod` file, or declared Go
commands.

| Rule ID                             | Severity | Finding path                   | Finding message                                                |
| ----------------------------------- | -------- | ------------------------------ | -------------------------------------------------------------- |
| `go.module-required`                | `error`  | `go.mod`                       | `Go projects must include a go.mod file`                       |
| `go.tests-required`                 | `error`  | `.github/workflows`            | `Go projects must declare go test ./... in CI or scripts`      |
| `go.vet-required`                   | `error`  | `.github/workflows`            | `Go projects must declare go vet ./... in CI or scripts`       |
| `go.lint-required`                  | `error`  | `.golangci.yml`                | `Go projects must configure and declare golangci-lint`         |
| `go.coverage-required`              | `error`  | `scripts/check-go-coverage.sh` | `Go projects must declare a coverage gate`                     |
| `go.complexity-required`            | `error`  | `.golangci.yml`                | `Go projects must enable a complexity linter`                  |
| `go.dry-required`                   | `error`  | `.github/workflows`            | `Go projects must declare a DRY check`                         |
| `go.crap-required`                  | `error`  | `.github/workflows`            | `Go projects must declare crap4go with a threshold`            |
| `go.mutation-required`              | `error`  | `.github/workflows`            | `Go projects must declare mutate4go`                           |
| `go.dependency-boundaries-required` | `error`  | `slophammer.yml`               | `Go projects must respect configured dependency boundaries`    |

## TypeScript Rules

TypeScript rules apply only when the target appears to be a TypeScript project.
A repo appears to be a TypeScript project when it contains `tsconfig.json`,
production `.ts` or `.tsx` source, or `package.json` TypeScript signals such as
the `typescript` package, `@typescript-eslint/*`, `tsc`, or a `typecheck`
script. `@types/*` packages alone do not make a JavaScript package a
TypeScript project.

| Rule ID                               | Severity | Finding path        | Finding message                                                |
| ------------------------------------- | -------- | ------------------- | -------------------------------------------------------------- |
| `ts.package-required`                 | `error`  | `package.json`      | `TypeScript projects must include package.json`                |
| `ts.typecheck-required`               | `error`  | `.github/workflows` | `TypeScript projects must declare tsc --noEmit in CI or scripts` |
| `ts.strict-required`                  | `error`  | `tsconfig.json`     | `TypeScript projects must enable strict mode`                  |
| `ts.no-explicit-any`                  | `error`  | `eslint.config.mjs` | `TypeScript projects must reject explicit any`                 |
| `ts.no-unsafe-types`                  | `error`  | `eslint.config.mjs` | `TypeScript projects must reject unsafe type operations`       |
| `ts.lint-required`                    | `error`  | `.github/workflows` | `TypeScript projects must declare ESLint in CI or scripts`     |
| `ts.format-required`                  | `error`  | `.github/workflows` | `TypeScript projects must declare a formatter check`           |
| `ts.test-required`                    | `error`  | `.github/workflows` | `TypeScript projects must declare tests in CI or scripts`      |
| `ts.coverage-required`                | `error`  | `.github/workflows` | `TypeScript projects must declare a coverage gate`             |
| `ts.complexity-required`              | `error`  | `eslint.config.mjs` | `TypeScript projects must enforce complexity limits`           |
| `ts.dry-required`                     | `error`  | `.github/workflows` | `TypeScript projects must declare a DRY check`                 |
| `ts.mutation-required`                | `error`  | `.github/workflows` | `TypeScript projects must declare mutation testing`            |
| `ts.dependency-boundaries-required`   | `error`  | `slophammer.yml`    | `TypeScript projects must respect configured dependency boundaries` |

## Rule Descriptions

### `repo.readme-required`

The target repo should have a `README.md`.

The filename comparison is case-insensitive.

### `repo.agents-required`

The target repo should have an `AGENTS.md`.

The filename comparison is case-insensitive.

### `repo.ci-required`

The target repo should have a CI workflow under `.github/workflows`.

Any regular file directly under `.github/workflows` with a `.yml` or `.yaml`
extension satisfies the rule.

### `go.module-required`

Go projects should include a `go.mod` file.

The file may live at the repository root or inside a language implementation
directory such as `go/`.

### `go.tests-required`

Go projects should declare a `go test` command against `./...` in an
inspectable workflow or script. Common test flags before the package pattern are
accepted.

### `go.vet-required`

Go projects should declare `go vet ./...` in an inspectable workflow or script.

### `go.lint-required`

Go projects should configure `golangci-lint` and declare a lint check in CI or
scripts.

The rule accepts `.golangci.yml` or `.golangci.yaml`.

### `go.coverage-required`

Go projects should declare a coverage gate using coverage output from `go test`,
`go tool cover`, and a minimum threshold in a workflow or script that Slophammer
can inspect.

### `go.complexity-required`

Go projects should enable a complexity linter through `golangci-lint`.
Comments, settings, and disabled linter entries do not satisfy this rule.

The accepted linter names are:

- `cyclop`
- `gocognit`
- `gocyclo`

`linters.default: all` is also accepted because it enables the complexity
linters through golangci-lint.

### `go.dry-required`

Go projects should declare `slophammer go dry` for structural and copied-block
duplicate detection.

Slophammer checks for an inspectable declaration. It does not run the DRY
engine in static mode. Existing `dry4go` declarations remain accepted as legacy
evidence, but new repos should declare the Slophammer command.

### `go.crap-required`

Go projects should declare `crap4go` with a threshold for complexity and
coverage risk scoring.

Slophammer checks for an inspectable declaration and threshold. It does not run
`crap4go` in static mode.

### `go.mutation-required`

Go projects should declare `mutate4go` in an inspectable workflow or script.

The mutation command may live in a normal CI workflow, nightly workflow, manual
workflow, or script. Slophammer checks for a declaration in static mode.

### `go.dependency-boundaries-required`

Go projects should keep imports inside the dependency boundaries declared in
`slophammer.yml`.

The rule is active when `go.dependency_boundaries` contains at least one
boundary. External imports are ignored. Local imports are checked by resolving
the import path through the nearest `go.mod` module path.

Boundary paths may be written relative to the repository root or relative to the
Go module root. For example, both `go/internal/rules` and `internal/rules` can
describe the same package when the module root is `go/`.

### `ts.package-required`

TypeScript projects should include `package.json`.

The file may live at the repository root or inside a language implementation
directory such as `typescript/`.

### `ts.typecheck-required`

TypeScript projects should declare `tsc --noEmit` in an inspectable workflow,
script, or `package.json`. A script named `typecheck` does not satisfy this rule
unless it actually runs the TypeScript compiler with `--noEmit`.

### `ts.strict-required`

TypeScript projects should enable strict compiler settings in `tsconfig.json`.

The rule requires `strict`, `noImplicitAny`, `noImplicitOverride`,
`noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`,
`noFallthroughCasesInSwitch`, `noPropertyAccessFromIndexSignature`,
`useUnknownInCatchVariables`, and `noEmitOnError`.

### `ts.no-explicit-any`

TypeScript projects should configure ESLint to reject explicit `any`.

### `ts.no-unsafe-types`

TypeScript projects should configure ESLint to reject unsafe assignments, calls,
member access, and returns.

### `ts.lint-required`

TypeScript projects should declare ESLint in CI, scripts, or `package.json`.

### `ts.format-required`

TypeScript projects should declare a formatter check, normally Prettier.

### `ts.test-required`

TypeScript projects should declare a test command, normally Vitest or
`npm test`.

### `ts.coverage-required`

TypeScript projects should declare a coverage gate. The recommended minimum is
`85`.

### `ts.complexity-required`

TypeScript projects should enforce complexity limits through ESLint. The
recommended cyclomatic complexity maximum is `8`.

### `ts.dry-required`

TypeScript projects should declare `slophammer typescript dry` for native
copied-block duplicate detection.

### `ts.mutation-required`

TypeScript projects should declare TypeScript mutation testing, normally through
StrykerJS. Go mutation commands do not satisfy this rule.

### `ts.dependency-boundaries-required`

TypeScript projects should declare dependency boundaries in `slophammer.yml`
and keep imports inside those boundaries.

If no TypeScript boundaries are declared, the rule reports a finding. External
package imports are ignored. Relative imports are resolved against the importing
file.

## Finding Order

Reports sort findings by `rule_id`, then by `path`.

## Machine-Readable Registry

The machine-readable rule registry lives in [`rules.json`](rules.json).
