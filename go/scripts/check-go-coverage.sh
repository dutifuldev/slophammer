#!/usr/bin/env bash

set -euo pipefail

minimum_total_coverage="80.0"
declare -A package_minimums=(
  ["./internal/rules"]="90.0"
  ["./internal/scan"]="85.0"
  ["./internal/report"]="85.0"
)
coverfile="$(mktemp)"
package_coverfile="$(mktemp)"

cleanup() {
  rm -f "$coverfile"
  rm -f "$package_coverfile"
}
trap cleanup EXIT

coverage_total() {
  go tool cover -func="$1" |
    awk '/^total:/ {print substr($3, 1, length($3)-1)}'
}

go test ./... -coverprofile="$coverfile"

total="$(coverage_total "$coverfile")"
printf 'Total coverage: %s%%\n' "$total"
awk -v total="$total" -v minimum="$minimum_total_coverage" \
  'BEGIN { exit !(total + 0 >= minimum + 0) }'

for pkg in "${!package_minimums[@]}"; do
  minimum="${package_minimums[$pkg]}"
  go test "$pkg" -coverprofile="$package_coverfile" >/dev/null
  package_total="$(coverage_total "$package_coverfile")"
  printf '%s coverage: %s%%\n' "$pkg" "$package_total"
  awk -v total="$package_total" -v minimum="$minimum" \
    'BEGIN { exit !(total + 0 >= minimum + 0) }'
done
