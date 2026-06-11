---
title: Enforcement and Adoption Plan
author: Onur Solmaz <onur.solmaz@huggingface.co>
date: 2026-06-11
status: active
---

# Enforcement and Adoption Plan

Slophammer's goal is quality enforced by machinery, not by hoping someone
reads the diff. Testing the released checkers against real-world repositories
and crafted counterexamples shows where the current rules fall short of that
goal. This plan turns those findings into ordered, verifiable tasks.

Two findings motivate the top of the list. First, the command-presence rules
credit declarations that cannot execute: a workflow with a never-matching
branch filter, `if: false` on the job, and `continue-on-error: true` on every
step still passes `check` with exit 0. Second, config scope can quietly carve
real code out of checking: pointing `dry.paths` at an innocuous directory
hides genuine duplication, and nothing in the report says the production tree
went unscanned. Both holes matter because the adversary in Slophammer's
threat model is an agent optimizing to make the check pass.

## Phase 1: Make Declarations Bind

1. [ ] Refuse to credit CI steps that cannot run or cannot fail.

   Every `*-required` command rule accepts any command-file text containing
   the right command. Evidence collection lives in
   `go/internal/rules/go_command_files.go` (`commandFiles`, fed by
   `snapshot.WorkflowFiles()` plus Makefile/Taskfile/justfile and `scripts/`
   trees), `typescript/src/repo/repo.ts` (`commandFiles`, consumed by
   `commandSegmentsWithPackageExpansion` in `typescript/src/rules/rules.ts`),
   and the evidence helpers in
   `rust/crates/slophammer-cli/src/rust_rules/mod.rs`. None of them parse
   workflow structure; they strip comments and split command segments.

   Change: parse workflow YAML before extracting command segments, and drop
   neutralized evidence:

   - steps with a literal `continue-on-error: true`, and all steps of jobs
     with a literal job-level `continue-on-error: true`
   - jobs with a literal-false `if:` (`false`, `${{ false }}`)
   - workflows whose triggers cannot fire for integration. Version 1 keeps a
     workflow when `on` includes `pull_request`, `merge_group`, `schedule`,
     or a `push` with no `branches` filter or a filter containing a wildcard
     or a plausible integration branch (`main`, `master`, `trunk`,
     `develop`). Everything else is non-binding.

   Decisions already made:

   - Expressions are only neutralizing when literal. A conditional
     `continue-on-error: ${{ matrix.experimental }}` or a non-constant `if:`
     keeps the step credited; the checker must not evaluate expressions.
   - Reusable workflows (`uses:` at job level) stay credited in version 1;
     following `uses:` references is out of scope.
   - Makefiles and scripts remain credited without reachability analysis.
     The product contract deliberately accepts "in CI or scripts"; tying
     scripts to workflow invocation is a possible version 2 tightening and
     would need its own decision.
   - Unparseable workflow YAML stays credited as raw text (today's
     behavior), so this change can only remove false passes, not add false
     failures on exotic-but-working setups.

   Implementation per checker: a shared "binding workflow content" filter
   applied where workflow files enter the command-file set — new
   `go/internal/rules/go_workflow_binding.go`, a workflow filter in
   `typescript/src/repo/repo.ts` (the `yaml` dependency is already present),
   and a module beside `rust_rules/mod.rs` (`yaml_serde` is already a
   dependency). Makefile/script files bypass the filter.

   Fixtures: `go-neutralized-ci`, `typescript-neutralized-ci`, and
   `rust-neutralized-ci` — copies of the clean fixtures with neutralized
   workflows — must fail with the same findings as their `missing-*`
   counterparts. Register all three in `scripts/check-conformance.mjs` and
   add a per-neutralization-mechanism unit test in each implementation
   (filter, `if`, `continue-on-error` at step and job level).

   Done when: the crafted counterexample produces findings in every
   implementation, the three fixtures lock the behavior into conformance,
   and the clean fixtures still pass unchanged.

