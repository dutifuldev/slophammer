# Coverage And CRAP

Coverage and CRAP analysis work best together.

Coverage shows which code was executed by tests. CRAP combines coverage with
cyclomatic complexity so the project can find functions that are both hard to
understand and weakly tested.

## Coverage

Coverage is useful because it reveals untested regions.

It is limited because execution is not the same as verification. A test can run
a line without asserting the behavior that line controls.

Use coverage to answer:

```text
Which important code paths are not exercised?
```

Do not use coverage alone to answer:

```text
Is this behavior well tested?
```

## CRAP

CRAP stands for Change Risk Anti-Pattern.

The common formula is:

```text
CRAP = CC^2 * (1 - coverage)^3 + CC
```

Where:

- `CC` is cyclomatic complexity.
- `coverage` is the covered fraction of the function or method.

The exact threshold can vary by language and project, but the pressure is the
same: complex uncovered code is risky.

## Why It Works For Agents

CRAP gives an agent a measurable target:

```text
Drive CRAP below the project threshold.
```

That target tends to force two useful behaviors:

- add tests around uncovered behavior
- split large branching functions into smaller pieces

This is more useful than telling an agent to "make the code cleaner." The metric
turns cleanliness into a visible feedback loop.

## What Good Output Looks Like

After a CRAP-driven cleanup, expect to see:

- smaller functions
- fewer deep branches
- clearer names
- better covered edge cases
- less hidden risk in central modules

The goal is not to make every function tiny. The goal is to make complicated
behavior visible, named, and tested.

## CI Guidance

Useful project defaults:

- enforce a repo-wide coverage floor
- enforce stricter floors on risky packages
- track complexity with a linter
- add a CRAP gate once the language tooling is stable
- review high-scoring functions before broad refactors

Coverage and CRAP should guide attention. They should not become a game where
agents add weak tests only to move a number.

## Source Notes

- [Morning Bathrobe Rant: AI Out-Codes You; Deal With It](uncle-bob/2026-04-20-ai-out-codes-you.md)
- [Morning Bathrobe Rant: CRAP](uncle-bob/2026-04-24-crap.md)
- [Uncle Bob GitHub Projects: Agentic Coding Guardrails](uncle-bob/2026-05-12-uncle-bob-github-projects.md)
