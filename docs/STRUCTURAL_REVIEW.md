# Structural Review

Structural review is the habit of judging code by its shape before reading every
line.

When agents produce code faster than humans can read it, review has to move up a
level. The human still manages quality, but the first pass is about structure,
not syntax.

## Core Idea

Do not try to keep up with agents by reading every character.

Instead, inspect the signals that reveal whether the code is under control:

- module boundaries
- directory names
- function names
- function size
- argument counts
- dependency direction
- branching depth
- public API shape
- test shape

This does not mean ignoring code. It means choosing the right level of attention
first.

## What To Look For

Healthy structure usually has:

- small modules with clear jobs
- boring dependency direction
- business rules away from framework plumbing
- names that reveal intent
- tests that name behavior, not implementation trivia
- few large switches or deeply nested branches
- minimal broad dynamic types

Weak structure often shows:

- utility files that collect unrelated behavior
- handlers that contain business rules
- duplicated branches with small differences
- functions with too many arguments
- tests that only assert that something exists
- dependencies pointing both ways between packages

## Review Loop

A practical review loop:

1. Scan the changed file list.
2. Check whether changes landed in the right layer.
3. Look at function and module sizes.
4. Check names and public APIs.
5. Inspect tests before implementation details.
6. Read the riskiest code paths deeply.
7. Ask for focused refactors, not broad rewrites.

This keeps the human in control without turning review into line-by-line
transcription.

## Automated Support

Structural review gets stronger when backed by tools:

- complexity linting
- dependency checks
- duplicate detection
- coverage gates
- CRAP analysis
- mutation testing
- architecture tests

The human sees the shape. The tools make the shape measurable.

## Source Notes

- [Morning Bathrobe Rant: Disengage From the Syntax](uncle-bob/2026-05-06-disengage-from-the-syntax.md)
- [Morning Bathrobe Rant: AI Slop](uncle-bob/2026-04-26-ai-slop.md)
- [Uncle Bob GitHub Projects: Agentic Coding Guardrails](uncle-bob/2026-05-12-uncle-bob-github-projects.md)
