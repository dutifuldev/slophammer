---
title: Rust Implementation Plan
author: Bob <dutifulbob@gmail.com>
date: 2026-06-08
---

# Rust Implementation Plan

## Goal

Add a production-ready Rust implementation of the Slophammer product contract.

The implementation should be a first-class shipped command named
`slophammer-rs`. It should follow the same shared contract as the Go and
TypeScript implementations:

- stable rule IDs
- strict `slophammer.yml` config validation
- text, JSON, and SARIF reports
- stable exit codes
- shared fixture conformance
- pure rule evaluation over typed repository snapshots
- opt-in process execution only through `check --execute`

This is not a staged implementation plan. The target is the long-term shape.
Work can still land in coherent commits, but the design should not rely on
temporary command names, temporary rule IDs, or intentionally incomplete policy.

## Non-Goals

- Do not rewrite Slophammer's shared product contract.
- Do not make Rust a special case with different report or exit-code behavior.
- Do not implement a Rust parser, Cargo resolver, formatter, linter, coverage
  engine, or mutation engine from scratch.
- Do not shell out in static `check`.
- Do not publish public library APIs unless there is a concrete downstream need.

## Product Surface

The public command should be:

```sh
slophammer-rs check <path>
slophammer-rs check <path> --format text
slophammer-rs check <path> --format json
slophammer-rs check <path> --format sarif
slophammer-rs check <path> --execute
slophammer-rs check <path> --only <rule-id>
slophammer-rs explain <rule-id>
slophammer-rs rules
slophammer-rs rules --format json
slophammer-rs dry <path>
slophammer-rs boundaries <path>
slophammer-rs unsafe <path>
```

`check` is the main gate. Direct commands are allowed only when they expose
Slophammer-owned policy directly and preserve the normal finding and exit-code
model.

## Repository Layout

Create a top-level Rust implementation:

```text
rust/
├── Cargo.toml
├── crates/
│   ├── slophammer-core/
│   ├── slophammer-config/
│   ├── slophammer-scan/
│   ├── slophammer-report/
│   ├── slophammer-rust/
│   ├── slophammer-exec/
│   ├── slophammer-app/
│   └── slophammer-cli/
└── tests/
```

The crate split is an architecture boundary, not a promise to create a public
library ecosystem.

Default dependency direction:

```text
slophammer-cli
-> slophammer-app
   -> slophammer-config
   -> slophammer-scan
   -> slophammer-rust
   -> slophammer-report
   -> slophammer-exec

slophammer-config -> slophammer-core
slophammer-scan -> slophammer-core
slophammer-rust -> slophammer-core
slophammer-report -> slophammer-core
slophammer-exec -> slophammer-core
```

`slophammer-core` is the bottom layer. It owns shared data types such as rule
definitions, findings, reports, severities, and exit-code concepts. The
important boundary is that rule packages do not depend on filesystem, process,
terminal, or report-rendering adapters.

## Internal Boundaries

Keep the implementation model aligned with the existing Slophammer shape:

```text
CLI
-> app orchestration
-> scanner and typed repository snapshot
-> config parser
-> pure rule evaluation
-> report renderer
```

Static `check` must be deterministic and read-only. It may read files, parse
manifests, parse Rust source, inspect workflows, and evaluate config. It must
not run Cargo, build scripts, network access, package installation, tests, or
third-party commands.

`check --execute` is the only mode that runs tools. Execution adapters should
return Slophammer findings rather than printing tool output as the primary API.

## Dependency Posture

Use existing Rust crates and ecosystem tools where they are the normal answer.
Slophammer should own policy, not commodity parsing or build tooling.

Default packages and tools:

