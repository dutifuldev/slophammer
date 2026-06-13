use yaml_serde::Value;

/// Returns the command evidence a workflow may contribute: run scripts and
/// action references from steps that can execute and can fail on integration
/// branches. Returns `None` when the workflow cannot be structurally
/// filtered (unparseable, or no jobs mapping), in which case the caller
/// keeps the raw text so filtering only removes false passes.
pub fn binding_workflow_text(content: &str) -> Option<String> {
    let document: Value = yaml_serde::from_str(content).ok()?;
    let mapping = document.as_mapping()?;
    let jobs = mapping.get("jobs")?.as_mapping()?;
    if jobs.is_empty() {
        return None;
    }
    if !binding_triggers(workflow_triggers(&document)) {
        return Some(String::new());
    }
    let mut evidence = Vec::new();
    for (_, job) in jobs {
        collect_job_evidence(job, &mut evidence);
    }
    Some(evidence.join("\n"))
}

/// YAML 1.1 resolves a plain `on` key as boolean true, so the trigger entry
/// is looked up under both spellings.
fn workflow_triggers(document: &Value) -> Option<&Value> {
    let mapping = document.as_mapping()?;
    mapping.get("on").or_else(|| mapping.get(Value::Bool(true)))
}

fn collect_job_evidence(job: &Value, evidence: &mut Vec<String>) {
    if neutralized(job) {
        return;
    }
    let Some(steps) = job.get("steps").and_then(Value::as_sequence) else {
        return;
    };
    let matrix_commands = job_matrix_commands(job);
    for step in steps {
        collect_step_evidence(step, &matrix_commands, evidence);
    }
}

fn collect_step_evidence(step: &Value, matrix_commands: &[String], evidence: &mut Vec<String>) {
    if neutralized(step) {
        return;
    }
    if let Some(uses) = step.get("uses").and_then(Value::as_str) {
        if !uses.trim().is_empty() {
            evidence.push(format!("uses: {uses}"));
        }
    }
    let Some(run) = step.get("run").and_then(Value::as_str) else {
        return;
    };
    if run.trim().is_empty() {
        return;
    }
    if direct_matrix_command(run) && !matrix_commands.is_empty() {
        evidence.extend(matrix_commands.iter().cloned());
        return;
    }
    evidence.push(run.to_owned());
}

/// A run consisting solely of the matrix.command expression expands to the
/// job's matrix command values, mirroring the raw-text evidence that the
/// structural filter would otherwise drop.
fn direct_matrix_command(run: &str) -> bool {
    let trimmed = run.trim();
    trimmed
        .strip_prefix("${{")
        .and_then(|rest| rest.strip_suffix("}}"))
        .is_some_and(|inner| inner.trim() == "matrix.command")
}

fn job_matrix_commands(job: &Value) -> Vec<String> {
    let Some(matrix) = job
        .get("strategy")
        .and_then(|strategy| strategy.get("matrix"))
    else {
        return Vec::new();
    };
    let mut commands = sequence_strings(matrix.get("command"));
    if let Some(include) = matrix.get("include").and_then(Value::as_sequence) {
        commands.extend(
            include
                .iter()
                .filter_map(|item| item.get("command"))
                .filter_map(Value::as_str)
                .map(str::to_owned),
        );
    }
    commands
}

fn sequence_strings(value: Option<&Value>) -> Vec<String> {
    value
        .and_then(Value::as_sequence)
        .map(|items| {
            items
                .iter()
                .filter_map(Value::as_str)
                .map(str::to_owned)
                .collect()
        })
        .unwrap_or_default()
}

/// A job or step is neutralized when it cannot run or cannot fail: a literal
/// false `if:` condition or a literal `continue-on-error: true`. Non-literal
/// expressions stay credited; the checker ships no expression evaluator.
fn neutralized(entry: &Value) -> bool {
    if literal_bool(entry.get("continue-on-error"), true) {
        return true;
    }
    literal_bool(entry.get("if"), false)
}

fn literal_bool(value: Option<&Value>, wanted: bool) -> bool {
    match value {
        Some(Value::Bool(actual)) => *actual == wanted,
        Some(Value::String(text)) => literal_bool_text(text, wanted),
        _ => false,
    }
}

fn literal_bool_text(text: &str, wanted: bool) -> bool {
    let mut trimmed = text.trim();
    if let Some(inner) = trimmed
        .strip_prefix("${{")
        .and_then(|rest| rest.strip_suffix("}}"))
    {
        trimmed = inner.trim();
    }
    trimmed == if wanted { "true" } else { "false" }
}

