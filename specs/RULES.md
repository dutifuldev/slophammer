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
| `go.dry-required`                   | `error`  | `.github/workflows`            | `Go projects must declare dry4go`                              |
| `go.crap-required`                  | `error`  | `.github/workflows`            | `Go projects must declare crap4go with a threshold`            |
| `go.mutation-required`              | `error`  | `.github/workflows`            | `Go projects must declare mutate4go`                           |
| `go.dependency-boundaries-required` | `error`  | `slophammer.yml`               | `Go projects must respect configured dependency boundaries`    |

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

Go projects should declare `dry4go` for structural duplicate detection.

Slophammer checks for an inspectable declaration. It does not run `dry4go` in
static mode.

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

## Finding Order

Reports sort findings by `rule_id`, then by `path`.

## Machine-Readable Registry

The machine-readable rule registry lives in [`rules.json`](rules.json).
