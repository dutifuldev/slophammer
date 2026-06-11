cargo check --workspace
cargo fmt --check
cargo clippy --workspace --all-targets -- -D warnings
cargo test --workspace --all-targets
cargo llvm-cov --workspace --fail-under-lines 85
cargo audit
slophammer-rs dry .
slophammer-rs check .
cargo mutants --workspace
