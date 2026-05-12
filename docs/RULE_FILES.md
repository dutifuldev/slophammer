# Rule Files

Rule files are project instructions for coding agents. Common names include
`AGENTS.md`, `CLAUDE.md`, and similar tool-specific files.

The useful version of a rule file is short, direct, and operational. It tells
the agent what to do next, which constraints matter, and which commands define
done.

## Core Idea

Agents do not become more reliable because a file sounds stern.

Long moral instructions waste context and are easy to ignore. Short rules with
clear commands have a better chance of surviving the work loop.

Good rule files are closer to checklists than essays.

## Good Rules

Use rules that are:

- specific
- short
- observable
- tied to repository commands
- stable across tasks

Examples:

```text
Run go test ./... before finishing.
Do not introduce any in TypeScript source files.
Keep domain logic out of HTTP handlers.
Add a regression test for bug fixes.
```

These rules can be checked by a human, a script, or CI.

## Weak Rules

Avoid rules that only express attitude:

```text
Always write good code.
Do not be lazy.
Follow all best practices.
Remember everything above.
```

Those lines consume tokens without adding an enforceable constraint.

## How To Write A Project Rule File

Start with the smallest useful set:

1. State the project boundary.
2. Name the important commands.
3. List the forbidden shortcuts.
4. Define the test expectation.
5. Define the review expectation.

Then stop.

If a rule can move into CI, lint, tests, type checks, or a script, move it
there. Rule files should tell the agent about constraints that exist elsewhere,
not replace those constraints.

## Source Notes

- [Morning Bathrobe Rant: Rule Files](uncle-bob/2026-04-18-rule-files.md)
