---
title: Config Dialect Decision
author: Onur Solmaz <onur.solmaz@huggingface.co>
date: 2026-06-10
status: completed
---

# Config Dialect Decision

`slophammer.yml` uses two key shapes for the same concepts. Go and TypeScript
use flat keys:

```yaml
go:
  coverage_threshold: 85
  crap_max_score: 8
typescript:
  coverage_threshold: 85
  complexity_max: 8
```

Rust uses nested keys:

```yaml
rust:
  coverage:
    threshold: 85
  complexity:
    cognitive_max: 8
  mutation:
    targets:
      - rust/crates/slophammer-cli/src/app.rs
```

Both shapes are documented in [CONFIG.md](../specs/CONFIG.md) and validated
strictly by their implementations, so the divergence is consistent today. It
is still a cost: agents copying config between languages guess wrong, and the
planned umbrella `slophammer` package would have to speak both dialects.

## Decision

The nested shape is the target. The flat Go and TypeScript keys are a
compatibility dialect, not a second standard.

- New config sections in any implementation use the nested shape.
- The Go and TypeScript checkers keep reading the flat keys indefinitely;
  strict validation already rejects unknown keys, so both dialects stay
  unambiguous.
- Migrating Go and TypeScript to also accept nested keys is deferred until
  umbrella-package work starts. Doing it earlier churns every adopter's
  `slophammer.yml` for no immediate benefit.
- When the migration lands, flat keys remain readable for at least one minor
  release line, and `explain`-style documentation points at the nested
  replacements.

## Why not declare the divergence permanent

A permanent split makes the umbrella package's config translation layer a
forever cost and keeps cross-language copying error-prone. The nested shape
won because it groups related settings (`coverage.threshold`,
`coverage.paths`, `coverage.exclude`) instead of multiplying prefixed flat
keys, and because Rust — the newest implementation — already proves it out.

## Why not migrate now

The flat keys are released behavior in two shipped checkers. Breaking or
duplicating them now would force every adopter to touch config without any
new capability. The decision point that actually requires one dialect is the
umbrella package, so the work is scheduled there.
