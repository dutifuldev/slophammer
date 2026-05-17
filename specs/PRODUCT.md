# Product

Slophammer is a repository quality standard for agent-assisted software
projects.

Each language implementation exposes the same product behavior, even
though each implementation uses its language's normal project structure and
tooling. Installable implementations use short language-specific product
names:

| Implementation | Public command    |
| -------------- | ----------------- |
| Go             | `slophammer-go`   |
| TypeScript     | `slophammer-ts`   |
| Python         | `slophammer-py`   |

The product name identifies the implementation package. It does not forbid
cross-language checking. An implementation is native-first, and it can also
support other languages when those checks follow the shared Slophammer rules,
config, report, and exit-code contracts.

## Commands

The public command surface is the same for each implementation, using that
implementation's executable name:

```sh
slophammer-go check <path>
slophammer-go check <path> --format json
slophammer-go check <path> --format sarif
slophammer-go check <path> --execute
slophammer-go explain <rule-id>
slophammer-go rules
```

During the rename from the early `slophammer ...` command shape,
implementations keep compatibility aliases. Documentation and help text point
users to the language-specific public names.

## Check

`check` scans the target repository, evaluates the default rule set, writes a
report, and exits with a stable exit code.

The default report format is text. JSON and SARIF report formats are selected
with `--format json` and `--format sarif`.

`--execute` is opt-in. It runs configured tool checks and adds tool failures to
the same report model as static findings.

## Explain

`explain` prints rule metadata for a known rule ID.

Unknown rule IDs are command errors.

## Rules

`rules` prints the implemented rule catalog for the current implementation.
The catalog includes rule ID, category, severity, implementation status, and
the backing tool when one exists.

`rules` is an inspection command. It does not scan a repository and it exits
successfully when the catalog is written.

## Implementation Boundary

Implementations keep this dependency direction:

```text
CLI
-> app orchestration
-> scanner
-> typed repository snapshot
-> pure rule evaluation
-> reporter
```

Rules do not walk the filesystem, parse command-line arguments, or write
reports.