| Need                  | Use                                      |
| --------------------- | ---------------------------------------- |
| CLI parsing           | `clap`                                   |
| serialization         | `serde`, `serde_json`                    |
| errors                | `thiserror` in crates, narrow `anyhow` at CLI edges |
| Cargo workspace model | `cargo_metadata`                         |
| dependency graph      | `guppy` only if `cargo_metadata` is not enough |
| repository walking    | `ignore`                                 |
| UTF-8 paths           | `camino`                                 |
| glob matching         | `globset`                                |
| TOML manifests        | `toml_edit`                              |
| Rust AST parsing      | `syn`                                    |
| token-oriented parsing | `tree-sitter-rust` only if needed for DRY |
| coverage execution    | `cargo-llvm-cov`                         |
| mutation execution    | `cargo-mutants`                          |
| formatting            | `cargo fmt --check` / `rustfmt`          |
| linting               | `cargo clippy`                           |
| dependency audit      | `cargo audit` and/or `cargo deny`        |

YAML should be isolated behind `slophammer-config`. Choose a maintained parser
after checking current dependency health. The rest of the code should not care
which YAML package backs `slophammer.yml`.

Before adding any dependency, answer:

- Is this solving a commodity problem?
- Is it maintained enough for a quality gate?
- Can it be isolated behind one crate boundary?
- Does it reduce custom code without hiding Slophammer policy?

Do not add dependencies for simple string matching, small data transforms, or
one-off wrappers around process execution.

## Slophammer-Owned Logic

Slophammer should implement:

- rule definitions and rule IDs
- strict config schema and threshold validation
- report rendering
- fixture conformance
- static evidence detection in scripts, package metadata, and workflows
- Rust dependency-boundary policy
- unsafe-code policy
- native DRY findings when external tools do not provide the Slophammer report
  contract directly

Slophammer should not implement:

- Cargo dependency resolution
- general Rust parsing
- formatting
- linting
- coverage measurement
- mutation testing
- vulnerability databases

## Rust Rule Set

Add Rust rules to the shared rule catalog:

| Rule ID                                  | Finding path          | Message |
| ---------------------------------------- | --------------------- | ------- |
| `rust.manifest-required`                 | `Cargo.toml`          | `Rust projects must include Cargo.toml` |
| `rust.msrv-required`                     | `Cargo.toml`          | `Rust projects must declare a minimum supported Rust version` |
| `rust.check-required`                    | `.github/workflows`   | `Rust projects must declare cargo check in CI or scripts` |
| `rust.fmt-required`                      | `.github/workflows`   | `Rust projects must declare cargo fmt --check in CI or scripts` |
| `rust.clippy-required`                   | `.github/workflows`   | `Rust projects must declare cargo clippy in CI or scripts` |
| `rust.test-required`                     | `.github/workflows`   | `Rust projects must declare cargo test in CI or scripts` |
| `rust.coverage-required`                 | `.github/workflows`   | `Rust projects must declare a coverage gate` |
| `rust.complexity-required`               | `.github/workflows`   | `Rust projects must enforce complexity limits` |
| `rust.dry-required`                      | `.github/workflows`   | `Rust projects must declare a DRY check` |
| `rust.mutation-required`                 | `.github/workflows`   | `Rust projects must declare mutation testing` |
| `rust.unsafe-policy-required`            | `slophammer.yml`      | `Rust projects must declare and respect an unsafe-code policy` |
| `rust.dependency-audit-required`         | `.github/workflows`   | `Rust projects must declare dependency audit checks` |
| `rust.dependency-boundaries-required`    | `slophammer.yml`      | `Rust projects must respect configured dependency boundaries` |

Rust detection should activate when a repository contains `Cargo.toml`, Rust
source files, or inspectable Rust commands.

## Config Shape

Extend `slophammer.yml` with a Rust section:

```yaml
rust:
  coverage:
    threshold: 85
    paths:
      - rust/crates
    exclude:
      - "target/**"
      - "fixtures/**"
  complexity:
    cognitive_max: 8
  targets:
    - rust/crates
  exclude:
    - "target/**"
    - "fixtures/**"
  dry:
    max_findings: 0
    paths:
      - rust/crates
    exclude:
      - "**/*_test.rs"
      - "fixtures/**"
      - "target/**"
    copied_blocks:
      enabled: true
      min_tokens: 100
  unsafe:
    policy: forbid
    allow:
      - path: rust/crates/slophammer-rust/src/ffi.rs
        reason: "narrow FFI boundary"
  mutation:
    targets:
      - rust/crates/slophammer-rust/src
    exclude:
      - "rust/crates/slophammer-rust/src/generated/**"
  dependency_boundaries:
    - from: rust/crates/slophammer-rust
      allow:
        - rust/crates/slophammer-core
        - rust/crates/slophammer-config
        - rust/crates/slophammer-scan
        - rust/crates/slophammer-report
```

Hard recommended bounds:

- `rust.coverage.threshold` must be at least `85`.
- `rust.complexity.cognitive_max` must be at most `8`.
- `rust.dry.max_findings` must be `0` for production code.
- `rust.unsafe.policy` defaults to `forbid` when configured.

Projects may choose stricter values. They may not weaken Slophammer's
recommended bounds through config.

Config validation must reject unknown keys. Invalid config exits with code `2`.

## Static Rule Semantics

Static checks should inspect normal project evidence:

- `Cargo.toml` and workspace manifests
- `rust-toolchain.toml`
- `.cargo/config.toml`
- GitHub Actions workflows
- scripts under `scripts/`
- package or task-runner command files when present
- `slophammer.yml`
- Rust source files under configured targets

Static rules should accept wrapper scripts only when the inspected wrapper
clearly executes the required command. Do not accept comments as evidence.

Coverage should recognize `cargo-llvm-cov` as the preferred gate and accept
other credible Rust coverage gates only when an enforceable threshold is visible.

Mutation should recognize `cargo-mutants` as the preferred gate. Static checks
may accept nightly or manual workflow declarations because mutation is expensive.

Dependency audit should recognize `cargo audit` and `cargo deny`. Prefer
`cargo deny` when the repository needs policy over licenses, advisories,
sources, or bans.

Complexity should prefer Clippy-backed configuration when possible. If Rust's
available lint surface cannot express the whole Slophammer complexity policy,
implement the remaining static AST metric in `slophammer-rust` with `syn`.

## Direct Commands

### `slophammer-rs dry`

Run Slophammer-owned duplicate detection over configured Rust production
targets. Prefer reused parsing/tokenization crates over custom parsers. Produce
normal Slophammer findings and support JSON/text output.

The long-term DRY check should cover:

- copied blocks
- near-identical function or method bodies when reliable enough
- production code only by default
- deterministic ordering

### `slophammer-rs boundaries`

Evaluate `rust.dependency_boundaries` against local crate/module imports.

Use Cargo metadata for package and workspace structure. Use parsed Rust source
for local module imports when needed. External crates are ignored unless a
future rule explicitly covers them.

### `slophammer-rs unsafe`

Evaluate `rust.unsafe` policy.

Default behavior should make unsafe code visible and explainable. Under
`policy: forbid`, any unsafe block, unsafe function, unsafe trait, unsafe impl,
or unsafe extern block in configured targets is a finding unless covered by a
specific allow entry with a reason.

## Execute Mode

`check --execute` should run from the discovered Cargo workspace root.

Default execute checks:

```sh
cargo fmt --check
cargo clippy --workspace --all-targets -- -D warnings
cargo test --workspace --all-targets
cargo llvm-cov --workspace --fail-under-lines 85
cargo audit
```

When configured, execute mutation checks with:

```sh
cargo mutants --workspace
```

Execute-mode adapters must:

- run only when `--execute` is present
- use configured thresholds and target scope
- capture command failures as Slophammer findings
- keep stdout/stderr available for humans without making it the report contract
- support fake-runner tests without invoking real tools

## Fixtures And Conformance

Add Rust fixture repositories under `fixtures/repos/` and expected reports under
`fixtures/expected/`.

Required fixtures:

- clean Rust workspace
- missing `Cargo.toml`
- missing `rust-version` or equivalent MSRV declaration
- missing CI
- missing format check
- missing Clippy check
- missing tests
- missing coverage gate
- missing mutation declaration
- unsafe code without allow entry
- dependency-boundary violation
- invalid Rust config
- unknown Rust config key
- adoption before/after example

Update shared conformance so Rust runs beside Go and TypeScript. The
conformance script should verify:

- JSON report shape
- finding IDs
- finding paths
- finding messages
- clean exit code `0`
- findings exit code `1`
- usage/config/runtime error exit code `2`

## CI And Release Checks

Rust CI should run:

```sh
cargo fmt --check
cargo clippy --workspace --all-targets -- -D warnings
cargo test --workspace --all-targets
cargo test --workspace --doc
cargo llvm-cov --workspace --fail-under-lines 85
cargo install --path rust/crates/slophammer-cli --locked
slophammer-rs rules --format json
slophammer-rs check fixtures/repos/rust-clean --format json
node scripts/check-conformance.mjs
```

Add release dry-run coverage before publishing:

- verify package metadata
- verify `cargo install` from the package path
- verify command help and rule catalog
- verify fixture checks from the installed binary
- verify lockfile reproducibility
- verify the CLI-only crates.io package/publish sequence for `slophammer-rs`

Do not publish until the installed artifact is proven by CI. The merged Rust
implementation is source-installable, but not published to crates.io yet.

Before the first crates.io release, refactor the Rust implementation to publish
only the user-facing `slophammer-rs` Cargo package. Do not publish internal
workspace crates just to satisfy Cargo dependency resolution; that would create
unwanted public library APIs. The current multi-crate publish workflow must be
retired or replaced before release. See
[Rust CLI-only Cargo publish plan](2026-06-08-rust-cli-only-cargo-publish-plan.md).

## Documentation Updates

The implementation should update:

- `README.md` product status and install instructions
- `specs/PRODUCT.md` command and release policy
- `specs/RULES.md` Rust rule catalog and descriptions
- `specs/CONFIG.md` Rust config shape and validation rules
- `specs/REPORT_FORMAT.md` only if Rust needs report behavior already shared
- `specs/EXIT_CODES.md` only if behavior changes, which it should not
- `docs/AGENT_ENTRYPOINT.md` language selection table and Rust CI contract
- `docs/IMPLEMENTATION_MODEL.md` Rust implementation section
- `slophammer.yml` for Slophammer's own Rust implementation policy

## Implementation Checklist

- [ ] Add Rust product docs and shared Rust rule definitions.
- [ ] Scaffold `rust/` workspace and crate boundaries.
- [ ] Implement shared report, rule, and exit-code model.
- [ ] Implement strict Rust config parsing and validation.
- [ ] Implement scanner and typed Rust repository snapshot.
- [ ] Implement shared repository rules.
- [ ] Implement Rust static rules.
- [ ] Implement direct `dry`, `boundaries`, and `unsafe` commands.
- [ ] Implement execute-mode adapters with fake-runner tests.
- [ ] Add Rust fixtures and expected reports.
- [ ] Wire Rust into shared conformance.
- [ ] Add Rust CI and install checks.
- [ ] Update agent entrypoint and product documentation.
- [ ] Run full Go, TypeScript, Rust, and conformance gates.

## Acceptance Criteria

The Rust implementation is complete when:

- `slophammer-rs` exposes the documented command surface.
- `slophammer-rs rules --format json` prints Rust and shared repo rules.
- `slophammer-rs explain <rule-id>` works for every Rust rule.
- Static `check` is read-only and deterministic.
- `check --execute` runs Cargo/tool adapters only when requested.
- JSON reports match the shared Slophammer report contract.
- SARIF reports are produced from the same finding model.
- Strict config rejects unknown Rust keys and weak thresholds.
- Rust fixtures pass shared conformance.
- CI proves formatting, Clippy, tests, coverage, installability, and
  conformance.
- No custom code duplicates a mature ecosystem package without a documented
  Slophammer-specific reason.
