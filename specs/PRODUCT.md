# Product

Slophammer is a repository quality checker for agent-assisted software projects.

Each language implementation should expose the same product behavior, even
though each implementation uses its language's normal project structure and
tooling.

## Commands

The public command surface is:

```sh
slophammer check <path>
slophammer check <path> --format json
slophammer explain <rule-id>
```

## Check

`check` scans the target repository, evaluates the default rule set, writes a
report, and exits with a stable exit code.

The default report format is text. The JSON report format is selected with
`--format json`.

## Explain

`explain` prints rule metadata for a known rule ID.

Unknown rule IDs are command errors.

## Implementation Boundary

Implementations should keep this dependency direction:

```text
CLI
-> app orchestration
-> scanner
-> typed repository snapshot
-> pure rule evaluation
-> reporter
```

Rules should not walk the filesystem, parse command-line arguments, or write
reports.
