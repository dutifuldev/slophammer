# Slophammer Go

Go implementation of the Slophammer repository quality checker.

## Commands

```sh
go run ./cmd/slophammer check ..
go run ./cmd/slophammer check .. --format json
go run ./cmd/slophammer explain repo.agents-required
go run ./cmd/slophammer go dry . --max-candidates 40
go run ./cmd/slophammer go dry . --max-candidates 40 --show-report
go run ./cmd/slophammer go crap . --max-score 30
go run ./cmd/slophammer go mutate . --target internal/rules/rules.go --scan
```

## Local Checks

```sh
gofmt -w .
go vet ./...
go test ./...
./scripts/check-go-coverage.sh
go run ./cmd/slophammer go dry . --max-candidates 40
go run ./cmd/slophammer go crap . --max-score 30
go run ./cmd/slophammer go mutate . --target internal/rules/rules.go --scan
go build ./cmd/slophammer
go run ./cmd/slophammer check ..
```