/// Triggers bind when the workflow can fire for integration: pull requests,
/// merge groups, schedules, or pushes whose branch filter is absent,
/// wildcarded, or names an integration branch.
fn binding_triggers(triggers: Option<&Value>) -> bool {
    match triggers {
        Some(Value::String(name)) => binding_trigger_name(name),
        Some(Value::Sequence(names)) => names
            .iter()
            .filter_map(Value::as_str)
            .any(binding_trigger_name),
        Some(Value::Mapping(entries)) => entries.iter().any(|(name, value)| {
            name.as_str()
                .is_some_and(|name| binding_trigger_entry(name, value))
        }),
        _ => false,
    }
}

fn binding_trigger_name(name: &str) -> bool {
    matches!(
        name,
        "push" | "pull_request" | "pull_request_target" | "merge_group" | "schedule"
    )
}

fn binding_trigger_entry(name: &str, value: &Value) -> bool {
    match name {
        "pull_request" | "pull_request_target" | "merge_group" | "schedule" => true,
        "push" => binding_push_filter(value),
        _ => false,
    }
}

fn binding_push_filter(value: &Value) -> bool {
    let Some(branches) = value.get("branches") else {
        // Defining only tags or tags-ignore stops the workflow from firing
        // for branch pushes entirely, so it is a release trigger, not
        // integration CI; a branches-ignore filter still fires for branches.
        if value.get("branches-ignore").is_some() {
            return true;
        }
        return value.get("tags").is_none() && value.get("tags-ignore").is_none();
    };
    match branches {
        Value::String(pattern) => integration_branch_pattern(pattern),
        Value::Sequence(patterns) => patterns
            .iter()
            .filter_map(Value::as_str)
            .any(integration_branch_pattern),
        _ => true,
    }
}

fn integration_branch_pattern(pattern: &str) -> bool {
    pattern.contains('*') || matches!(pattern, "main" | "master" | "trunk" | "develop")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn neutralized_workflow_contributes_no_evidence() {
        let workflow = r#"
on:
  push:
    branches: [branch-that-never-existed]
jobs:
  go:
    steps:
      - run: go test ./...
"#;
        assert_eq!(binding_workflow_text(workflow), Some(String::new()));
    }

    #[test]
    fn neutralized_steps_and_jobs_are_dropped() {
        let workflow = r#"
on: [push]
jobs:
  skipped:
    if: false
    steps:
      - run: cargo test --workspace
  soft:
    steps:
      - run: cargo clippy
        continue-on-error: true
      - run: cargo fmt --check
"#;
        assert_eq!(
            binding_workflow_text(workflow),
            Some("cargo fmt --check".to_owned())
        );
    }

    #[test]
    fn surviving_steps_contribute_runs_and_uses() {
        let workflow = r#"
on:
  pull_request:
jobs:
  ci:
    steps:
      - uses: actions/checkout@v6
      - run: cargo test --workspace
"#;
        let text = binding_workflow_text(workflow).expect("binding text");
        assert!(text.contains("uses: actions/checkout@v6"));
        assert!(text.contains("cargo test --workspace"));
    }

    #[test]
    fn non_literal_expressions_stay_credited() {
        let workflow = r#"
on: [push]
jobs:
  ci:
    if: github.repository == 'no/such-repo'
    steps:
      - run: cargo test --workspace
        continue-on-error: ${{ matrix.experimental }}
"#;
        let text = binding_workflow_text(workflow).expect("binding text");
        assert!(text.contains("cargo test --workspace"));
    }

    #[test]
    fn matrix_commands_expand_into_evidence() {
        let workflow = r#"
on: [push]
jobs:
  ci:
    strategy:
      matrix:
        command: [cargo test --workspace]
        include:
          - command: cargo audit
    steps:
      - run: ${{ matrix.command }}
"#;
        let text = binding_workflow_text(workflow).expect("binding text");
        assert!(text.contains("cargo test --workspace"));
        assert!(text.contains("cargo audit"));
        assert!(!text.contains("matrix.command"));
    }

    #[test]
    fn tag_only_push_triggers_do_not_bind() {
        let workflow = r#"
on:
  push:
    tags: ["v*"]
jobs:
  release:
    steps:
      - run: cargo test --workspace
"#;
        assert_eq!(binding_workflow_text(workflow), Some(String::new()));
    }

    #[test]
    fn branches_ignore_push_triggers_still_bind() {
        let workflow = r#"
on:
  push:
    branches-ignore: [release-only]
jobs:
  ci:
    steps:
      - run: cargo test --workspace
"#;
        let text = binding_workflow_text(workflow).expect("binding text");
        assert!(text.contains("cargo test --workspace"));
    }

    #[test]
    fn unparseable_workflows_fall_back_to_raw() {
        assert_eq!(binding_workflow_text("cargo test --workspace"), None);
        assert_eq!(binding_workflow_text(": ["), None);
    }
}
