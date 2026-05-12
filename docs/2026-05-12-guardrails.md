---
title: Guardrails
author: Bob <dutifulbob@gmail.com>
date: 2026-05-12
---

# Guardrails

Use this checklist when evaluating a generated or agent-assisted project.

## Required Project Checks

- Formatting is deterministic and automated.
- Linting fails on unsafe patterns instead of only enforcing style.
- Type checking runs in strict mode where the language supports it.
- Unit tests cover core policy without requiring external services.
- Integration tests exist for IO boundaries that matter.
- CI runs the same checks developers run locally.
- Dependency additions are intentional and reviewed.
- Secrets are loaded from environment or secret managers, never committed.

## Architecture Checks

- Domain logic can be tested without starting servers or databases.
- External input is validated at the boundary.
- Error handling is explicit and observable.
- Time, randomness, file systems, networks, and process state are injectable where behavior depends on them.
- Framework-specific code does not leak into core policy.

## Review Questions

- Could a maintainer explain the module boundaries without reading every file?
- Does the code fail closed when input is malformed?
- Are unsafe type escapes rare, named, and justified?
- Can the most important tests run in under a few seconds?
- Would removing the AI prompt history make this project hard to maintain?
