#!/usr/bin/env bash
set -euo pipefail

go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan
