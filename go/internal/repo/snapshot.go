package repo

import (
	"path"
	"strings"
)

type File struct {
	Path    string
	Content string
}

type Snapshot struct {
	Root  string
	Files map[string]File
}

func NewSnapshot(root string, files map[string]File) Snapshot {
	if files == nil {
		files = map[string]File{}
	}
	return Snapshot{Root: root, Files: files}
}

func (s Snapshot) HasFileFold(want string) bool {
	want = normalizePath(want)
	for got := range s.Files {
		if strings.EqualFold(got, want) {
			return true
		}
	}
	return false
}

func (s Snapshot) WorkflowFiles() []File {
	files := make([]File, 0)
	for filePath, file := range s.Files {
		dir, name := path.Split(filePath)
		if dir != ".github/workflows/" {
			continue
		}
		if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
			files = append(files, file)
		}
	}
	return files
}

func normalizePath(value string) string {
	return strings.TrimPrefix(path.Clean(strings.ReplaceAll(value, "\\", "/")), "./")
}
