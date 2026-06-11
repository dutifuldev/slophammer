cd go
go test ./...
go vet ./...
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.0 run
./scripts/check-go-coverage.sh
slophammer go dry . --max-candidates 40
slophammer go crap . --max-score 30
slophammer go mutate . --target main.go --scan
