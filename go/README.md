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
./scripts/check-dry.sh
./scripts/check-crap.sh
./scripts/check-mutation.sh
go build ./cmd/slophammer
go run ./cmd/slophammer check ..
```
