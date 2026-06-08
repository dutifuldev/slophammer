use std::process::Command;

#[test]
fn cli_checks_clean_fixture() {
    let output = command()
        .args(["check", &fixture("rust-clean"), "--format", "json"])
        .output()
        .expect("run slophammer-rs");
    assert!(output.status.success());
    assert!(stdout(&output).contains("\"ok\": true"));
}

#[test]
fn cli_reports_findings_exit_code() {
    let output = command()
        .args(["check", &fixture("rust-unsafe"), "--format", "json"])
        .output()
        .expect("run slophammer-rs");
    assert_eq!(output.status.code(), Some(1));
    assert!(stdout(&output).contains("rust.unsafe-policy-required"));
}

#[test]
fn cli_reports_config_errors() {
    let output = command()
        .args(["check", &fixture("rust-invalid-config"), "--format", "json"])
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

    let boundaries = command()
        .args([
            "boundaries",
            &fixture("rust-bad-dependency"),
            "--format",
            "json",
        ])
        .output()
        .expect("run slophammer-rs");
    assert_eq!(boundaries.status.code(), Some(1));
    assert!(stdout(&boundaries).contains("rust.dependency-boundaries-required"));

    let unsafe_result = command()
        .args(["unsafe", &fixture("rust-unsafe"), "--format", "json"])
        .output()
        .expect("run slophammer-rs");
    assert_eq!(unsafe_result.status.code(), Some(1));
    assert!(stdout(&unsafe_result).contains("rust.unsafe-policy-required"));
}

fn command() -> Command {
    Command::new(env!("CARGO_BIN_EXE_slophammer-rs"))
}

fn fixture(name: &str) -> String {
    let manifest_dir = std::path::Path::new(env!("CARGO_MANIFEST_DIR"));
    manifest_dir
        .join("../../..")
        .join("fixtures/repos")
        .join(name)
        .to_string_lossy()
        .into_owned()
}

fn stdout(output: &std::process::Output) -> String {
    String::from_utf8_lossy(&output.stdout).into_owned()
}

fn stderr(output: &std::process::Output) -> String {
    String::from_utf8_lossy(&output.stderr).into_owned()
}
