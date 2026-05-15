# Duplication Detection

Duplication detection finds code that has the same shape, even when the text is
not identical.

This matters for agentic projects because agents can produce repeated solutions
quickly. The code may pass tests while still growing in a copy/paste direction.

See [DRY](DRY.md) for the refactoring policy behind duplicate reports.

See [Unified DRY Engine Plan](2026-05-15-unified-dry-engine-plan.md)
for the plan to combine structural function detection and copied-block
detection into one Slophammer module.

## Core Idea

Do not wait for duplication to become obvious by eye.

Use tools that compare structure:

- function bodies
- methods
- declarations
- expressions
- statements
- normalized syntax trees

The goal is not to remove every repeated line. The goal is to catch repeated
logic before it becomes a maintenance burden.

## Why Agents Create Duplication

Agents often optimize for the immediate task. They may:

- copy a nearby pattern
- fork a branch instead of extracting a shared rule
- create parallel helpers with different names
- preserve old code while adding a new path
- avoid touching shared code because it seems risky

That behavior can be useful for a first pass. It becomes expensive when nobody
comes back to consolidate the shape.

## What To Do With A Duplicate Report

Treat duplicate reports as review prompts:

1. Check whether the repeated code represents the same idea.
2. If yes, extract the shared rule or helper.
3. If no, keep the duplication and improve names so the distinction is clear.
4. Add tests around the shared behavior before refactoring.
5. Rerun the duplicate detector.

Do not blindly abstract every match. Some duplication is cheaper than a bad
abstraction.

## Useful Thresholds

Structural duplicate tools often use a similarity threshold. A high threshold
finds close matches. A lower threshold finds broader families of similar code
but creates more noise.

Start strict. Lower the threshold only when the team is ready to inspect more
candidates.

## Source Notes

- [Unified DRY Engine Plan](2026-05-15-unified-dry-engine-plan.md)
- [Uncle Bob GitHub Projects: Agentic Coding Guardrails](uncle-bob/2026-05-12-uncle-bob-github-projects.md)
- [Structural Review](STRUCTURAL_REVIEW.md)
