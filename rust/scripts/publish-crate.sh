#!/usr/bin/env bash
set -euo pipefail

package="slophammer-rs"
mode="dry-run"
tag=""

usage() {
  cat <<'USAGE'
Usage: scripts/publish-crate.sh [--dry-run | --publish] --tag rust/vX.Y.Z

Packages or publishes the single user-facing Rust CLI package, slophammer-rs.
Dry-run mode runs cargo package. Publish mode requires CARGO_REGISTRY_TOKEN and
skips the package version if it is already published on crates.io.
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

package_version_published() {
  local status
  status="$(
    curl \
      -sS \
      -L \
      -A "slophammer-rs-release" \
      -o /dev/null \
      -w '%{http_code}' \
      "https://crates.io/api/v1/crates/${package}/${version}" || true
  )"
  [ "$status" = "200" ]
}

wait_for_package_version() {
  local attempt

  for attempt in $(seq 1 60); do
    if package_version_published; then
      echo "${package}@${version} is visible on crates.io."
      return 0
    fi
    echo "Waiting for ${package}@${version} to become visible on crates.io (${attempt}/60)."
    sleep 10
  done

  echo "${package}@${version} was not visible on crates.io after waiting." >&2
  return 1
}

package_crate() {
  local args=(package -p "$package" --locked)

  if [ "${ALLOW_DIRTY:-}" = "1" ]; then
    args+=(--allow-dirty)
  fi

  cargo "${args[@]}"
}

dry_run() {
  echo "== package ${package}@${version} =="
  package_crate
}

publish() {
  if [ -z "${CARGO_REGISTRY_TOKEN:-}" ]; then
    echo "CARGO_REGISTRY_TOKEN is required for crates.io publishing." >&2
    exit 1
  fi

  if package_version_published; then
    echo "${package}@${version} is already published; skipping."
    return 0
  fi

  echo "== publish ${package}@${version} =="
  cargo publish -p "$package" --locked
  wait_for_package_version
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
