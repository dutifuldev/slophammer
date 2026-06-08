# Slophammer Rust

Rust implementation of the Slophammer repository quality checker. The
user-facing product name is `slophammer-rs`.

## Commands

```sh
cargo run -p slophammer-rs -- check ..
cargo run -p slophammer-rs -- check .. --format json
cargo run -p slophammer-rs -- check .. --format sarif
cargo run -p slophammer-rs -- check .. --execute
cargo run -p slophammer-rs -- check .. --only rust.unsafe-policy-required
cargo run -p slophammer-rs -- rules --format json
cargo run -p slophammer-rs -- dry ..
cargo run -p slophammer-rs -- boundaries ..
cargo run -p slophammer-rs -- unsafe ..
```

Install the local package:

```sh
cargo install --path crates/slophammer-cli --locked
slophammer-rs check .
```

## Quality Gate

Run before finishing Rust changes:

```sh
cargo fmt --check
cargo check --workspace
cargo clippy --workspace --all-targets -- -D warnings
cargo test --workspace --all-targets
cargo test --workspace --doc
cargo llvm-cov --workspace --fail-under-lines 85
cargo install --path crates/slophammer-cli --locked
cargo run -p slophammer-rs -- dry .. --format json
cargo run -p slophammer-rs -- boundaries .. --format json
cargo run -p slophammer-rs -- unsafe .. --format json
cargo run -p slophammer-rs -- check .. --format json
```

`cargo audit` and `cargo mutants` are declared in CI. They require installing
`cargo-audit` and `cargo-mutants` locally.

## Crates.io Status

`slophammer-rs` is not published to crates.io yet. Until it is published,
install from this source tree:

```sh
cargo install --path crates/slophammer-cli --locked
```

After publication, the intended public install command is:

```sh
cargo install slophammer-rs --locked
```

## Publish Prerequisites

The production publish target is a single user-facing Cargo package:
`slophammer-rs`.

Do not publish the current internal workspace crates just to make crates.io
resolve the CLI package. Publishing `slophammer-core`, `slophammer-config`,
`slophammer-rust`, or similar crates would make their public types long-term
semver contracts before there is a real library use case.

Before the first crates.io release, refactor the Rust implementation so the CLI
package can be packaged without unpublished internal dependencies:

```sh
cargo package -p slophammer-rs --locked
```

The stable public contract should be the CLI behavior, config format, report
formats, exit codes, and rule IDs. Keep scanner, config, rule, report, and
execution internals private unless a deliberate Rust library API is added later.

See
[Rust CLI-only Cargo publish plan](../docs/2026-06-08-rust-cli-only-cargo-publish-plan.md).

## Release Workflow

The current Rust release workflow and `scripts/publish-crates.sh` were created
for an ordered multi-crate publish path. Do not use that publish path for the
first crates.io release.

Before the first release, replace the release workflow with a CLI-only publish
path that:

1. validates the `rust/vX.Y.Z` tag against the `slophammer-rs` package version,
2. verifies the tagged commit is on `origin/main`,
3. runs the Rust quality gate,
4. runs `cargo package -p slophammer-rs --locked`,
5. installs the packaged CLI artifact,
6. runs installed CLI smoke checks and shared conformance,
7. publishes only `slophammer-rs`.

Publishing to crates.io is permanent. Do not publish until the CLI-only package
artifact is proven by CI.
