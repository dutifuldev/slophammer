# Report Format

Slophammer currently supports text and JSON reports.

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
