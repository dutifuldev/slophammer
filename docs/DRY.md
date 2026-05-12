# DRY

DRY means one idea should have one home.

It does not mean every repeated token is wrong. It means duplicated knowledge
should be visible early enough that a human can decide whether to extract a
shared rule, keep the code separate, or name the difference more clearly.

## Core Idea

Agent-written code tends to repeat working shapes.

That is useful in the first pass because the agent can copy a nearby pattern and
make progress quickly. It becomes dangerous when repeated shapes become parallel
business rules. The code may still pass tests, but later changes now have to be
made in multiple places.

DRY enforcement should therefore ask a structural question:

```text
Do these two pieces of code express the same shape strongly enough
that a human should review whether they are the same idea?
```

## What The DRY Tools Enforce

Uncle Bob's `dry4clj`, `dry4java`, and `dry4go` implementations all treat
duplication as structural similarity, not text equality.

The common model is:

1. Parse source code into language-aware structure.
2. Pick comparison units.
3. Normalize away incidental details.
4. Fingerprint the normalized structure.
5. Compare fingerprint sets with Jaccard similarity.
6. Report candidates above a threshold.

The score is:

```text
score = shared fingerprints / all fingerprints seen in either candidate
```

A score of `1.0` means the normalized structures have the same fingerprint set.
Lower scores can still matter when most of the shape is shared.

## Normalization

The important implementation move is normalization.

The DRY tools intentionally ignore names and literal values that often differ in
copy/paste variants:

- function names
- local variable names
- selector or field names where the language model treats them as incidental
- constants and literal values

They keep structural information that changes the meaning or maintenance shape:

- function, method, form, declaration, or lambda shape
- parameter and result structure
- statement order
- branch and loop structure
- operators
- collection shape
- modifiers, annotations, primitive types, and switch entry kinds where relevant

This is why two functions can look different by eye and still be a duplicate
candidate. If only the names and constants changed, the maintenance burden is
probably shared.

## Comparison Units

The tools use language-specific comparison units:

- `dry4clj` compares top-level Clojure forms, excluding namespace forms.
- `dry4java` compares Java declarations and lambdas: classes, records, enums,
  annotations, methods, constructors, fields, initializer blocks, enum
  constants, and lambda expressions.
- `dry4go` compares Go functions and methods.

That boundary matters. DRY enforcement works best when it reports units a human
can refactor. A whole file is usually too large. A single expression is often too
small. Functions, declarations, and top-level forms are reviewable chunks.

## Thresholds

The implementations use these practical defaults:

- minimum similarity threshold: `0.82`
- minimum candidate size: `4` lines
- minimum normalized node count: `20`

Those defaults keep the tool from reporting tiny coincidences. Start strict.
Lower the threshold only when the team is ready to inspect more candidates.

For agentic work, a good default policy is:

1. Run DRY detection before accepting a large agent change.
2. Treat each report as a review prompt, not an automatic refactor order.
3. Prefer extraction when the duplicate candidates encode the same rule.
4. Keep both copies when they only share mechanics, but improve names so the
   distinction is visible.
5. Add or strengthen tests before consolidating behavior.
6. Rerun the detector after refactoring.

## What To Refactor

A duplicate report should push the reviewer to identify the duplicated
knowledge, not just the duplicated syntax.

Good extraction targets:

- repeated business rules
- repeated validation logic
- repeated branching tables
- repeated traversal or filtering pipelines
- repeated test setup that hides behavior intent
- repeated command or report formatting rules

Risky extraction targets:

- code that happens to have the same control flow but different domain meaning
- tests that should stay explicit for readability
- small helpers where abstraction would hide the actual rule
- temporary duplication during an active migration

The goal is not a lower duplication number at any cost. The goal is fewer places
where one rule can drift into several rules.

## How This Applies To Slophammer

`slophammer` is intentionally implemented across languages, so some duplication
is part of the product. The Go, TypeScript, and Python implementations should
share the same product contract while staying idiomatic in each ecosystem.

DRY enforcement should focus on duplication inside one implementation:

- repeated rule evaluation logic
- repeated report construction
- repeated CLI parsing branches
- repeated filesystem scanning code
- repeated test fixture setup that could become a shared helper

Do not force the language implementations into one artificial abstraction. The
cross-language repetition is documentation by example. The intra-language
copy/paste is the risk.

## Source Notes

- [Duplication Detection](DUPLICATION_DETECTION.md)
- [Structural Review](STRUCTURAL_REVIEW.md)
- [Uncle Bob GitHub Projects: Agentic Coding Guardrails](uncle-bob/2026-05-12-uncle-bob-github-projects.md)
- [`dry4clj`](https://github.com/unclebob/dry4clj)
- [`dry4java`](https://github.com/unclebob/dry4java)
- [`dry4go`](https://github.com/unclebob/dry4go)
