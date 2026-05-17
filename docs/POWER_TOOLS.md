# Power Tools

AI coding agents are power tools.

They amplify human effort, but amplification cuts both ways. A careful operator
can do more work with better feedback. A careless operator can create more
damage, faster.

## Core Idea

Use agents for mechanical production and keep human review focused on:

- choosing behavior
- setting constraints
- checking structure
- reviewing risk
- deciding what is acceptable

Review the generated change against those points before merging it.

## What Responsible Use Looks Like

Responsible agent use means:

- small tasks
- clear acceptance criteria
- fast checks
- reviewed diffs
- rollback discipline
- explicit ownership
- tested behavior
- visible quality metrics

It also means knowing when to stop the agent and inspect the situation yourself.

## What Reckless Use Looks Like

Reckless use looks like:

- asking for broad rewrites without tests
- accepting large diffs without structure review
- using prompts instead of executable checks
- letting agents delete or weaken tests
- shipping generated code because it looks plausible
- treating CI failures as chores instead of feedback

The danger is not that the tool is powerful. The danger is using the tool
without a workbench, guards, and a way to measure the result.

## Repository Guardrails

For an agent-ready repo, make the responsible path the easy path:

- default commands in `README.md`
- agent rules in `AGENTS.md`
- strict lint and type checks
- automated tests in CI
- coverage and complexity gates
- small templates that show the intended structure
- docs that explain the quality loop

## Source Notes

- [Power Tools](uncle-bob/2026-04-28-power-tools.md)
- [Agentic Code Quality](AGENTIC_CODE_QUALITY.md)
