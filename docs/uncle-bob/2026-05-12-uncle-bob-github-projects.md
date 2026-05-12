---
title: "Uncle Bob GitHub Projects: Agentic Coding Guardrails"
author: Bob <dutifulbob@gmail.com>
date: 2026-05-12
---

# Uncle Bob GitHub Projects: Agentic Coding Guardrails

Uncle Bob's GitHub profile is [github.com/unclebob](https://github.com/unclebob).

As of 2026-05-12, the profile identifies him as Robert C. Martin, links to
`cleancoder.com`, and has a recent cluster of public repositories around AI
agent coordination, CRAP analysis, mutation testing, duplication detection,
dependency checks, and acceptance-test-driven generation.

This matters for `slophammer` because the repos form a practical tool
stack for the same problem described in the 2026 bathrobe rants: AI agents can
produce code quickly, but they need external constraints that measure structure,
coverage, mutation strength, duplication, dependency direction, and acceptance
behavior.

## Recent Activity Snapshot

GitHub public activity and repository metadata show this recent pattern:

| Date       | Activity                                                                                             |
| ---------- | ---------------------------------------------------------------------------------------------------- |
| 2026-05-11 | Created/pushed Go guardrail tools: `crap4go`, `mutate4go`, `dry4go`; also created/pushed `dry4java`. |
| 2026-05-07 | Pushed Clojure guardrail tools: `dry4clj` and `crap4clj`.                                            |
| 2026-04-17 | Created `swarm-forge`, an AI-agent coordination repo.                                                |
| 2026-03    | Pushed Java/Clojure guardrail tools: `crap4java`, `mutate4java`, `clj-mutate`, `scrap`, `AIR-J`.     |
| 2026-02    | Pushed Claude-generated `Pharaoh` rewrites using extracted specs/features.                           |

## Project Map

