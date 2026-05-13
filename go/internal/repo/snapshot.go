package repo

import (
	"path"
	"sort"
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

func (s Snapshot) HasFileNamedFold(want string) bool {
	for filePath := range s.Files {
		_, name := path.Split(filePath)
		if strings.EqualFold(name, want) {
			return true
		}
	}
	return false
}

func (s Snapshot) FilesUnder(dir string) []File {
	dir = normalizePath(dir)
	if dir == "." {
		dir = ""
	}
	if dir != "" && !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	files := make([]File, 0)
	for filePath, file := range s.Files {
		if strings.HasPrefix(filePath, dir) {
			files = append(files, file)
		}
	}
	sortFiles(files)
	return files
}

func (s Snapshot) FilesNamedFold(names ...string) []File {
	files := make([]File, 0)
	for filePath, file := range s.Files {
		_, got := path.Split(filePath)
		for _, want := range names {
			if strings.EqualFold(got, want) {
				files = append(files, file)
				break
			}
		}
	}
	sortFiles(files)
	return files
}

func (s Snapshot) FilesWithSuffix(suffixes ...string) []File {
	files := make([]File, 0)
	for filePath, file := range s.Files {
		for _, suffix := range suffixes {
			if strings.HasSuffix(filePath, suffix) {
				files = append(files, file)
				break
			}
		}
	}
	sortFiles(files)
	return files
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
	sortFiles(files)
	return files
}

func ContainsAny(files []File, needles ...string) bool {
	for _, file := range files {
		for _, needle := range needles {
			if strings.Contains(file.Content, needle) {
				return true
			}
		}
	}
	return false
}

func sortFiles(files []File) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
}

func normalizePath(value string) string {
	return strings.TrimPrefix(path.Clean(strings.ReplaceAll(value, "\\", "/")), "./")
}
