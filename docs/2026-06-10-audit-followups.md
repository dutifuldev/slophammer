---
title: Audit Follow-ups
author: Onur Solmaz <onur.solmaz@huggingface.co>
date: 2026-06-10
status: active
---

# Audit Follow-ups

A repository audit on 2026-06-10 found contract-level inconsistencies between
the spec, the README, and the implementations, plus coverage gaps in the
shared fixture contract and CI. This plan turns those findings into ordered,
verifiable tasks.

Phases are ordered by how directly each protects the product contract. Tasks
within a phase are independent unless a dependency is stated.

## Phase 1: Contract Fixes

These are documented-versus-actual mismatches in the product contract. They
ship first because the repo's whole pitch is a trustworthy contract.

1. [x] Fix the stale Rust release status in `specs/PRODUCT.md`.

   The Release Policy section still says the Rust checker "is not published to
   crates.io yet" and prescribes source-tree installation. `slophammer-rs` is
   published (`rust/v0.1.0`, `rust/v0.1.1` tags, `rust-release.yml`), and the
   README and `docs/AGENT_ENTRYPOINT.md` already say so.

   - Rewrite the paragraph: `slophammer-rs` is released to crates.io; the
     primary install is `cargo install slophammer-rs --locked`; the source-tree
     install is the development alternative.
   - Keep the single-package policy text; it is still accurate.

   Done when: `specs/PRODUCT.md`, `README.md`, and
   `docs/AGENT_ENTRYPOINT.md` agree on Rust release status and install
   commands.

2. [x] Implement `check --only <rule-id>` in the Go checker.

   The README advertises `--only` as part of the shared command surface, and
   TypeScript (`typescript/src/cli/cli.ts`) and Rust
   (`rust/crates/slophammer-cli/src/main.rs`) implement it. Go does not
   (`go/internal/cli/cli.go` parses only `--format`, `--execute`,
   `--coverage-profile`).

   - Parse repeated `--only <rule-id>` arguments in `go/internal/cli`,
     matching the TypeScript semantics: unknown rule IDs are usage errors
     (exit 2), findings filter to the listed rules, exit codes unchanged
     otherwise.
   - Thread the filter through `go/internal/app` into rule evaluation.
   - Update both usage strings in `go/internal/cli/cli.go`.
   - Add CLI-level unit tests mirroring the TypeScript `--only` tests.

   The alternative — removing `--only` from the README's shared surface — is
   rejected: two of three implementations already support it and the README
   sells a uniform surface.

   Done when: `slophammer-go check <path> --only repo.readme-required` works
   against `fixtures/repos/missing-readme` and the Go test suite covers
   accepted and rejected rule IDs.

3. [x] Update the README's description of the conformance script.

   `README.md` says `scripts/check-conformance.mjs` covers "the Go and
   TypeScript implementations"; the script also runs the full Rust fixture
   set plus two Rust config-error cases.

   Done when: the sentence names all three implementations.

4. [x] Document the real per-implementation command surface in
   `specs/PRODUCT.md`.

   The direct-commands paragraph lists `slophammer-ts boundaries` and
   `slophammer-rs dry|boundaries|unsafe`, but omits Go's direct commands
   (`dry`, `crap`, `mutate --scan`) that CI and the agent entrypoint depend
   on, TypeScript's `dry` and `typescript` subcommands, and Go's
   `--coverage-profile` flag.

   - Enumerate each implementation's direct commands in PRODUCT.md.
   - Document `--coverage-profile` in `go/README.md` (and PRODUCT.md if it is
     meant to be contract rather than implementation detail).

   Done when: every command exercised in `.github/workflows/ci.yml` and
   `docs/AGENT_ENTRYPOINT.md` appears in the spec or an implementation README.

## Phase 2: Conformance and CI Coverage

5. [x] Bring TypeScript fixture coverage up to Go/Rust parity.

   Go has `go-missing-*` fixtures for 9 of 10 rules and Rust has 11; the
   TypeScript set has only `typescript-missing-any-rule`, `-missing-dry`, and
   `-missing-strict`. Ten TS rules have no cross-implementation fixture.

   - Add `fixtures/repos/typescript-missing-{coverage,lint,format,typecheck,
     test,package,mutation,complexity,unsafe-types}` and
     `typescript-bad-dependency`, each with the matching
     `fixtures/expected/*.json`.
   - Register the new fixtures in `scripts/check-conformance.mjs`
     (`typeScriptFixtures`).
   - Follow the existing fixture shape: minimal repo, one missing control per
     fixture.

   Done when: every `ts.*` rule in `specs/rules.json` has at least one
   fixture that triggers it, and conformance passes.

