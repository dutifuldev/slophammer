# Acceptance Tests

Acceptance tests describe what the system must do from the outside. They are
the behavioral anchor for agent-written code.

When an agent adds one feature and breaks another, the project needs a source of
truth outside the conversation. Acceptance tests provide that source of truth.

## Core Idea

Do not rely on a long prompt to preserve product behavior.

Prompts drift. Context gets crowded. Earlier requirements become weaker as the
conversation grows. Executable acceptance tests keep the behavior fixed even
when the model forgets.

## What They Protect

Acceptance tests protect:

- user-visible workflows
- cross-feature behavior
- edge cases that must not regress
- domain rules that span several modules
- compatibility promises

They are not a replacement for unit tests. They answer a different question:

```text
Does the system still do what it promised?
```

## Useful Formats

Readable formats work well because humans and agents can both use them:

```gherkin
Given an existing account
When the user changes their email
Then the account requires verification again
```

The exact tool matters less than executability. Gherkin, FitNesse-style tables,
snapshot-backed API scenarios, and ordinary end-to-end tests can all work if
they are reliable and run automatically.

## Agent Workflow

A good agent workflow makes acceptance tests part of every change:

1. Add or update the acceptance scenario.
2. Generate or write executable tests from the scenario.
3. Implement the feature.
4. Run all acceptance tests, not only the new one.
5. Fix every regression before moving on.

This prevents the feature set from behaving like a moving target.

## Repository Guardrails

For this repo, templates should make acceptance behavior easy to add:

- keep scenario files close to tests
- run acceptance tests in CI
- keep scenarios small and named by behavior
- avoid slow, fragile acceptance suites as the only default check
- use unit tests for detailed branching behavior

## Source Notes

- [Morning Bathrobe Rant: Quicksilver](uncle-bob/2026-04-21-quicksilver.md)
- [Uncle Bob GitHub Projects: Agentic Coding Guardrails](uncle-bob/2026-05-12-uncle-bob-github-projects.md)
