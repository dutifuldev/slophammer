---
title: Rust CLI-Only Cargo Publish Plan
author: Bob <dutifulbob@gmail.com>
date: 2026-06-08
---

# Rust CLI-Only Cargo Publish Plan

## Decision

The first crates.io release should publish one user-facing package:
`slophammer-rs`.

Do not publish the internal workspace crates just to make Cargo resolve the CLI
package. Publishing internal support crates would turn their public types into
long-term semver contracts before we have a real embedding use case.

This package layout and release path are now implemented in the Rust workspace.

## Target Shape

`cargo install slophammer-rs --locked` should install the checker.

The stable public contract is:

- CLI command names, flags, and exit codes
- `slophammer.yml` config format
- text, JSON, and SARIF report formats
- rule IDs and rule catalog output

The implementation can still stay modular, but those modules should be private
inside the published package unless a deliberate library API is added later.

## Implementation Plan

1. Refactor the Rust workspace into one publishable CLI package. Completed.
   Keep the code organized as internal modules under the `slophammer-rs`
   package, or use another single-package layout that lets `cargo package -p
   slophammer-rs --locked` succeed without unpublished internal crates.

2. Keep public API small. Completed.
   The CLI crate should not expose internal rule, scan, config, execution, or
   report APIs as stable libraries by accident.

3. Replace the Rust release workflow. Completed.
   The release workflow packages and publishes only `slophammer-rs`.

4. Replace the publish helper. Completed.
   The publish helper targets only the single CLI package.

5. Verify the package artifact before publishing. Completed.
   The release gate should run `cargo package -p slophammer-rs --locked`, test
   the verified package directory, install from the packaged crate or package
   path, and run the installed CLI smoke checks plus shared conformance.

## Non-Goals

- Do not publish internal crates as day-one library APIs.
- Do not add a public Rust library facade until there is a concrete downstream
  embedding need.
- Do not use crates.io publication as a way to preserve the current workspace
  layout.

## Acceptance

- `cargo package -p slophammer-rs --locked` succeeds before publish.
- `cargo test` succeeds from Cargo's verified package directory.
- The published package installs with `cargo install slophammer-rs --locked`.
- The installed binary passes `help`, `rules`, Rust fixture checks, direct Rust
  commands, and shared conformance.
- Internal implementation modules can change without semver promises beyond the
  CLI, config, report, exit-code, and rule-ID contracts.