6. [x] Dogfood the TypeScript and Rust checkers on this repository in CI.

   Resolved during implementation: the audit premise was wrong. Both
   implementations already gate the repo in CI — the `rust-implementation`
   job runs `slophammer-rs check ..` as its final step, and the TypeScript
   `npm run check` chain includes `self-check`
   (`node dist/src/cli/main.js check ..`). Both verified locally with exit 0.
   No CI change was needed.

7. [x] Replace the hand-maintained `test -f` list with a real link check.

   The "Check markdown links to local files" step in
   `.github/workflows/ci.yml` is a manual file list that already omits
   `docs/AGENT_ENTRYPOINT.md` — the README's primary entrypoint — and will
   keep drifting.

   - Determine whether `npx @simpledoc/simpledoc check` already validates
     relative markdown links. If yes, delete the redundant `test -f` lines.
   - If not, add an offline markdown link checker (for example `lychee
     --offline` or a small node script) over `**/*.md`.
   - Keep the structural guards that are not link checks
     (`test ! -d rust/crates/slophammer-core`, packaging-shape assertions);
     move them to a clearly named step.

   Done when: removing a file that any markdown document links to fails CI
   without anyone editing a list.

## Phase 3: Documentation Hygiene

8. [x] Mark superseded dated plan docs.

   `docs/` mixes evergreen reference docs with dated plans, some now
   misleading when read cold (`2026-05-17-required-next-work.md` predates the
   Rust implementation; the two Rust plan docs describe completed work).

   - Add a `status:` field to dated plan frontmatter — `completed`,
     `superseded by <doc>`, or `active` — and a one-line note at the top of
     completed plans.
   - Alternative: move completed plans to `docs/archive/`. The status field
     is preferred because external links into `docs/` keep working.

   Done when: an agent reading any dated plan can tell without other context
   whether to act on it.

9. [x] Bring the TypeScript-only fixture into the shared contract.

   `fixtures/repos/typescript-duplicate-blocks` was used only by TypeScript
   unit tests (`typescript/tests/dry.test.ts`, `cli.test.ts`), had no
   `fixtures/expected/` entry, and was not in the conformance script.

   Resolution: promoted into the contract rather than moved. Moving it under
   `typescript/tests/fixtures/` would break the repository self-check — the
   checker only ignores top-level `fixtures/` and `templates/` paths, so the
   relocated fixture would be scanned as a real project scope. Its static
   `check` report is deterministic, so it now has
   `fixtures/expected/typescript-duplicate-blocks.json` and a
   `typeScriptFixtures` conformance entry, while the unit tests keep using it
   for `dry` behavior.

   Done when: everything under `fixtures/` is exercised by
   `scripts/check-conformance.mjs`.

## Phase 4: Ergonomics and Direction

10. [x] Add a root `make check` entry point.

    Each directory has its own gate (`npm run check`, golangci plus shell
    scripts, cargo commands); only CI composes them. Add a root Makefile (or
    justfile) with `check-go`, `check-typescript`, `check-rust`,
    `check-templates`, `conformance`, and an aggregate `check`, each
    mirroring the corresponding CI job's commands. Reference it from
    `AGENTS.md` so agents run the full gate locally.

    Done when: a clean checkout passes `make check` and the targets match
    what CI runs.

11. [x] Decide the config-dialect question explicitly.

    `slophammer.yml` uses flat keys for Go and TypeScript
    (`coverage_threshold`, `complexity_max`, `crap_max_score`) and nested
    keys for Rust (`coverage.threshold`, `complexity.cognitive_max`,
    `mutation.targets`). The divergence is documented in `specs/CONFIG.md`,
    so it is consistent today — but it triples the schema surface for the
    planned umbrella `slophammer` package and makes cross-language copying
    error-prone.

    - Write a short decision doc: either converge on the nested shape with a
      deprecation window for flat keys, or declare the divergence permanent
      and say why.
    - Do not churn implementations until that decision is recorded.

    Done when: the decision doc exists and `specs/CONFIG.md` links to it.

## Sequencing Notes

- Tasks 1–4 are independent and small; they can land as one docs-and-Go PR
  series.
- Task 6 depends on the repo actually passing `slophammer-ts` and
  `slophammer-rs` checks; do the local run first and budget for findings.
- Task 5 is mechanical but large in file count; it can proceed in parallel
  with everything else.
- Task 11 deliberately produces a decision, not code.
