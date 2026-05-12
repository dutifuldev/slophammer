# Rules

This file defines the shared rule contract. Every implementation should use the
same rule IDs, severities, finding paths, messages, and descriptions for shared
rules.

## Shared Repository Rules

| Rule ID                | Severity | Finding path        | Finding message                                                      |
| ---------------------- | -------- | ------------------- | -------------------------------------------------------------------- |
| `repo.readme-required` | `error`  | `README.md`         | `README.md is required`                                              |
| `repo.agents-required` | `error`  | `AGENTS.md`         | `AGENTS.md is required`                                              |
| `repo.ci-required`     | `error`  | `.github/workflows` | `.github/workflows must contain at least one .yml or .yaml workflow` |

## Rule Descriptions

### `repo.readme-required`

The target repo should have a `README.md`.

The filename comparison is case-insensitive.

### `repo.agents-required`

The target repo should have an `AGENTS.md`.

The filename comparison is case-insensitive.

### `repo.ci-required`

The target repo should have a CI workflow under `.github/workflows`.

Any regular file directly under `.github/workflows` with a `.yml` or `.yaml`
extension satisfies the rule.

## Finding Order

Reports sort findings by `rule_id`, then by `path`.