| Area                       | Repos                                                                                     | Guardrail role                                                         |
| -------------------------- | ----------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| Agent coordination         | [`swarm-forge`](https://github.com/unclebob/swarm-forge)                                  | Coordinate multiple agents without losing role, worktree, or task flow |
| AI-first representation    | [`AIR-J`](https://github.com/unclebob/AIR-J)                                              | Reduce ambiguity and representation drift for agent-written programs   |
| CRAP analysis              | `crap4clj`, `crap4java`, `crap4go`                                                        | Combine complexity and coverage to expose risky code                   |
| Mutation testing           | `clj-mutate`, `mutate4java`, `mutate4go`                                                  | Prove tests detect behavior changes, not just execute lines            |
| Duplication detection      | `dry4clj`, `dry4java`, `dry4go`                                                           | Find structurally similar code for refactoring                         |
| Test/spec quality          | `scrap`, `speclj-structure-check`                                                         | Keep generated tests/specs structurally meaningful                     |
| Dependency/architecture    | `dependency-checker`, `arch-view`                                                         | Keep module boundaries and dependency direction visible                |
| Acceptance-test generation | `Pharaoh`, `Pharaoh-js`, [`fitnesse`](https://github.com/unclebob/fitnesse) as background | Anchor generated systems in external behavioral facts                  |

## Agent Coordination and AI-Oriented Code

### swarm-forge

[`swarm-forge`](https://github.com/unclebob/swarm-forge) is described as "a
simple tool for coordinating several AI agents." Its README frames it as a
tmux-based agent orchestration platform for turning multiple agents into
reliable software engineers.

Important design points:

- It coordinates agents working in different git worktrees.
- It uses project-local configuration under `swarmforge/`.
- It assigns roles through per-role prompts.
- It creates tmux sessions and Terminal windows per configured role.
- It provides message-passing and cleanup scripts.

For `slophammer`, this repo is the operational side of the problem:
multiple agents need explicit boundaries, ownership, and communication paths.
The tool is less about code quality metrics and more about making a swarm
manageable enough that quality checks can be applied coherently.

### AIR-J

[`AIR-J`](https://github.com/unclebob/AIR-J) is described as "a simple language
for AIs to use and humans to ignore." The README calls it an AI-first,
JVM-targeting language whose primary artifact is a canonical, typed,
effect-tracked intermediate representation.

Key ideas:

- "One meaning, one representation."
- Explicit imports, exports, types, effects, contracts, and invariants.
- JVM lowering and bytecode emission.
- Canonical standard modules for boundaries such as files, JSON, environment,
  subprocesses, tests, and test runners.
- A goal of reducing ambiguity, representation variance, and re-analysis cost.

This is a deeper version of the guardrails thesis. Instead of only linting a
human language after generation, AIR-J tries to make the representation itself
more machine-checkable and less ambiguous for agents.

## CRAP Tools

CRAP means **Change Risk Anti-Pattern** in these repos. The shared idea is:

```text
CRAP(fn) = CC^2 * (1 - coverage)^3 + CC
```

Where:

- `CC` is cyclomatic complexity.
- `coverage` is the test coverage fraction for the function or method.

The useful agentic insight is that CRAP creates a single pressure metric that
forces two behaviors at once:

- Split complex functions.
- Add tests where complexity remains.

### crap4go

[`crap4go`](https://github.com/unclebob/crap4go) is the Go implementation. It
was created/pushed on 2026-05-11.

What it does:

- Runs `go test ./... -coverprofile=target/coverage/coverage.out`.
- Reads Go coverage profiles.
- Computes function/method cyclomatic complexity.
- Maps coverage to function source ranges.
- Reports CRAP scores sorted by risk.
- Skips `_test.go`, `target/`, `vendor/`, and `.git/`.
- Includes a `SKILL.md` so coding agents can learn how to run it.

The README's risk bands are straightforward:

| Score | Meaning                                  |
| ----- | ---------------------------------------- |
| 1-5   | Low risk                                 |
| 5-30  | Moderate risk; refactor or add tests     |
| 30+   | High risk; complex and under-tested code |

This is the most directly reusable project for Go templates in this repo.

### crap4java

[`crap4java`](https://github.com/unclebob/crap4java) is a Java CRAP analyzer. It
uses Maven and JaCoCo:

- Deletes stale JaCoCo artifacts.
- Runs JaCoCo coverage.
- Reads `target/site/jacoco/jacoco.xml`.
- Computes method-level CRAP scores.
- Exits with code `2` when the CRAP threshold is exceeded.

Its default threshold is documented as `> 8.0`, which makes it suitable as a CI
quality gate.

### crap4clj

[`crap4clj`](https://github.com/unclebob/crap4clj) is the Clojure version. It
uses Cloverage and is built around Clojure projects using Speclj or
`clojure.test`.

Notable details:

- It runs Cloverage before analysis.
- It reads LCOV-style coverage output.
- It computes CRAP for Clojure functions.
- Its repo description explicitly says it is easy to adjust to other testers.

This appears to be the earlier version of the CRAP tool family that later gets
ported into Java and Go.

## Mutation Testing Tools

The mutation tools are the second half of the testing story. Coverage says code
was executed. Mutation testing asks whether the test suite detects meaningful
behavior changes.

### mutate4go

[`mutate4go`](https://github.com/unclebob/mutate4go) was created/pushed on
2026-05-11. It targets one Go source file at a time.

Main behavior:

- Runs coverage with `go test ./... -coverprofile=target/coverage/coverage.out`.
- Runs a baseline test command.
- Discovers mutation sites.
- Skips uncovered mutation sites.
- Applies each mutation.
- Runs tests with timeouts.
- Restores the original source file.
- Reports killed, survived, and uncovered mutations.
- Writes an embedded footer manifest with test date and function hashes.
- Defaults to differential mutation when a footer manifest exists.

Supported mutation categories include arithmetic, comparison, equality,
boolean, logical, and constant changes.

For agentic coding, this is a direct way to tell an agent: "Do not merely raise
coverage; kill the surviving mutants."

### mutate4java

[`mutate4java`](https://github.com/unclebob/mutate4java) provides the same
basic pattern for Java:

- One Java source file target at a time.
- Maven baseline tests.
- JaCoCo coverage filtering.
- AST-based mutation sites.
- Differential mutation through an embedded manifest.
- Optional worker copies for parallel isolation.
- Line targeting, scan mode, manifest refresh, coverage reuse, and timeout
  controls.

It is more mature than the Go version in CLI surface area and parallel worker
support.

### clj-mutate

[`clj-mutate`](https://github.com/unclebob/clj-mutate) is the Clojure mutation
tester. It targets projects using Speclj and supports:

- Mutation-site discovery.
- Killed/survived reporting.
- Coverage-aware mutation.
- Changed-form/differential workflows using a footer manifest.
- Integration with `scrap` analysis.

This forms the Clojure ancestor of the Java and Go mutation tools.

## DRY / Duplication Detection Tools

The DRY tools are structural duplicate detectors. They do not simply compare
text. They parse code, normalize syntax, and compare structural fingerprints
using Jaccard similarity:

```text
score = shared fingerprints / all fingerprints seen in either candidate
```

The default threshold in the Java and Go READMEs is `0.82`.

### dry4go

[`dry4go`](https://github.com/unclebob/dry4go) compares Go functions and
methods. It normalizes identifiers, local names, selector names, and literal
values while preserving Go syntax shape.

It reports duplicate candidates by filename and line range and supports text or
JSON output.

### dry4java

[`dry4java`](https://github.com/unclebob/dry4java) parses Java with JavaParser.
It compares declarations such as classes, records, enums, methods,
constructors, fields, initializer blocks, lambdas, expressions, statements,
modifiers, and operators.

It supports text and EDN output.

### dry4clj

[`dry4clj`](https://github.com/unclebob/dry4clj) applies the same structural
fingerprint idea to top-level Clojure forms. It normalizes Clojure syntax nodes
and compares forms by Jaccard similarity.

For `slophammer`, the DRY family provides a measurable way to push
agents away from copy/paste growth and toward intentional abstraction.

## Spec, Test, and Architecture Guardrails

### scrap

[`scrap`](https://github.com/unclebob/scrap) is a structural quality analyzer
for Speclj specs. Its README says it is aimed at test code in the same way CRAP
is aimed at production code.

It measures:

- Structural complexity.
- Weak-spec smells.
- Pressure to extract duplicated test scaffolding.

The key agentic detail is that SCRAP provides recommendations, not hard
directives. An agent should use it as decision support and then confirm the
intent of the spec before refactoring.

### speclj-structure-check

[`speclj-structure-check`](https://github.com/unclebob/speclj-structure-check)
is a static checker for Speclj structure. It catches mistakes that Speclj might
silently ignore, such as:

- `(it)` nested inside `(it)`.
- `(describe)` nested inside `(describe)` or `(context)`.
- setup forms inside `(it)`.
- unclosed forms.

This is a useful pattern for agent-generated tests: check the test file
structure before trusting test results.

### dependency-checker

[`dependency-checker`](https://github.com/unclebob/dependency-checker) analyzes
Clojure namespace dependencies against an explicit config.

It supports:

- Initial config generation.
- Source path configuration.
- EDN output.
- Component boundary checking.

This maps directly to the "push details down/out" guardrail: agents should not
be allowed to introduce dependency direction violations silently.

### arch-view

[`arch-view`](https://github.com/unclebob/arch-view) visualizes Clojure
architecture as layered namespaces with dependency indicators.

It:

- Scans `ns` forms.
- Builds namespace dependency graphs.
- Marks cycles.
- Shows incoming/outgoing dependency indicators.
- Supports interactive drill-down and headless EDN export.

For agentic development, this is a higher-level observability tool: it lets a
human or agent inspect architectural drift without reading every file.

## Acceptance-Test and Generated-System Examples

### Pharaoh and Pharaoh-js

[`Pharaoh`](https://github.com/unclebob/Pharaoh) is a Clojure rewrite of an old
Mac game, guided by original source and a detailed specification.

[`Pharaoh-js`](https://github.com/unclebob/Pharaoh-js) is the JavaScript
version, described as generated from Gherkin by Claude.

These are less like reusable guardrail tools and more like worked examples of
agent-assisted reconstruction:

- Start from original source/specification.
- Capture behavior as features/contracts.
- Use the agent to generate an implementation.
- Keep behavior anchored by external acceptance facts.

### fitnesse

[`fitnesse`](https://github.com/unclebob/fitnesse) is the long-running FitNesse
acceptance-test wiki. It is not a new AI repo, but it is relevant background:
the 2026 rants repeatedly emphasize acceptance tests as a way to keep agents
from breaking existing behavior.

## What This Suggests For slophammer

The recent Uncle Bob repos imply a concrete guardrail stack:

1. **Acceptance tests first.**
   Keep external behavior facts outside the model context and run them after
   every meaningful change.

2. **Unit coverage is necessary but insufficient.**
   Use coverage to find unexecuted logic, then mutation testing to find missing
   assertions.

3. **Use CRAP as a steering metric.**
   Make agents drive CRAP down by reducing complexity and adding tests.

4. **Detect structural duplication.**
   Use AST/fingerprint-based DRY tools rather than raw text matching.

5. **Check test structure.**
   Generated tests can be malformed or weak even when the test runner exits
   green.

6. **Check dependency direction.**
   Agents need explicit boundaries and automated dependency checks.

7. **Visualize architecture.**
   Humans should be able to review structure without reading every generated
   line.

8. **Coordinate multi-agent work explicitly.**
   Worktrees, roles, prompts, and message flow need their own guardrails.

## Candidate Follow-Ups For This Repo

- Add language-template hooks for CRAP-style thresholds once comparable tools
  are available or implemented.
- Add mutation-testing slots to each template:
  - Go: `mutate4go`
  - Java: `mutate4java`
  - Clojure: `clj-mutate`
- Add duplication-check slots:
  - Go: `dry4go`
  - Java: `dry4java`
  - Clojure: `dry4clj`
- Add a dependency-boundary example for each language template.
- Add a "test quality" checklist that distinguishes coverage, mutation score,
  and acceptance behavior.
- Add an agent-facing rule that generated code is not done until CRAP,
  mutation, dependency, and acceptance checks are green or explicitly waived.

## Source Links

- GitHub profile: [unclebob](https://github.com/unclebob)
- Agent coordination: [swarm-forge](https://github.com/unclebob/swarm-forge)
- AI-first language: [AIR-J](https://github.com/unclebob/AIR-J)
- Go tools: [crap4go](https://github.com/unclebob/crap4go), [mutate4go](https://github.com/unclebob/mutate4go), [dry4go](https://github.com/unclebob/dry4go)
- Java tools: [crap4java](https://github.com/unclebob/crap4java), [mutate4java](https://github.com/unclebob/mutate4java), [dry4java](https://github.com/unclebob/dry4java)
- Clojure tools: [crap4clj](https://github.com/unclebob/crap4clj), [clj-mutate](https://github.com/unclebob/clj-mutate), [dry4clj](https://github.com/unclebob/dry4clj)
- Structure tools: [scrap](https://github.com/unclebob/scrap), [speclj-structure-check](https://github.com/unclebob/speclj-structure-check), [dependency-checker](https://github.com/unclebob/dependency-checker), [arch-view](https://github.com/unclebob/arch-view)
- Acceptance/generated examples: [Pharaoh](https://github.com/unclebob/Pharaoh), [Pharaoh-js](https://github.com/unclebob/Pharaoh-js), [fitnesse](https://github.com/unclebob/fitnesse)
