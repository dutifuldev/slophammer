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
  coverage_threshold: 85
  dry:
    max_findings: 0
    paths:
      - go/cmd
      - go/internal
    exclude:
      - "**/*_test.go"
      - "fixtures/**"
      - "templates/**"
    structural:
      enabled: true
      threshold: 0.82
      min_lines: 4
      min_nodes: 20
    copied_blocks:
      enabled: true
      min_tokens: 100
  crap_max_score: 8
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
typescript:
  coverage_threshold: 85
  complexity_max: 8
  dry:
    max_findings: 0
    paths:
      - typescript/src
    exclude:
      - "**/*.test.ts"
      - "**/*.spec.ts"
      - "typescript/fixtures/**"
      - "typescript/dist/**"
      - "typescript/coverage/**"
    copied_blocks:
      enabled: true
      min_tokens: 100
  mutation_targets:
    - typescript/src/rules/rules.ts
  dependency_boundaries:
    - from: typescript/src/app
      allow:
        - typescript/src/config
        - typescript/src/dry
        - typescript/src/repo
        - typescript/src/report
        - typescript/src/rules
        - typescript/src/scan
        - typescript/src/toolchecks
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

`go.coverage_threshold`, `go.dry_max_candidates`, `go.dry`,
`go.crap_max_score`, and `go.mutation_targets` are parsed as typed policy
fields.

The Go policy values have hard recommended bounds. Slophammer rejects config
that weakens them:

- `go.coverage_threshold` must be at least `85`.
- `go.crap_max_score` must be at most `8`.

Projects may choose stricter values, such as higher coverage or lower CRAP
limits, but they cannot configure weaker values through `slophammer.yml`.

`go.dry_paths` and `go.dry_exclude` configure production-only DRY enforcement.
The nested `go.dry` shape is the preferred spelling for new repos:

- `go.dry.max_findings` sets the finding budget.
- `go.dry.paths` selects paths to scan.
- `go.dry.exclude` excludes tests, fixtures, templates, or generated code.
- `go.dry.structural` configures function and method similarity.
- `go.dry.copied_blocks` configures CPD-style copied-block detection.

Slophammer expands the configured paths to Go source files before running its
native DRY engine. The old top-level fields remain accepted for compatibility.

The direct Go commands use these values as defaults:

- `slophammer-go dry` uses `go.dry_max_candidates` unless
  `--max-candidates` is passed.
- `slophammer-go crap` uses `go.crap_max_score` unless `--max-score` is
  passed.
- `slophammer-go mutate` uses `go.mutation_targets` unless `--target` is
  passed.

`slophammer-go check --execute` runs configured Go tool checks and adds failures
to the normal report. Go tool execution runs from discovered Go module roots,
so a repo-level config can drive a nested module such as `go/`. Embedded
`fixtures/`, `templates/`, and `vendor/` modules are not execution targets.

The configured DRY budget is zero for production code. Tests are reviewed
selectively, fixtures are excluded, and templates are checked as independent
reference projects.

`go.dependency_boundaries` is active now. Each boundary declares:

| Field   | Meaning                                                   |
| ------- | --------------------------------------------------------- |
| `from`  | Repository-root or module-root-relative package path.     |
| `allow` | Local package paths that code under `from` may import.    |

External imports are ignored. Local imports are resolved through the nearest
`go.mod` module path.

## TypeScript Config

`typescript.coverage_threshold`, `typescript.complexity_max`,
`typescript.dry`, `typescript.mutation_targets`, and
`typescript.dependency_boundaries` are parsed as typed policy fields.

The TypeScript policy values have hard recommended bounds. Slophammer rejects
config that weakens them:

- `typescript.coverage_threshold` must be at least `85`.
- `typescript.complexity_max` must be at most `8`.

Projects may choose stricter values, such as higher coverage or lower
complexity limits, but they cannot configure weaker values through
`slophammer.yml`.

The nested `typescript.dry` shape configures native copied-block duplicate
detection:

- `typescript.dry.max_findings` sets the finding budget.
- `typescript.dry.paths` selects paths to scan.
- `typescript.dry.exclude` excludes tests, fixtures, build output, or generated
  code.
- `typescript.dry.copied_blocks` configures CPD-style copied-block detection.

The configured DRY budget is zero for production code.

`typescript.dependency_boundaries` declares import boundaries:

| Field   | Meaning                                                   |
| ------- | --------------------------------------------------------- |
| `from`  | Repository-root-relative source path.                     |
| `allow` | Local source paths that code under `from` may import.     |

External package imports are ignored. Relative imports are resolved against the
importing source file.
