# Dependency Checks

Dependency checks keep architecture from drifting while agents move quickly.

They answer a simple question:

```text
Is this code allowed to depend on that code?
```

## Core Idea

Architecture is dependency direction.

If domain logic starts importing framework code, HTTP handlers, database models,
or cloud SDKs directly, the project becomes harder to test and harder to change.
Agents can make that drift happen accidentally unless the repo checks it.

## What To Check

Useful dependency rules include:

- domain packages do not import delivery packages
- core logic does not import framework libraries
- CLI and HTTP layers call into application services, not the reverse
- infrastructure packages depend inward, not outward
- generated code stays in generated-code boundaries
- tests may use helpers that production code may not import

The exact rules depend on the project. The important part is that the rules are
written down and executable.

## How To Enforce It

Dependency rules can be enforced with:

- architecture tests
- import graph checks
- language-specific linters
- module boundary tests
- package visibility
- simple scripts in CI

Start with the most important forbidden imports. Add more rules when real drift
appears.

## Agent Workflow

For agents, dependency checks are valuable because they turn architecture into a
concrete failure:

```text
This import is not allowed.
Move the code to the correct layer.
Keep the core independent of the framework.
```

That is much more effective than asking the agent to remember a design
philosophy.

## Relationship To Structural Review

Structural review can spot dependency problems by eye. Dependency checks make
them repeatable.

Use both:

- humans decide whether the boundary still makes sense
- tools prevent accidental boundary crossings

## Source Notes

- [Morning Bathrobe Rant: AI Slop](uncle-bob/2026-04-26-ai-slop.md)
- [Uncle Bob GitHub Projects: Agentic Coding Guardrails](uncle-bob/2026-05-12-uncle-bob-github-projects.md)
- [Structural Review](STRUCTURAL_REVIEW.md)
