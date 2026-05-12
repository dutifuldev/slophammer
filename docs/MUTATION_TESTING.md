# Mutation Testing

Mutation testing checks whether tests can detect wrong behavior.

Coverage can say that code was executed. Mutation testing asks a harder
question: if the implementation were changed in a small, meaningful way, would
the tests fail?

## Core Idea

A mutation tester changes source code on purpose, then runs the test suite.

Examples of mutations:

- `>` becomes `<`
- `==` becomes `!=`
- `&&` becomes `||`
- a constant changes value
- a return value becomes `nil`, `null`, `false`, or `0`

If the tests fail, the mutant was killed. If the tests still pass, the mutant
survived.

Surviving mutants are evidence that the test suite missed something.

## Why It Matters

High coverage can still hide weak tests.

A test may execute a function without checking the important result. Mutation
testing finds those gaps because it changes the behavior and expects the tests
to notice.

This is especially important for agent-written tests. Agents can produce tests
that look plausible while making shallow assertions.

## Agent Workflow

Use mutation results as concrete work for the agent:

1. Run the mutation tester on the changed package or file.
2. Review surviving mutants.
3. Ask the agent to add tests that kill specific survivors.
4. Rerun the mutation tester.
5. Keep the meaningful survivors at zero or document why a survivor is
   equivalent.

The agent should not change production code merely to avoid a mutant unless the
production code is genuinely unclear or wrong.

## Where To Start

Start mutation testing on:

- domain rules
- parsers
- validators
- pricing, billing, permissions, or security logic
- code with high CRAP scores
- code that agents recently changed

Do not begin by mutating the entire repo if the suite is large. Start with the
highest-risk files and expand from there.

## Useful Metrics

Track:

- killed mutants
- surviving mutants
- uncovered mutants
- timed-out mutants
- equivalent or ignored mutants

The most useful number is not raw mutation score. The most useful outcome is a
short list of behavior gaps that can be fixed.

## Source Notes

- [Morning Bathrobe Rant: Mutation Testing](uncle-bob/2026-05-05-mutation-testing.md)
- [Morning Bathrobe Rant: AI Out-Codes You; Deal With It](uncle-bob/2026-04-20-ai-out-codes-you.md)
- [Uncle Bob GitHub Projects: Agentic Coding Guardrails](uncle-bob/2026-05-12-uncle-bob-github-projects.md)
