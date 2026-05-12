#!/usr/bin/env bash
set -euo pipefail

go run github.com/unclebob/crap4go/cmd/crap4go@latest
awk -v score="0" -v maximum="30" 'BEGIN { exit !(score + 0 <= maximum + 0) }'
