# Config

Slophammer loads optional project config from the first matching file:

- `slophammer.yml`
- `slophammer.yaml`

Config is parsed after repository scanning and before rule evaluation. Invalid
config fails the check with exit code `2`.

## Shape

```yaml
rules:
  go.crap-required:
    severity: warn
go:
  coverage_threshold: 80
  dry_max_candidates: 40
  crap_max_score: 30
  mutation_targets:
    - go/internal/rules/rules.go
  dependency_boundaries:
    - from: go/internal/rules
      allow:
        - go/internal/config
        - go/internal/gotools
        - go/internal/repo
    - from: go/internal/repo
      allow: []
```

## Rule Config

Rule config currently supports severity overrides:

```yaml
rules:
  repo.readme-required:
    severity: warn
```

Valid severities are:

- `error`
- `warn`

Rule disabling is reserved in the config shape, but disabling a rule requires a
reason and is not used by the current Go implementation to hide findings.

## Go Config

`go.coverage_threshold`, `go.dry_max_candidates`, `go.crap_max_score`, and
`go.mutation_targets` are parsed as typed policy fields. They are available to
future execution/config-driven checks.

`go.dependency_boundaries` is active now. Each boundary declares:

| Field   | Meaning                                                   |
| ------- | --------------------------------------------------------- |
| `from`  | Repository-root or module-root-relative package path.     |
| `allow` | Local package paths that code under `from` may import.    |

External imports are ignored. Local imports are resolved through the nearest
`go.mod` module path.
