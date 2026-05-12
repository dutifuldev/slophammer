package repo

import "testing"

func TestSnapshotHasFileFold(t *testing.T) {
	snapshot := NewSnapshot("/repo", map[string]File{
		"README.md": {Path: "README.md"},
	})

	if !snapshot.HasFileFold("readme.md") {
		t.Fatal("expected case-insensitive file match")
	}
}

func TestSnapshotWorkflowFiles(t *testing.T) {
	snapshot := NewSnapshot("/repo", map[string]File{
		".github/workflows/ci.yml":     {Path: ".github/workflows/ci.yml"},
		".github/workflows/release.md": {Path: ".github/workflows/release.md"},
		"other.yml":                    {Path: "other.yml"},
	})

	workflows := snapshot.WorkflowFiles()
	if len(workflows) != 1 {
		t.Fatalf("len(workflows) = %d, want 1", len(workflows))
	}
	if workflows[0].Path != ".github/workflows/ci.yml" {
		t.Fatalf("workflow path = %q", workflows[0].Path)
	}
}

func TestNewSnapshotNormalizesNilFiles(t *testing.T) {
	snapshot := NewSnapshot("/repo", nil)
	if snapshot.Files == nil {
		t.Fatal("Files is nil")
	}
}
