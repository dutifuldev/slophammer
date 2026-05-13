# Report Format

Slophammer currently supports text, JSON, and SARIF reports.

The JSON report is the shared compatibility contract for fixtures and future
language implementations.

## JSON Shape

```json
{
  "ok": false,
  "findings": [
    {
      "rule_id": "repo.agents-required",
      "severity": "error",
      "path": "AGENTS.md",
      "message": "AGENTS.md is required"
    }
  ]
}
```

## Fields

`ok` is `true` when there are no findings and `false` when at least one finding
exists.

`findings` is always present. Clean reports use an empty array.

Each finding contains:

| Field      | Meaning                                       |
| ---------- | --------------------------------------------- |
| `rule_id`  | Stable public rule identifier.                |
| `severity` | Rule severity. Current shared value: `error`. |
| `path`     | Repository-relative path for the finding.     |
| `message`  | Human-readable finding summary.               |

Findings are sorted by `rule_id`, then by `path`.

## SARIF

SARIF output is selected with:

```sh
slophammer check <path> --format sarif
```

SARIF uses version `2.1.0`. Each Slophammer finding becomes one SARIF result
with:

- `ruleId` from `finding.rule_id`
- `level` mapped from severity: `error` to `error`, `warn` to `warning`
- `message.text` from `finding.message`
- `locations[0].physicalLocation.artifactLocation.uri` from `finding.path`

SARIF is an output adapter over the shared findings model. It is not a second
rule model.
