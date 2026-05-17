# Slophammer Go

Go implementation of the Slophammer repository quality checker. The
user-facing product name is `slophammer-go`.

The Go implementation is native-first. It can also carry selected TypeScript or
Python checks when those checks are covered by the shared specs and fixtures.

## Commands

Installed command:

```sh
slophammer-go check ..
slophammer-go check .. --format json
slophammer-go check .. --format sarif
slophammer-go check .. --execute
slophammer-go explain repo.agents-required
slophammer-go dry ..
slophammer-go dry .. --show-report
slophammer-go crap ..
slophammer-go mutate .. --scan
```

Source-tree development command:

```sh
go run ./cmd/slophammer-go check ..
go run ./cmd/slophammer-go check .. --format json
go run ./cmd/slophammer-go check .. --format sarif
go run ./cmd/slophammer-go check .. --execute
go run ./cmd/slophammer-go explain repo.agents-required
go run ./cmd/slophammer-go dry ..
go run ./cmd/slophammer-go dry .. --show-report
go run ./cmd/slophammer-go crap ..
go run ./cmd/slophammer-go mutate .. --scan
```

## Local Checks

```sh
gofmt -w .
golangci-lint fmt --diff
go vet ./...
go test ./...
./scripts/check-go-coverage.sh
go run ./cmd/slophammer-go dry ..
go run ./cmd/slophammer-go crap ..
go run ./cmd/slophammer-go mutate .. --scan
go build ./cmd/slophammer
go build ./cmd/slophammer-go
go run ./cmd/slophammer-go check ..
go run ./cmd/slophammer-go check .. --execute
```

The direct `go dry`, `go crap`, and `go mutate` commands read
`slophammer.yml` from the target path and use its Go policy values as defaults
when the matching CLI flag is not provided. `check --execute` runs configured
Go tool checks and reports failures through the normal Slophammer report model.

Public packaging exposes those as `slophammer-go dry`, `slophammer-go crap`,
and `slophammer-go mutate`. The nested `go ...` source commands remain
compatibility forms for the older local development shape.

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
