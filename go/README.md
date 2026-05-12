# Slophammer Go

Go implementation of the Slophammer repository quality checker.

## Commands

```sh
go run ./cmd/slophammer check ..
go run ./cmd/slophammer check .. --format json
go run ./cmd/slophammer explain repo.agents-required
```

## Local Checks

```sh
gofmt -w .
go vet ./...
go test ./...
./scripts/check-go-coverage.sh
go build ./cmd/slophammer
```
