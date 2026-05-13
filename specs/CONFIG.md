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
  dry_max_candidates: 50
  dry_paths:
    - go/cmd
    - go/internal
  dry_exclude:
    - "**/*_test.go"
    - "fixtures/**"
    - "templates/**"
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
`go.mutation_targets` are parsed as typed policy fields.

`go.dry_paths` and `go.dry_exclude` are the intended policy shape for
production-only DRY enforcement. They document where Slophammer should run
`dry4go` once path filtering is implemented. Until then, the Go implementation
uses the target module root and the candidate budget.

The direct Go commands use these values as defaults:

- `slophammer go dry` uses `go.dry_max_candidates` unless
  `--max-candidates` is passed.
- `slophammer go crap` uses `go.crap_max_score` unless `--max-score` is
  passed.
- `slophammer go mutate` uses `go.mutation_targets` unless `--target` is
  passed.

`slophammer check --execute` runs configured Go tool checks and adds failures
to the normal report. Go tool execution runs from discovered Go module roots,
so a repo-level config can drive a nested module such as `go/`. Embedded
`fixtures/`, `templates/`, and `vendor/` modules are not execution targets.

The long-term DRY budget should be zero for production code. Tests are reviewed
selectively, fixtures are excluded, and templates are checked as independent
reference projects.

`go.dependency_boundaries` is active now. Each boundary declares:

| Field   | Meaning                                                   |
| ------- | --------------------------------------------------------- |
| `from`  | Repository-root or module-root-relative package path.     |
| `allow` | Local package paths that code under `from` may import.    |

External imports are ignored. Local imports are resolved through the nearest
`go.mod` module path.
