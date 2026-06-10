# Local development gate. CI (.github/workflows/ci.yml) additionally runs
# Rust coverage, dependency audit, mutation listing, packaging dry runs, the
# Python template gate, and SARIF upload.

.PHONY: check check-docs check-go check-typescript check-rust check-templates conformance

check: check-docs check-go check-typescript check-rust check-templates conformance

check-docs:
	node scripts/check-doc-links.mjs
	npx -y @simpledoc/simpledoc check

check-go:
	cd go && test -z "$$(gofmt -l .)"
	cd go && golangci-lint run ./...
	cd go && go vet ./...
	cd go && go test ./...
	cd go && ./scripts/check-go-coverage.sh
	cd go && go build ./cmd/slophammer ./cmd/slophammer-go
	cd go && go run ./cmd/slophammer-go dry ..
	cd go && go run ./cmd/slophammer-go crap ..
	cd go && go run ./cmd/slophammer-go mutate .. --scan
	cd go && go run ./cmd/slophammer-go check ..
	cd go && go run ./cmd/slophammer-go check .. --execute

check-typescript:
	cd typescript && npm run check

check-rust:
	cd rust && cargo fmt --check
	cd rust && cargo check --workspace
	cd rust && cargo clippy --workspace --all-targets -- -D warnings
	cd rust && cargo test --workspace --all-targets
	cd rust && cargo run -q -p slophammer-rs -- check ..

check-templates:
	cd templates/go && go test ./... && go vet ./...
	cd templates/typescript && npm run check

conformance:
	node scripts/check-conformance.mjs
