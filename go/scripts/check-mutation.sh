#!/usr/bin/env bash

set -euo pipefail

go run github.com/unclebob/mutate4go/cmd/mutate4go@latest internal/rules/rules.go --scan

