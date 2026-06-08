#!/usr/bin/env bash
set -euo pipefail

package="slophammer-rs"

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

package_dir="target/package/${package}-${version}"

if [ ! -f "$package_dir/Cargo.toml" ]; then
  echo "verified package directory not found: $package_dir" >&2
  echo "run scripts/publish-crate.sh --dry-run --tag rust/v${version} first" >&2
  exit 1
fi

cargo test --manifest-path "$package_dir/Cargo.toml" --locked
