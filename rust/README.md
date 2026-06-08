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

The Rust implementation is a multi-crate workspace. Do not publish the CLI crate
first: crates.io must be able to resolve every internal dependency by version.

Before publishing each crate, verify that crate packages cleanly:

```sh
cargo package -p slophammer-core --locked
cargo package -p slophammer-scan --locked
cargo package -p slophammer-config --locked
cargo package -p slophammer-report --locked
cargo package -p slophammer-rust --locked
cargo package -p slophammer-exec --locked
cargo package -p slophammer-app --locked
cargo package -p slophammer-rs --locked
```

For the first crates.io release, later package commands may require earlier
internal crates to already exist in the registry. Treat packaging as part of the
ordered publish sequence, not as a single all-at-once workspace step.

The current release dry-run workflow proves source installation and foundational
package metadata. A crates.io release should expand that gate to document or
automate the ordered package/publish sequence for every workspace crate.

## Publish Order

Publish internal crates in dependency order, then publish the CLI package:

1. `slophammer-core`
2. `slophammer-scan`
3. `slophammer-config`
4. `slophammer-report`
5. `slophammer-rust`
6. `slophammer-exec`
7. `slophammer-app`
8. `slophammer-rs`

The CLI package depends on the internal crates by version and path. Cargo can
install it locally before publication. Full `cargo package -p slophammer-rs`
verification succeeds after the internal crate versions exist in the registry.

Use the same order for `cargo publish -p <crate> --locked`. Wait for each crate
version to become available on crates.io before publishing the next dependent
crate.
