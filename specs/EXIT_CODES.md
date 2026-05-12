# Exit Codes

Slophammer commands use stable exit codes so agents and CI systems can treat the
tool as a simple gate.

| Code | Meaning                                            |
| ---- | -------------------------------------------------- |
| `0`  | The command succeeded and no findings were found.  |
| `1`  | The command succeeded and findings were found.     |
| `2`  | The command failed due to usage or runtime errors. |

`explain` returns `0` for known rule IDs and `2` for unknown rule IDs.
