# Unit Tests

Unit tests are the first local defense against agent damage.

Agents can edit code quickly. They can also break code quickly. A fast unit test
suite gives the agent immediate feedback when a small change violates existing
behavior.

## Core Idea

If agents are writing or changing implementation code, they must also write and
run unit tests.

The test suite should be fast enough that the agent can run it repeatedly during
normal work. If tests are slow, flaky, or hard to run, they will stop acting as a
guardrail.

## What Unit Tests Should Cover

Unit tests should focus on:

- domain rules
- branching logic
- validation
- parsing
- formatting
- error handling
- edge cases
- regression cases

They should not spend most of their effort mocking internal implementation
details. Mock real boundaries: network, filesystem, clocks, queues, databases,
and external services.

## Test-Driven Pressure

The ideal workflow is test-first:

1. Write a failing test for the behavior.
2. Write the smallest implementation that passes.
3. Refactor with the test still green.

Agents may not follow this perfectly just because a rule file says so. That is
why CI, review, and coverage checks still matter.

## Practical Agent Rules

Good instructions for agents are concrete:

```text
Add tests for new behavior.
Add a regression test for bug fixes.
Run the focused test package first.
Run the full default test command before finishing.
Do not weaken or delete a failing test to make progress.
```

The last rule is important. A failing test is information, not an obstacle to
hide.

## Relationship To Coverage

Unit tests and coverage are related, but they are not the same thing.

Coverage can show that code ran. It cannot prove that the test made a meaningful
assertion. Use coverage to find blind spots, then use review and mutation
testing to improve the strength of the tests.

## Source Notes

- [Morning Bathrobe Rant: Unit Testing](uncle-bob/2026-04-22-unit-testing.md)
- [Mutation Testing](MUTATION_TESTING.md)
- [Coverage And CRAP](COVERAGE_AND_CRAP.md)
