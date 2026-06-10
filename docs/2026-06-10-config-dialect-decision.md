---
title: Config Dialect Decision
author: Onur Solmaz <onur.solmaz@huggingface.co>
date: 2026-06-10
status: completed
---

# Config Dialect Decision

`slophammer.yml` used two key shapes for the same concepts. Go and TypeScript
used flat keys (`coverage_threshold`, `complexity_max`, `crap_max_score`,
`mutation_targets`), while Rust used nested keys (`coverage.threshold`,
`complexity.cognitive_max`, `mutation.targets`). Both were documented and
strictly validated, but the split made cross-language config copying
error-prone and would have forced the planned umbrella `slophammer` package to
speak two dialects.

## Decision

The nested shape is the only shape. The flat keys were removed in a single
cutover across all three implementations:

- `go.coverage_threshold` and `go.coverage_profile` became
  `go.coverage.threshold` and `go.coverage.profile`.
- `go.crap_max_score` became `go.crap.max_score`.
- `go.dry_max_candidates`, `go.dry_paths`, and `go.dry_exclude` were removed
  in favor of the existing nested `go.dry` block.
- `typescript.coverage_threshold` became `typescript.coverage.threshold`.
- `typescript.complexity_max` became `typescript.complexity.max`.
- `typescript.mutation_targets` became `typescript.mutation.targets`.
- `rust.coverage_threshold` was removed; `rust.coverage.threshold` was already
  the primary spelling.

Strict key validation rejects the removed flat keys in every implementation,
so a stale config fails loudly with exit code `2` instead of being silently
ignored.

The nested shape won because it groups related settings
(`coverage.threshold`, `coverage.paths`, `coverage.exclude`) instead of
multiplying prefixed flat keys, and because Rust — the newest implementation —
already proved it out. One dialect means one schema for the umbrella package
and no guesswork when copying config between language sections.
