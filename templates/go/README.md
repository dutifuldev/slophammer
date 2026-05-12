# Go Template

Small Go baseline for services, CLIs, libraries, and agent-generated modules.

## Commands

```sh
go test ./...
go vet ./...
golangci-lint run
```

## Guardrails

- Keep domain packages independent from process and network details.
- Return errors with enough context for callers.
- Keep interfaces near the code that consumes them.
- Use `go test` and `go vet` as the minimum local checks.

