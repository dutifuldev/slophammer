#!/usr/bin/env bash

set -euo pipefail

maximum_candidates="${MAXIMUM_DRY_CANDIDATES:-40}"
report="$(mktemp)"
trap 'rm -f "$report"' EXIT

go run github.com/unclebob/dry4go/cmd/dry4go@latest --format json . | tee "$report"

candidate_count="$(jq '.candidates | length' "$report")"
printf 'DRY candidates: %s; maximum: %s\n' "$candidate_count" "$maximum_candidates"
awk -v candidates="$candidate_count" -v maximum="$maximum_candidates" \
  'BEGIN { exit !(candidates + 0 <= maximum + 0) }'
