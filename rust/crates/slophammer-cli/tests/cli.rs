use std::process::Command;

#[path = "support/fixtures.rs"]
mod fixtures;
use fixtures::{fixture, fixture_path};

#[test]
fn cli_exposes_version() {
    let output = command()
        .arg("--version")
        .output()
        .expect("run slophammer-rs");
    assert!(output.status.success());
    assert_eq!(
        stdout(&output).trim(),
        format!("slophammer-rs {}", env!("CARGO_PKG_VERSION"))
    );
}

#[test]
fn cli_checks_clean_fixture() {
    let fixture = fixture("rust-clean");
    let root = fixture_path(&fixture);
    let output = command()
        .args(["check", &root, "--format", "json"])
        .output()
        .expect("run slophammer-rs");
    assert!(output.status.success());
    assert!(stdout(&output).contains("\"ok\": true"));
}

#[test]
fn cli_reports_findings_exit_code() {
    let fixture = fixture("rust-unsafe");
    let root = fixture_path(&fixture);
    let output = command()
        .args(["check", &root, "--format", "json"])
        .output()
        .expect("run slophammer-rs");
    assert_eq!(output.status.code(), Some(1));
    assert!(stdout(&output).contains("rust.unsafe-policy-required"));
}

#[test]
fn cli_reports_config_errors() {
    let fixture = fixture("rust-invalid-config");
    let root = fixture_path(&fixture);
    let output = command()
        .args(["check", &root, "--format", "json"])
        .output()
        .expect("run slophammer-rs");
    assert_eq!(output.status.code(), Some(2));
    assert!(stderr(&output).contains("config failed"));
}

#[test]
fn cli_exposes_direct_commands_and_rules() {
    let rules = command()
        .args(["rules", "--format", "json"])
        .output()
        .expect("run slophammer-rs");
    assert!(rules.status.success());
    assert!(stdout(&rules).contains("rust.check-required"));

    let bad_dependency_fixture = fixture("rust-bad-dependency");
    let bad_dependency_root = fixture_path(&bad_dependency_fixture);
    let boundaries = command()
        .args(["boundaries", &bad_dependency_root, "--format", "json"])
        .output()
        .expect("run slophammer-rs");
    assert_eq!(boundaries.status.code(), Some(1));
    assert!(stdout(&boundaries).contains("rust.dependency-boundaries-required"));

    let unsafe_fixture = fixture("rust-unsafe");
    let unsafe_root = fixture_path(&unsafe_fixture);
    let unsafe_result = command()
        .args(["unsafe", &unsafe_root, "--format", "json"])
        .output()
        .expect("run slophammer-rs");
    assert_eq!(unsafe_result.status.code(), Some(1));
    assert!(stdout(&unsafe_result).contains("rust.unsafe-policy-required"));
}

fn command() -> Command {
    Command::new(env!("CARGO_BIN_EXE_slophammer-rs"))
}

fn stdout(output: &std::process::Output) -> String {
    String::from_utf8_lossy(&output.stdout).into_owned()
}

fn stderr(output: &std::process::Output) -> String {
    String::from_utf8_lossy(&output.stderr).into_owned()
}
