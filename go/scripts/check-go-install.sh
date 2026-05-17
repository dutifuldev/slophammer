#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOBIN_DIR="$(mktemp -d)"
trap 'rm -rf "$GOBIN_DIR"' EXIT

cd "$ROOT"

GOBIN="$GOBIN_DIR" go install ./cmd/slophammer-go

"$GOBIN_DIR/slophammer-go" help >/tmp/slophammer-go-help.txt
grep -q "slophammer-go check" /tmp/slophammer-go-help.txt

"$GOBIN_DIR/slophammer-go" rules >/tmp/slophammer-go-rules.txt
grep -q "go.dry-required" /tmp/slophammer-go-rules.txt

"$GOBIN_DIR/slophammer-go" check ../fixtures/repos/clean
"$GOBIN_DIR/slophammer-go" dry ..
"$GOBIN_DIR/slophammer-go" crap ..
"$GOBIN_DIR/slophammer-go" mutate .. --scan
