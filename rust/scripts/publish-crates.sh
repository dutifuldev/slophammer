#!/usr/bin/env bash
set -euo pipefail

crates=(
  slophammer-core
  slophammer-scan
  slophammer-config
  slophammer-report
  slophammer-rust
  slophammer-exec
  slophammer-app
  slophammer-rs
)

mode="dry-run"
tag=""

usage() {
  cat <<'USAGE'
Usage: scripts/publish-crates.sh [--dry-run | --publish] --tag rust/vX.Y.Z

Runs the ordered Rust workspace crates.io package/publish sequence.
Dry-run mode packages what can be verified before internal crates exist on
crates.io, and explains deferred crates. Publish mode requires
CARGO_REGISTRY_TOKEN and skips crate versions that are already published.
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --dry-run)
      mode="dry-run"
      ;;
    --publish)
      mode="publish"
      ;;
    --tag)
      shift
      tag="${1:-}"
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

if [ ! -f Cargo.toml ] || [ ! -d crates/slophammer-cli ]; then
  echo "run this script from the rust/ workspace root" >&2
  exit 2
fi

version="$(
  awk '
    /^\[workspace.package\]/ { in_workspace_package = 1; next }
    /^\[/ { in_workspace_package = 0 }
    in_workspace_package && /^version = / {
      gsub(/"/, "", $3)
      print $3
      exit
    }
  ' Cargo.toml
)"

if [ -z "$version" ]; then
  echo "could not read workspace package version from Cargo.toml" >&2
  exit 1
fi

expected_tag="rust/v$version"

if [ -z "$tag" ]; then
  echo "--tag is required; expected $expected_tag" >&2
  usage >&2
  exit 2
fi

if ! printf '%s' "$tag" | grep -Eq '^rust/v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$'; then
  echo "invalid Rust release tag: $tag" >&2
  exit 1
fi

if [ "$tag" != "$expected_tag" ]; then
  echo "release tag $tag does not match Rust workspace version $version; expected $expected_tag" >&2
  exit 1
fi

crate_version_published() {
  local crate="$1"
  local status
  status="$(
    curl \
      -sS \
      -L \
      -A "slophammer-rust-release" \
      -o /dev/null \
      -w '%{http_code}' \
      "https://crates.io/api/v1/crates/${crate}/${version}" || true
  )"
  [ "$status" = "200" ]
}

wait_for_crate_version() {
  local crate="$1"
  local attempt

  for attempt in $(seq 1 60); do
    if crate_version_published "$crate"; then
      echo "${crate}@${version} is visible on crates.io."
      return 0
    fi
    echo "Waiting for ${crate}@${version} to become visible on crates.io (${attempt}/60)."
    sleep 10
  done

  echo "${crate}@${version} was not visible on crates.io after waiting." >&2
  return 1
}

package_crate() {
  local crate="$1"
  local args=(package -p "$crate" --locked)

  if [ "${ALLOW_DIRTY:-}" = "1" ]; then
    args+=(--allow-dirty)
  fi

  cargo "${args[@]}"
}

dry_run() {
  local crate
  local output

  for crate in "${crates[@]}"; do
    echo "== package ${crate}@${version} =="
    output="$(mktemp)"
    if package_crate "$crate" >"$output" 2>&1; then
      cat "$output"
      rm -f "$output"
      continue
    fi

    cat "$output"
    if grep -q 'no matching package named `slophammer-' "$output"; then
      echo "Deferring ${crate}: an earlier internal crate is not published to crates.io yet."
      rm -f "$output"
      continue
    fi

    rm -f "$output"
    return 1
  done
}

publish() {
  local crate

  if [ -z "${CARGO_REGISTRY_TOKEN:-}" ]; then
    echo "CARGO_REGISTRY_TOKEN is required for crates.io publishing." >&2
    exit 1
  fi

  for crate in "${crates[@]}"; do
    if crate_version_published "$crate"; then
      echo "${crate}@${version} is already published; skipping."
      continue
    fi

    echo "== publish ${crate}@${version} =="
    cargo publish -p "$crate" --locked
    wait_for_crate_version "$crate"
  done
}

case "$mode" in
  dry-run)
    dry_run
    ;;
  publish)
    publish
    ;;
  *)
    echo "invalid mode: $mode" >&2
    exit 2
    ;;
esac
