#!/usr/bin/env bash

set -euo pipefail

maximum_crap_score="${MAXIMUM_CRAP_SCORE:-30}"
report="$(mktemp)"
trap 'rm -f "$report"' EXIT

go run github.com/unclebob/crap4go/cmd/crap4go@latest | tee "$report"

awk -v maximum="$maximum_crap_score" '
  NF >= 5 && $NF ~ /^[0-9]+(\.[0-9]+)?$/ {
    if ($NF + 0 > maximum + 0) {
      printf "CRAP score %.1f exceeds maximum %.1f for %s\n", $NF, maximum, $1 > "/dev/stderr"
      failed = 1
    }
  }
  END { exit failed }
' "$report"
