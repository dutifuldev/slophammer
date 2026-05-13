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

func TestSnapshotHasFileNamedFold(t *testing.T) {
	snapshot := NewSnapshot("/repo", map[string]File{
		"go/go.mod": {Path: "go/go.mod"},
	})

	if !snapshot.HasFileNamedFold("GO.MOD") {
		t.Fatal("expected case-insensitive base name match")
	}
}

func TestSnapshotFilesUnder(t *testing.T) {
	snapshot := NewSnapshot("/repo", map[string]File{
		"scripts/check.sh": {Path: "scripts/check.sh"},
		"scripts/nested/a": {Path: "scripts/nested/a"},
		"other/check.sh":   {Path: "other/check.sh"},
	})

	files := snapshot.FilesUnder("scripts")
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	if files[0].Path != "scripts/check.sh" {
		t.Fatalf("files[0].Path = %q, want scripts/check.sh", files[0].Path)
	}
}

func TestSnapshotFilesNamedFold(t *testing.T) {
	snapshot := NewSnapshot("/repo", map[string]File{
		"go/.golangci.yml":   {Path: "go/.golangci.yml"},
		"other/.golangci.md": {Path: "other/.golangci.md"},
	})

	files := snapshot.FilesNamedFold(".GOLANGCI.YML")
	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(files))
	}
	if files[0].Path != "go/.golangci.yml" {
		t.Fatalf("files[0].Path = %q, want go/.golangci.yml", files[0].Path)
	}
}

func TestSnapshotFilesWithSuffix(t *testing.T) {
	snapshot := NewSnapshot("/repo", map[string]File{
		"main.go":      {Path: "main.go"},
		"main_test.go": {Path: "main_test.go"},
		"README.md":    {Path: "README.md"},
	})

	files := snapshot.FilesWithSuffix(".go")
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
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

func TestContainsAny(t *testing.T) {
	files := []File{
		{Path: "scripts/check.sh", Content: "go test ./..."},
	}
	if !ContainsAny(files, "go test ./...") {
		t.Fatal("expected matching content")
	}
	if ContainsAny(files, "go vet ./...") {
		t.Fatal("unexpected matching content")
	}
}

func TestNewSnapshotNormalizesNilFiles(t *testing.T) {
	snapshot := NewSnapshot("/repo", nil)
	if snapshot.Files == nil {
		t.Fatal("Files is nil")
	}
}