2. [ ] Add a rule that the checker itself must run in CI.

   A repository can carry `slophammer.yml` and `AGENTS.md` while no CI step
   ever executes a Slophammer checker; the config is decoration and nothing
   reports it. Surveyed adopters include exactly this state.

   New shared rule `repo.slophammer-ci-required`, severity `error`,
   implemented by all checkers. It fires only when `slophammer.yml` or
   `slophammer.yaml` is present (config-gated, like dependency boundaries).
   It passes when binding command evidence (task 1 semantics) invokes any
   released checker. Accepted invocation shapes, mirroring the regex style
   of `hasTypeScriptDryCommand` in `typescript/src/rules/rules.ts`:

   - `slophammer-go|slophammer-ts|slophammer-rs|slophammer-py ... check`
   - `go run github.com/dutifuldev/slophammer/go/cmd/slophammer-go@<v>`
   - `npx|pnpm dlx|npm exec slophammer-ts[@<v>]`
   - package-script expansion, so `npm run <script>` counts when the script
     invokes the checker

   Spec and fixture changes:

   - `specs/rules.json` and `specs/RULES.md` gain the rule (40 rules total).
   - Every existing `*-clean` fixture that carries a `slophammer.yml` must
     gain a checker invocation in its workflow, or the rule makes it fail.
     This is an intentional cutover: config without enforcement becomes a
     finding. Fixture expected files regenerate accordingly.
   - New fixtures: `unenforced-config` (config present, CI present, no
     checker invocation — fails with exactly this rule) and the updated
     clean fixtures as the passing case.

   Done when: `unenforced-config` fails with only
   `repo.slophammer-ci-required` across all implementations and conformance
   passes with the regenerated expected reports.

3. [ ] Make scope carve-outs visible and justified.

   Config can restrict `paths`, `targets`, and `exclude` without limit, so
   findings can be hidden by retargeting scope instead of weakening
   thresholds — thresholds are validated, scope is not.

   Two complementary changes:

   - Justified production excludes. An `exclude` entry stays a plain string
     when it matches the conventional non-production list (test globs such
     as `**/*_test.go` and `**/*.test.ts`, and `fixtures/`, `templates/`,
     `testdata/`, `dist/`, `build/`, `coverage/`, `target/`,
     `node_modules/`, `vendor/`, `generated` segments). Any other exclude
     must use the object form, following the `rust.unsafe.allow` precedent:

     ```yaml
     dry:
       paths:
         - src
       exclude:
         - "**/*.test.ts"
         - pattern: "src/vendored_parser/**"
           reason: vendored upstream code, synced verbatim
     ```

     Strict validation rejects a production-matching string exclude with
     `<section>.exclude requires a reason for production paths`. The
     classification list is shared, spelled out in `specs/CONFIG.md`, and
     identical in all three validators (`go/internal/config/config.go`,
     `typescript/src/config/config.ts`,
     `rust/crates/slophammer-cli/src/config.rs`).

   - Scope coverage in the report. When a scoped check runs (native DRY,
     coverage, CRAP, mutation), the report gains an additive `scope` block:

     ```json
     "scope": { "scanned": 42, "production_files": 45 }
     ```

     where `production_files` counts the ecosystem's source files in the
     repository after the conventional non-production list, and `scanned`
     counts files actually in scope. The text renderer prints
     `DRY scanned 42 of 45 production files`. `specs/REPORT_FORMAT.md`
     documents the field; it is additive, so existing consumers keep
     working, but expected fixture reports regenerate.

   Decisions already made:

   - No hard floor on scope coverage in this task. The number being visible
     in the report (and in CI logs) is the version 1 goal; a configurable
     minimum can come later if visibility proves insufficient.
   - `paths` pointing at a nonexistent directory already fails loudly and
     keeps doing so.

   Done when: the carve-out counterexample (real duplication in `src/`,
   `dry.paths` pointed at a trivial directory) either fails validation for a
   reasonless production exclude or produces a report whose `scope` block
   shows the production tree unscanned, and conformance covers both the
   object-form exclude and the `scope` field.

## Phase 2: Make Adoption Possible

