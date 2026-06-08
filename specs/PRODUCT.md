# Product

Slophammer is a repository quality standard for agent-assisted software
projects.

Each language implementation exposes the same product behavior, even
though each implementation uses its language's normal project structure and
tooling. Installable implementations use short language-specific product
names:

| Implementation | Public command  |
| -------------- | --------------- |
| Go             | `slophammer-go` |
| TypeScript     | `slophammer-ts` |
| Rust           | `slophammer-rs` |
| Python         | `slophammer-py` |

The product name identifies the implementation package. It does not forbid
cross-language checking. An implementation is native-first, and it can also
support other languages when those checks follow the shared Slophammer rules,
config, report, and exit-code contracts.

## Commands

The public command surface is the same for each implementation, using that
implementation's executable name:

```sh
slophammer-go check <path>
slophammer-go check <path> --format json
slophammer-go check <path> --format sarif
slophammer-go check <path> --execute
slophammer-go check <path> --only <rule-id>
slophammer-go explain <rule-id>
slophammer-go rules [--format text|json]
```

During the rename from the early `slophammer ...` command shape,
implementations keep compatibility aliases. Documentation and help text point
users to the language-specific public names.

## Check

`check` scans the target repository, evaluates the default rule set, writes a
report, and exits with a stable exit code.

The default report format is text. JSON and SARIF report formats are selected
with `--format json` and `--format sarif`.

`--execute` is opt-in. It runs configured tool checks and adds tool failures to
the same report model as static findings.

`--only <rule-id>` evaluates a focused rule without requiring every other rule
to pass first. Implementations support it when focused adoption checks are part
of their command surface.

Direct language commands may exist for checks that Slophammer owns natively.
For example, `slophammer-ts boundaries <path>` runs the TypeScript dependency
boundary rule directly while preserving the normal finding and exit-code model.
`slophammer-rs dry <path>`, `slophammer-rs boundaries <path>`, and
`slophammer-rs unsafe <path>` expose Rust DRY, dependency-boundary, and unsafe
policy checks directly.

## Explain

`explain` prints rule metadata for a known rule ID.

Unknown rule IDs are command errors.

## Rules

`rules` prints the implemented rule catalog for the current implementation.
The catalog includes rule ID, category, severity, implementation status, and
the backing tool when one exists.

The default `rules` format is text. `--format json` writes the same rule
definitions as structured data for agents and automation.

`rules` is an inspection command. It does not scan a repository and it exits
successfully when the catalog is written.

## Release Policy

The Go, TypeScript, and Rust implementations are releasable products.

Go releases use the `go/` submodule tag shape:

```sh
git tag go/v0.1.0
git push origin go/v0.1.0
```

Users install a tagged Go release with:

```sh
go install github.com/dutifuldev/slophammer/go/cmd/slophammer-go@v0.1.0
```

The TypeScript checker is released as the `slophammer-ts` npm package. The
TypeScript release workflow validates `typescript/vX.Y.Z` tags, runs the full
TypeScript package gate, runs shared conformance, and publishes with npm trusted
publishing.

The Go release dry-run workflow validates `go/vX.Y.Z` tags, runs the Go release
checks, runs shared conformance, and verifies tagged `go install` on tag push.

The Rust checker is intended to publish as the `slophammer-rs` Cargo package
under `rust/crates/slophammer-cli`. It is not published to crates.io yet. Until
publication, users install it from this repository with
`cargo install --path rust/crates/slophammer-cli --locked`.

Rust is a multi-crate Cargo workspace. A crates.io release must publish internal
crates in dependency order before publishing `slophammer-rs`:

1. `slophammer-core`
2. `slophammer-scan`
3. `slophammer-config`
4. `slophammer-report`
5. `slophammer-rust`
6. `slophammer-exec`
7. `slophammer-app`
8. `slophammer-rs`

The Rust release dry-run checks source installation, command help, rule catalog
output, fixture checks from the installed binary, and shared conformance before
any publish step. A real crates.io release should also verify or automate the
ordered `cargo package` and `cargo publish` sequence for every Rust workspace
crate. After publication, users should install with
`cargo install slophammer-rs --locked`.

## Implementation Boundary

Implementations keep this dependency direction:

```text
CLI
-> app orchestration
-> scanner
-> typed repository snapshot
-> pure rule evaluation
-> reporter
```

Rules do not walk the filesystem, parse command-line arguments, or write
reports.
