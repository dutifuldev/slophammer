package scan

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRepoScansFilesAndSkipsHeavyDirs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", "# Test\n")
	writeFile(t, root, ".github/workflows/ci.yml", "name: CI\n")
	writeFile(t, root, ".git/config", "ignored\n")

	snapshot, err := Repo(root)
	if err != nil {
		t.Fatalf("Repo returned error: %v", err)
	}
	if !snapshot.HasFileFold("README.md") {
		t.Fatal("snapshot is missing README.md")
	}
	if snapshot.HasFileFold(".git/config") {
		t.Fatal("snapshot included .git/config")
	}
	if len(snapshot.WorkflowFiles()) != 1 {
		t.Fatalf("len(WorkflowFiles) = %d, want 1", len(snapshot.WorkflowFiles()))
	}
}

func TestRepoSkipsLargeAndBinaryContent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "large.txt", string(make([]byte, maxFileBytes+1)))
	writeFile(t, root, "binary.dat", "hello\x00world")

	snapshot, err := Repo(root)
	if err != nil {
		t.Fatalf("Repo returned error: %v", err)
	}
	if snapshot.Files["large.txt"].Content != "" {
		t.Fatal("large file content was read")
	}
	if snapshot.Files["binary.dat"].Content != "" {
		t.Fatal("binary file content was retained")
	}
}

func TestRepoReturnsErrorForMissingRoot(t *testing.T) {
	_, err := Repo(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("Repo returned nil error for missing root")
	}
}

func TestReadSmallTextFileReturnsInfoError(t *testing.T) {
	_, err := readSmallTextFile("missing", errDirEntry{})
	if err == nil {
		t.Fatal("readSmallTextFile returned nil error")
	}
}

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

type errDirEntry struct{}

func (errDirEntry) Name() string               { return "broken" }
func (errDirEntry) IsDir() bool                { return false }
func (errDirEntry) Type() os.FileMode          { return 0 }
func (errDirEntry) Info() (os.FileInfo, error) { return nil, errors.New("info failed") }
