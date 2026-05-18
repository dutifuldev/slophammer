---
title: Release Hardening Roadmap
author: Bob <dutifulbob@gmail.com>
date: 2026-05-17
---

# Release Hardening Roadmap

Slophammer has working Go and TypeScript implementations. The next tranche
turns those implementations into cleanly installable, externally usable
products without weakening the repository quality gates.

## Goal

Make Slophammer easy for a user or agent to install, run, and trust outside
this repository.

The current repo already proves that the implementations work in source-tree
development. The next work proves that the packaged commands work after
installation, that both implementations keep the shared contract, and that CI
can publish useful quality signals.

## 1. Installation Hardening

The public commands should work as installed products, not only through local
source-tree commands.

Go requirements:

- Document `go install` for `slophammer-go`.
- Add a local release check that installs `slophammer-go` into a temporary
  `GOBIN`.
- Run `slophammer-go help`.
- Run `slophammer-go check` against at least one fixture repo.
- Run `slophammer-go dry`, `slophammer-go crap`, and `slophammer-go mutate
--scan` against this repo where practical.

TypeScript requirements:

- Keep `slophammer-ts` package-checked before publishing.
- Keep `slophammer-ts` as the public bin.
- Reserve the `slophammer` npm package name for a future umbrella package or
  default installer instead of publishing it as a TypeScript bin alias.
- Verify `npm pack`, temporary install, and command execution in CI.
- Run the packed command against at least one fixture repo.

Done means a user can install the command and run the documented examples
without depending on the repository checkout layout.

## 2. TypeScript Package Scope

The TypeScript package should publish only what the runtime needs.

Requirements:

- Add a `files` field to `typescript/package.json`.
- Include runtime `dist/src/**` files.
- Include the package README and package metadata.
- Exclude tests, fixtures, source files, coverage, node modules, and local
  development outputs.
- Keep `scripts/check-package-bin.ts` as the release artifact check.
- Make the package check assert that unwanted files are absent from the packed
  tarball.

Done means `npm pack --dry-run` shows a small runtime package instead of a
development snapshot.

## 3. Shared Conformance Test

The repo needs one clear test that proves the implementations still agree on
the shared Slophammer contract.

Requirements:

- Add a top-level conformance script.
- Run the Go implementation against shared fixtures.
- Run the TypeScript implementation against shared fixtures it supports.
- Compare normalized JSON reports against `fixtures/expected`.
- Assert rule IDs, severities, paths, messages, report `ok`, and exit codes.
- Keep implementation-specific behavior out of the shared expected reports.

Done means a future implementation can change internals without silently
changing the public contract.

## 4. Inspectable Product Commands

Agents should be able to inspect the checker without reading source files.

Commands to add:

```sh
slophammer-go rules [--format text|json]
slophammer-ts rules [--format text|json]
```

The commands should print the implemented rule catalog. Text output is for
humans. JSON output is for agents and automation.

Later command:

```sh
slophammer-go fixtures
```

`fixtures` is useful after the conformance layer exists. It should list the
fixture repos and the expected report files that define the shared contract.

Done means an agent can discover the available rules and fixture contract from
the CLI.

## 5. SARIF Code Scanning

Slophammer already writes SARIF. CI should prove that SARIF output can feed
GitHub code scanning.

Requirements:

- Add a CI step that runs Slophammer with `--format sarif`.
- Upload the SARIF artifact with GitHub's code scanning upload action.
- Keep JSON as the stable internal report model.
- Treat SARIF as an adapter, not as the rule engine's source of truth.

Done means Slophammer findings can appear in GitHub's security/code scanning UI
when a repo enables that workflow.

## 6. Cross-Language Strategy

The docs allow implementations to check more than one language. The repo needs
a practical first target.

Recommended near-term direction:

- Keep `slophammer-go` as the strongest single-binary checker.
- Let `slophammer-go` own Go checks first.
- Add TypeScript static checks to `slophammer-go` only when they reuse the
  shared rule IDs, config fields, report shape, and fixtures.
- Keep `slophammer-ts` native-first for TypeScript.
- Do not require every implementation to check every language.

Done means cross-language support grows deliberately instead of turning into
duplicated partial implementations.

## 7. Adoption Fixture

The agent-entrypoint story needs a realistic before/after example.

Requirements:

- Add a small fixture that resembles an existing messy repo.
- Keep the example small enough to review quickly.
- Show the initial Slophammer findings.
- Add an after-state or remediation notes that show what changed.
- Use it to test the agent entrypoint instructions.

Done means the repo demonstrates how an agent applies Slophammer standards to a
realistic existing project, not only to hand-built clean fixtures.

## Recommended Order

1. TypeScript package scope.
2. Installation hardening.
3. Shared conformance test.
4. `rules` commands.
5. SARIF code scanning.
6. Adoption fixture.
7. Cross-language expansion.

Do not start the Python implementation until the Go and TypeScript products are
installable, package-checked, and conformance-tested.

## Implementation Status

This roadmap is implemented in the release-hardening tranche.

- `go/scripts/check-go-install.sh` installs and exercises `slophammer-go`.
- `typescript/scripts/check-package-bin.ts` verifies the packed npm artifact.
- `scripts/check-conformance.mjs` verifies shared fixture reports and exit
  codes.
- `slophammer-go rules` and `slophammer-ts rules` print the implemented rule
  catalogs in text and JSON formats.
- CI generates and uploads Slophammer SARIF.
- `fixtures/repos/adoption-before` captures a realistic before-state fixture.
- `fixtures/repos/adoption-after` captures the matching clean after-state.
- `.github/workflows/go-release-dry-run.yml` validates Go release tags and the
  Go release path.
- README and agent docs point to installed commands and conformance checks.

## Required Follow-Up

The required next work is tracked in
[Required Next Work](2026-05-17-required-next-work.md). The important release
decision was that Go could be released first, while TypeScript stayed
package-checked until npm publication.

## Acceptance Checklist

- [x] `slophammer-go` has an install check.
- [x] `slophammer-ts` has a publish-artifact check.
- [x] TypeScript package output excludes development-only files.
- [x] Shared fixture conformance runs from one top-level command.
- [x] Both implementations expose a `rules` command.
- [x] Both implementations expose JSON rule catalog output.
- [x] CI proves SARIF upload compatibility.
- [x] A realistic adoption fixture exists.
- [x] A matching adoption after-state fixture exists.
- [x] Go release dry-run workflow exists.
- [x] Documentation points agents to the install and conformance workflow.