4. [ ] Add a baseline ratchet mode for existing repositories.

   Thresholds are absolute and weakening is forbidden, so a repository that
   is not already clean has no incremental path in: surveyed non-adopted
   repositories fail with anywhere from one to over seventy findings, and
   the only options today are "fix everything first" or "do not adopt".
   Baselines are how mypy, ESLint bulk suppressions, and similar gates
   solved the same problem.

   Contract, to be specified in a new `specs/BASELINE.md`:

   - The baseline lives in a checked-in `slophammer-baseline.json`:

     ```json
     {
       "version": 1,
       "findings": [
         { "rule_id": "go.coverage-required", "path": ".github/workflows" }
       ]
     }
     ```

     Matching is on `rule_id` plus `path` (not `message`, so message
     rewording across checker versions does not invalidate baselines).
     Duplicate keys collapse.

   - `check --baseline` reads the file and exits `0` when current findings
     are a subset of the baseline, `1` when any non-baselined finding
     exists. Baselined findings still appear in the report, marked
     `"baselined": true`, and the text renderer always prints the debt:
     `12 findings baselined; 0 new`. Debt is never silent.
   - Stale entries (baselined findings that no longer occur) are an error
     (exit `2`, `baseline contains resolved findings; rewrite it`), which is
     what makes the ratchet shrink-only.
   - `check --baseline-write` writes the file from current findings, but
     refuses to write a superset of an existing baseline (exit `2`), so debt
     can be recorded once and reduced, never grown.
   - The flag is explicit; the file is never auto-read. An agent must not be
     able to silence findings by committing a baseline that the standard
     `check` invocation then honors invisibly.
   - Composition: `--baseline` combines with `--only` and `--execute`
     (matching applies after filtering). SARIF output maps baselined
     findings to SARIF `suppressions`, which code scanning already
     understands.

   Implementation: baseline read/write and matching live beside report
   assembly in each `app` layer (`go/internal/app/app.go`,
   `typescript/src/app/app.ts`, `rust/crates/slophammer-cli/src/app.rs`);
   the finding model gains the optional `baselined` field. Fixtures:
   `adoption-baseline` (failing repo plus baseline file → exit 0),
   a regression case (one finding not in baseline → exit 1), and a stale
   case (baseline lists a resolved finding → exit 2), registered for all
   implementations.

   Done when: the three fixture cases hold in every implementation and the
   format is specified well enough that a baseline written by one checker is
   readable by the others.

5. [ ] Document version pinning as the supported consumption pattern.

   Slophammer makes breaking releases deliberately and ships no
   compatibility shims, but neither the README nor the Agent Entrypoint says
   how to consume it safely; CI integrations that install `@latest` absorb
   breaking changes mid-pipeline, and surveyed integrations do exactly that.

   - Add a short "Pinning" note to the README install section and a matching
     instruction in `docs/AGENT_ENTRYPOINT.md`: pin an exact version in CI
     (`go run .../slophammer-go@v0.2.0`, `npx slophammer-ts@0.2.0`,
     `cargo install slophammer-rs --version 0.2.0 --locked`), ideally behind
     one env var so upgrades are single-line, and upgrade deliberately —
     strict config validation fails loudly across breaking releases by
     design.
   - Update the copy-paste agent block to tell agents to pin the version
     they verified against.

   Done when: both documents show pinned commands for all released checkers
   and the paste block mentions pinning.

## Phase 3: Cover the Code Agents Actually Write

