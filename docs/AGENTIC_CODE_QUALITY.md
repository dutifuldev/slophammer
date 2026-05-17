# Agentic Code Quality

Agentic code quality is the discipline of managing fast code generation with
external constraints.

The premise is simple: agents can write and change code faster than humans can
comfortably review it. That speed is only useful when the project has guardrails
that keep the output understandable, tested, and structurally sane.

## Core Idea

Do not ask the agent to be careful and stop there.

Build a workflow where care is enforced by tools:

- tests define behavior
- coverage exposes blind spots
- mutation testing exposes weak assertions
- complexity metrics expose risky functions
- dependency checks expose architectural drift
- duplicate detection exposes copy/paste growth
- CI blocks incomplete work

The agent can then spend its speed on fixing concrete failures.

## The Human Role

The human does not disappear.

The human sets:

- the behavior the system must preserve
- the quality thresholds
- the architecture boundaries
- the review standard
- the acceptance criteria

The agent performs more of the typing. The human manages the constraints and
decides whether the result is acceptable.

## Quality Loop

A useful loop for agentic work:

1. State the behavior.
2. Add or update acceptance tests.
3. Ask for unit tests with the implementation.
4. Run type checks, lint, and unit tests.
5. Raise coverage on meaningful gaps.
6. Drive down high-complexity, low-coverage code.
7. Run mutation testing on risky paths.
8. Check dependencies and duplication.
9. Review structure before reading syntax.

## Common Failure Modes

Agentic projects drift when:

- instructions are long but unenforced
- generated tests have weak assertions
- acceptance behavior lives only in prompts
- CI checks are slow or optional
- humans stop reviewing structure
- agents patch around failing tests instead of understanding them
- broad dynamic types become escape hatches
- duplicated code grows faster than abstraction

The fix is usually not a better speech to the agent. The fix is a tighter
feedback loop.

## Minimum Bar

For a serious project, the minimum bar should be:

- strict typing where the language supports it
- no broad dynamic escape hatches in core code
- default test command
- coverage gate
- complexity gate
- documented architecture boundary
- CI that runs the same checks locally documented for agents

Then add mutation, CRAP, dependency checking, and duplicate detection as the
project grows.

## Source Notes

- [Morning Bathrobe Rant: AI Out-Codes You; Deal With It](uncle-bob/2026-04-20-ai-out-codes-you.md)
- [Morning Bathrobe Rant: AI Slop](uncle-bob/2026-04-26-ai-slop.md)
- [Morning Bathrobe Rant: Disengage From the Syntax](uncle-bob/2026-05-06-disengage-from-the-syntax.md)
- [Uncle Bob GitHub Projects: Agentic Coding Guardrails](uncle-bob/2026-05-12-uncle-bob-github-projects.md)
