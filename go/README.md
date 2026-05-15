# Slophammer Go

Go implementation of the Slophammer repository quality checker.

## Commands

```sh
go run ./cmd/slophammer check ..
go run ./cmd/slophammer check .. --format json
go run ./cmd/slophammer check .. --format sarif
go run ./cmd/slophammer check .. --execute
go run ./cmd/slophammer explain repo.agents-required
go run ./cmd/slophammer go dry ..
go run ./cmd/slophammer go dry .. --show-report
go run ./cmd/slophammer go crap ..
go run ./cmd/slophammer go mutate .. --scan
```

## Local Checks

```sh
gofmt -w .
golangci-lint fmt --diff
go vet ./...
go test ./...
./scripts/check-go-coverage.sh
go run ./cmd/slophammer go dry ..
go run ./cmd/slophammer go crap ..
go run ./cmd/slophammer go mutate .. --scan
go build ./cmd/slophammer
go run ./cmd/slophammer check ..
go run ./cmd/slophammer check .. --execute
```

The direct `go dry`, `go crap`, and `go mutate` commands read
`slophammer.yml` from the target path and use its Go policy values as defaults
when the matching CLI flag is not provided. `check --execute` runs configured
Go tool checks and reports failures through the normal Slophammer report model.

`go dry` is native to Slophammer. It combines structural function similarity
with CPD-style copied-block detection under one `dry` report.

## Lint Policy

The Go implementation uses `golangci-lint` as the lint runner. The current
baseline covers unused code, unchecked errors, ineffective assignments,
complexity, cognitive complexity, security mistakes, error wrapping, nil
handling, exhaustive switches, HTTP cleanup, context use, unnecessary
conversions, whitespace, and `nolint` discipline.

`revive` enforces an 800-line production file limit. Test files are excluded
from that file-length rule because fixture-heavy tests can be long without
creating the same production maintenance risk.

See [Implementation Model](../docs/IMPLEMENTATION_MODEL.md) for the full Go
lint policy.