6. [ ] Ship `slophammer-py`.

   Python is the largest agent-coding ecosystem and the one planned
   implementation that does not exist; polyglot repositories with Python
   components have no gate for them today.

   Scope, mirroring the TypeScript checker's shape:

   - New `python/` implementation directory (src layout, `pyproject.toml`,
     same internal boundaries: CLI → app → scan → config → rules → report).
     The existing `templates/python` stays a template; the checker is new
     code held to Slophammer's own gates (coverage ≥ 85, complexity ≤ 8,
     DRY 0, mutation declared).
   - Rule set `py.*`, 13 rules at TypeScript parity:
     `py.project-required` (`pyproject.toml`), `py.typecheck-required`
     (mypy or pyright, strict-ish flags), `py.types-strict-required`
     (e.g. `disallow_untyped_defs`/pyright strict), `py.lint-required` and
     `py.format-required` (Ruff check and format, tool-agnostic where
     reasonable), `py.test-required` (pytest and friends),
     `py.coverage-required` (`fail_under`/`--cov-fail-under` ≥ 85),
     `py.complexity-required` (Ruff `C901` max ≤ 8 or radon),
     `py.dry-required` plus a native copied-block port,
     `py.mutation-required` (mutmut or cosmic-ray declared),
     `py.dependency-audit-required` (pip-audit or uv audit), and
     `py.dependency-boundaries-required` (import parsing against the shared
     boundary config).
   - Config: a `python:` section in the shared nested shape; validators in
     the other three checkers learn its allowed keys, exactly as they
     cross-validate each other today.
   - Conformance: `python-clean`, per-rule `python-missing-*` fixtures, and
     a `python-bad-dependency` fixture, all registered in
     `scripts/check-conformance.mjs`; specs `rules.json`/`RULES.md` grow the
     `py.*` rows.
   - Release: PyPI package `slophammer-py` via trusted publishing, a
     `python-release.yml` workflow cloned from the TypeScript one (tag shape
     `python-pkg/vX.Y.Z` or similar — decide against colliding with the
     `templates/python` history), GitHub release created by the workflow.

   This is the largest task in the plan and should be sequenced as its own
   dated implementation plan once Phase 1 lands; this entry fixes the scope
   and the contract, not the schedule.

   Done when: `slophammer-py check` passes shared conformance with a fixture
   set at parity with the other ecosystems and the checker passes its own
   gates in CI.

7. [ ] Build the `slophammer` umbrella dispatcher.

   Polyglot repositories need every per-language checker installed and wired
   by hand; in practice one language gets gated and the rest go unchecked.
   The reserved `slophammer` npm name becomes a thin dispatcher:

   - Detection by ecosystem markers, reusing the spec's project-detection
     semantics: `go.mod` → `slophammer-go`, TypeScript signals →
     `slophammer-ts`, `Cargo.toml` → `slophammer-rs`, `pyproject.toml` →
     `slophammer-py`.
   - For each detected ecosystem, resolve the checker on `PATH` and run
     `check <path> --format json`, forwarding `--execute`, `--only`, and
     `--format`. The dispatcher never installs checkers itself.
   - A detected ecosystem with no resolvable checker is itself a finding
     (`repo.checker-missing`, severity error), not a silent skip — the
     dispatcher must be unable to under-report.
   - Reports merge into one finding list (rule IDs are already namespaced);
     exit code is the maximum of the children's. SARIF output emits one run
     per tool, which is the format's native shape for multi-tool results.
   - Repo-level rules (`repo.*`) run once, not once per child; the
     dispatcher dedups identical findings from multiple checkers.

   Done when: one command checks a mixed Go/TypeScript/Rust/Python fixture
   end to end, a missing-checker case produces the new finding, and
   `specs/PRODUCT.md` documents the dispatcher as the default entry point
   for polyglot repositories.

## Out of Scope

- Verifying actual CI run history through the GitHub API. The checkers stay
  static and offline; binding-evidence analysis (task 1) is the static
  approximation.
- Script reachability analysis (only crediting scripts that a binding
  workflow invokes). Recorded as a possible version 2 tightening under
  task 1.
- Auto-installing checkers from the dispatcher.
- Trend dashboards or history beyond the shrink-only baseline.

## Sequencing Notes

- Tasks 1–3 are the priority: they close the gap between looking guarded
  and being guarded, and every later adopter benefits. Task 1 lands first;
  task 2 builds on its binding-evidence test.
- Task 2 and task 3's report change both regenerate expected fixture
  reports; batching them into one conformance update avoids churn.
- Task 4 unlocks the existing population of unclean repositories and should
  land before any adoption push. Task 5 is small and rides along with any
  release.
- Tasks 6 and 7 are large; 7 gains most of its value from 6 and follows it.
  Task 6 gets its own dated implementation plan when scheduled.
