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
goal. This plan turns those findings into ordered tasks.

Two findings motivate the top of the list. First, the command-presence rules
credit declarations that cannot execute: a workflow with a never-matching
branch filter, `if: false` on the job, and `continue-on-error: true` on every
step still passes `check` with exit 0. Second, config scope can quietly carve
real code out of checking: pointing `dry.paths` at an innocuous directory
hides genuine duplication with nothing in the report saying the production
tree went unscanned. Both holes matter because the adversary in Slophammer's
threat model is an agent optimizing to make the check pass.

## Phase 1: Make Declarations Bind

1. [ ] Refuse to credit CI steps that cannot run or cannot fail.

   The `*-required` command rules accept any workflow text containing the
   right command. They should reject evidence that is neutralized:

   - steps or jobs with `continue-on-error: true`
   - jobs with a constant-false `if:` condition
   - workflows whose triggers cannot fire on default-branch pushes or pull
     requests (for example a `branches:` filter naming no real integration
     branch; a conservative first version can require `push`, `pull_request`,
     or a schedule with no impossible filter)
   - disabled workflow files where detectable

   Implement the same evidence test in all three checkers and cover it with
   shared fixtures (a `*-neutralized-ci` fixture per ecosystem that must
   fail).

   Done when: the crafted counterexample above produces findings in every
   implementation, and conformance fixtures lock the behavior in.

2. [ ] Add a rule that the checker itself must run in CI.

   A repository can carry `slophammer.yml` and `AGENTS.md` while no CI step
   ever executes a Slophammer checker. The config is then decoration, and
   nothing reports that. Add `repo.slophammer-ci-required` (exact ID open):
   when `slophammer.yml` is present, some CI workflow or script reachable
   from CI must invoke a Slophammer checker, under the same binding-evidence
   test as task 1.

   Done when: a fixture with config but no checker invocation fails, and a
   fixture wiring any of the released checkers passes.

3. [ ] Make scope carve-outs visible and justified.

   Config can restrict `paths` and `exclude` for DRY, coverage, and targets
   without limit, so findings can be hidden by retargeting scope rather than
   weakening thresholds. Two complementary fixes:

   - Require a `reason` on scope restrictions that exclude production code,
     following the existing `rust.unsafe.allow` shape (`path` plus non-empty
     `reason`). Plain test/fixture/build-output excludes stay free-form.
   - Report scope coverage: when a scoped check runs, state how many
     production source files were in scope versus present in the repository,
     so a near-empty scope is visible in the report instead of silent.

   Done when: the carve-out counterexample (duplication in `src/`, scope
   pointed at a trivial directory) either fails validation or produces a
   report that names the unscanned production files.

## Phase 2: Make Adoption Possible

4. [ ] Add a baseline ratchet mode for existing repositories.

   The thresholds are absolute, weakening is forbidden, and a repository that
   is not already clean has no incremental path in: surveyed non-adopted
   repositories fail with anywhere from one to over seventy findings.
   Successful gate tools solve this with baselines. Add
   `check --baseline <file>`:

   - `--baseline-write` records current findings to a checked-in file
   - subsequent checks fail only on findings not in the baseline
   - fixing a baselined finding removes it permanently; the baseline only
     shrinks (the checker rejects a baseline file that grew)
   - the report always states the baseline debt so it cannot be forgotten

   The hard thresholds stay non-negotiable for new findings; the baseline
   only grandfathers history. Spec the file format in `specs/` so all
   implementations share it, and cover it with adoption fixtures.

   Done when: a failing fixture adopts via baseline with exit 0, a new
   regression on top of the baseline exits 1, and a grown baseline exits 2.

5. [ ] Document version pinning as the supported consumption pattern.

   Slophammer makes breaking releases deliberately and ships no compatibility
   shims, but the README and Agent Entrypoint never say how to consume it
   safely. CI integrations that install `@latest` absorb breaking changes
   mid-pipeline. Add a short section to the README and Agent Entrypoint:
   pin an exact version in CI (env-overridable), upgrade deliberately, and
   expect strict config validation to fail loudly across breaking releases.

   Done when: both documents show pinned install commands for all released
   checkers and the paste-block prompt tells agents to pin.

## Phase 3: Cover the Code Agents Actually Write

6. [ ] Ship `slophammer-py`.

   Python remains the largest agent-coding ecosystem and the one planned
   implementation that does not exist; polyglot repositories with Python
   components currently have no gate for them. Implement the checker per
   [Product](../specs/PRODUCT.md): repo rules plus Python rules for project
   metadata, type checking (mypy or pyright), Ruff lint and format, tests,
   coverage, complexity, DRY, mutation declaration, and dependency
   boundaries — same config shape, report model, exit codes, and conformance
   fixtures as the other three.

   Done when: `slophammer-py check` passes shared conformance with a Python
   fixture set at parity with the other ecosystems, and the
   [templates/python](../templates/python) template passes its own check.

7. [ ] Build the `slophammer` umbrella dispatcher.

   Polyglot repositories need every per-language checker installed and wired
   by hand, and in practice one language gets gated while the rest go
   unchecked. The reserved `slophammer` package should detect the languages
   present, run the matching released checkers, and merge their reports into
   one finding list with one exit code. Detection failures are loud: a
   recognized ecosystem with no installed checker is itself a finding.

   Done when: one command checks a mixed Go/TypeScript/Rust/Python fixture
   end to end and reports per-ecosystem and merged results.

## Sequencing Notes

- Tasks 1–3 are the priority: they close the gap between looking guarded and
  being guarded, and every later adopter benefits.
- Task 4 unlocks the existing population of unclean repositories and should
  land before any adoption push.
- Task 5 is small and can ride along with any release.
- Tasks 6 and 7 are large; 7 gains value from 6 and should follow it.
