use slophammer_scan::Snapshot;

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
    path.starts_with(".github/workflows/")
        || path.starts_with("scripts/")
        || path.ends_with(".sh")
        || path.ends_with("justfile")
        || path.ends_with("Justfile")
        || path.ends_with("Makefile")
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
