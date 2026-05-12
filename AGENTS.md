# AGENTS.md

These instructions apply to this repository.

## Baseline Rules

- Keep generated code reviewable by humans.
- Prefer small, typed modules with fast unit tests.
- Do not add dependencies unless they remove real complexity.
- Do not hide uncertainty with weak types, broad casts, reflection, or dynamic shapes.
- Keep business rules independent from IO, frameworks, databases, queues, clocks, and network clients.
- Add or update the nearest tests when changing behavior.
- Run the relevant formatter, linter, type checker, and test command before finishing a change.

## TypeScript

- Use `strict: true`.
- Do not use explicit `any`.
- Do not weaken types with unsafe assertions to bypass compiler or lint failures.
- Validate unknown external input at the boundary, then convert it to typed domain data.

## Python

- Use type annotations for public functions and meaningful internal helpers.
- Do not use explicit `Any` unless the boundary is genuinely dynamic and localized.
- Keep Ruff and mypy clean.

## Go

- Keep package APIs small and explicit.
- Return errors with context.
- Do not use global mutable state for domain behavior.
- Keep interfaces near their consumers.

