use super::workflow_binding::binding_workflow_text;
use crate::scan::Snapshot;

pub fn command_text(snapshot: &Snapshot) -> String {
    let workflow_text = workflow_evidence(snapshot);
    let candidates = candidate_files(snapshot);
    let first_hop = reachable_contents(&candidates, &workflow_text);
    let extended = format!("{workflow_text}\n{first_hop}");
    let second_hop = reachable_contents(&candidates, &extended);
    format!("{workflow_text}\n{second_hop}")
}

/// Binding workflow evidence is the reachability root: workflows contribute
/// only steps that can run and fail; unparseable workflows stay credited raw.
fn workflow_evidence(snapshot: &Snapshot) -> String {
    snapshot
        .files
        .values()
        .filter(|file| active_workflow_path(&file.path))
        .map(|file| match binding_workflow_text(&file.content) {
            Some(text) => normalized_content(&text),
            None => normalized_content(&file.content),
        })
        .collect::<Vec<_>>()
        .join("\n")
}

struct CandidateFile {
    reference: String,
    content: String,
}

/// Scripts, Makefiles, and justfiles count only when binding evidence
/// references them, following script-to-script references one level deep.
fn candidate_files(snapshot: &Snapshot) -> Vec<CandidateFile> {
    snapshot
        .files
        .values()
        .filter(|file| candidate_path(&file.path))
        .map(|file| CandidateFile {
            reference: candidate_reference(&file.path),
            content: normalized_content(&file.content),
        })
        .collect()
}

fn reachable_contents(candidates: &[CandidateFile], evidence: &str) -> String {
    candidates
        .iter()
        .filter(|candidate| contains_word(evidence, &candidate.reference))
        .map(|candidate| candidate.content.clone())
        .collect::<Vec<_>>()
        .join("\n")
}

fn candidate_path(path: &str) -> bool {
    path.starts_with("scripts/")
        || path.contains("/scripts/")
        || path.ends_with(".sh")
        || path.ends_with("justfile")
        || path.ends_with("Justfile")
        || path.ends_with("Makefile")
}

/// Runner-driven files are referenced through their runner command rather
/// than their file name.
fn candidate_reference(path: &str) -> String {
    let base_name = path.rsplit('/').next().unwrap_or(path);
    match base_name.to_ascii_lowercase().as_str() {
        "makefile" => "make".to_owned(),
        "justfile" => "just".to_owned(),
        "taskfile.yml" | "taskfile.yaml" => "task".to_owned(),
        lowered => lowered.to_owned(),
    }
}

fn contains_word(evidence: &str, word: &str) -> bool {
    let bytes = evidence.as_bytes();
    let mut start = 0;
    while let Some(found) = evidence[start..].find(word) {
        let index = start + found;
        if word_boundary(bytes, index, word.len()) {
            return true;
        }
        start = index + 1;
    }
    false
}

fn word_boundary(bytes: &[u8], index: usize, length: usize) -> bool {
    if index > 0 && word_byte(bytes[index - 1]) {
        return false;
    }
    let end = index + length;
    end >= bytes.len() || !word_byte(bytes[end])
}

fn word_byte(value: u8) -> bool {
    value.is_ascii_alphanumeric() || value == b'_' || value == b'-'
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
    use crate::scan::RepoFile as File;
    use std::collections::BTreeMap;

    fn snapshot(files: &[(&str, &str)]) -> Snapshot {
        let mut map = BTreeMap::new();
        for (path, content) in files {
            map.insert(
                (*path).to_owned(),
                File {
                    path: (*path).to_owned(),
                    content: (*content).to_owned(),
                },
            );
        }
        Snapshot {
            root: "/repo".into(),
            files: map,
        }
    }

    #[test]
    fn only_direct_workflow_children_are_active_evidence() {
        assert!(active_workflow_path(".github/workflows/ci.yml"));
        assert!(active_workflow_path(".github/workflows/release.yaml"));
        assert!(!active_workflow_path(".github/workflows/archive/old.yml"));
        assert!(!active_workflow_path(".github/workflows/notes.txt"));
    }

    #[test]
    fn unreferenced_scripts_are_not_evidence() {
        let text = command_text(&snapshot(&[
            (
                ".github/workflows/ci.yml",
                "on: [push]\njobs:\n  ci:\n    steps:\n      - run: cargo test\n",
            ),
            ("scripts/hidden.sh", "cargo mutants --workspace --list\n"),
        ]));
        assert!(text.contains("cargo test"));
        assert!(!text.contains("cargo mutants"));
    }

    #[test]
    fn referenced_scripts_are_evidence_one_hop_deep() {
        let text = command_text(&snapshot(&[
            (
                ".github/workflows/ci.yml",
                "on: [push]\njobs:\n  ci:\n    steps:\n      - run: ./scripts/gate.sh\n",
            ),
            ("scripts/gate.sh", "cargo test\n./scripts/audit.sh\n"),
            ("scripts/audit.sh", "cargo audit\n"),
        ]));
        assert!(text.contains("cargo test"));
        assert!(text.contains("cargo audit"));
    }

    #[test]
    fn makefiles_need_a_make_invocation() {
        let unreferenced = command_text(&snapshot(&[
            (
                ".github/workflows/ci.yml",
                "on: [push]\njobs:\n  ci:\n    steps:\n      - run: cargo check\n",
            ),
            ("Makefile", "test:\n\tcargo test\n"),
        ]));
        assert!(!unreferenced.contains("cargo test"));

        let referenced = command_text(&snapshot(&[
            (
                ".github/workflows/ci.yml",
                "on: [push]\njobs:\n  ci:\n    steps:\n      - run: make test\n",
            ),
            ("Makefile", "test:\n\tcargo test\n"),
        ]));
        assert!(referenced.contains("cargo test"));
    }
}
