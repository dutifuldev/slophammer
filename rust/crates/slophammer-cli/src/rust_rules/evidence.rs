use crate::scan::Snapshot;

pub fn command_text(snapshot: &Snapshot) -> String {
    snapshot
        .files
        .values()
        .filter(|file| evidence_path(&file.path))
        .map(|file| normalized_content(&file.content))
        .collect::<Vec<_>>()
        .join("\n")
}

fn evidence_path(path: &str) -> bool {
    active_workflow_path(path)
        || path.starts_with("scripts/")
        || path.ends_with(".sh")
        || path.ends_with("justfile")
        || path.ends_with("Justfile")
        || path.ends_with("Makefile")
}

fn active_workflow_path(path: &str) -> bool {
    let Some(name) = path.strip_prefix(".github/workflows/") else {
        return false;
    };
    !name.contains('/') && (name.ends_with(".yml") || name.ends_with(".yaml"))
}

fn normalized_content(content: &str) -> String {
    content
        .lines()
        .filter_map(strip_comment)
        .collect::<Vec<_>>()
        .join("\n")
        .replace("\\\n", " ")
        .split_whitespace()
        .collect::<Vec<_>>()
        .join(" ")
        .to_ascii_lowercase()
}

fn strip_comment(line: &str) -> Option<&str> {
    let trimmed = line.trim_start();
    if trimmed.starts_with('#') {
        return None;
    }
    Some(trimmed.split('#').next().unwrap_or(trimmed))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn only_direct_workflow_children_are_active_evidence() {
        assert!(evidence_path(".github/workflows/ci.yml"));
        assert!(evidence_path(".github/workflows/release.yaml"));
        assert!(!evidence_path(".github/workflows/archive/old.yml"));
        assert!(!evidence_path(".github/workflows/notes.txt"));
    }
}
