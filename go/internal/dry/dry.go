package dry

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	DefaultStructuralThreshold = 0.82
	DefaultStructuralMinLines  = 4
	DefaultStructuralMinNodes  = 20
	DefaultCopiedBlockTokens   = 100
)

type Options struct {
	Root                string
	Paths               []string
	StructuralEnabled   bool
	StructuralThreshold float64
	StructuralMinLines  int
	StructuralMinNodes  int
	CopiedBlockEnabled  bool
	CopiedBlockTokens   int
}

type Report struct {
	Findings []Finding `json:"findings"`
	Groups   []Group   `json:"groups"`
}

type Finding struct {
	Kind   string  `json:"kind"`
	Left   Range   `json:"left"`
	Right  Range   `json:"right"`
	Score  float64 `json:"score,omitempty"`
	Tokens int     `json:"tokens,omitempty"`
	Nodes  int     `json:"nodes,omitempty"`
	Engine string  `json:"engine"`
}

type Range struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

type Group struct {
	ID       string   `json:"id"`
	Findings []int    `json:"findings"`
	Kinds    []string `json:"kinds"`
	Left     Range    `json:"left"`
	Right    Range    `json:"right"`
}

type sourceFile struct {
	Path    string
	Content []byte
}

func Find(options Options) (Report, error) {
	options = withDefaults(options)
	files, err := loadFiles(options)
	if err != nil {
		return Report{}, err
	}

	var findings []Finding
	if options.StructuralEnabled {
		structural, err := findStructural(files, options)
		if err != nil {
			return Report{}, err
		}
		findings = append(findings, structural...)
	}
	if options.CopiedBlockEnabled {
		copied, err := findCopiedBlocks(files, options)
		if err != nil {
			return Report{}, err
		}
		findings = append(findings, copied...)
	}
	sortFindings(findings)
	return Report{Findings: findings, Groups: groupFindings(findings)}, nil
}

func withDefaults(options Options) Options {
	if options.Root == "" {
		options.Root = "."
	}
	if len(options.Paths) == 0 {
		options.Paths = []string{"."}
	}
	if !options.StructuralEnabled && !options.CopiedBlockEnabled {
		options.StructuralEnabled = true
		options.CopiedBlockEnabled = true
	}
	options = withStructuralDefaults(options)
	options = withCopiedBlockDefaults(options)
	return options
}

func withStructuralDefaults(options Options) Options {
	if options.StructuralThreshold == 0 {
		options.StructuralThreshold = DefaultStructuralThreshold
	}
	if options.StructuralMinLines == 0 {
		options.StructuralMinLines = DefaultStructuralMinLines
	}
	if options.StructuralMinNodes == 0 {
		options.StructuralMinNodes = DefaultStructuralMinNodes
	}
	return options
}

func withCopiedBlockDefaults(options Options) Options {
	if options.CopiedBlockTokens == 0 {
		options.CopiedBlockTokens = DefaultCopiedBlockTokens
	}
	return options
}

func loadFiles(options Options) ([]sourceFile, error) {
	seen := map[string]bool{}
	var files []sourceFile
	for _, target := range options.Paths {
		if err := collectTargetFiles(options.Root, target, seen, &files); err != nil {
			return nil, err
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func collectTargetFiles(root string, target string, seen map[string]bool, files *[]sourceFile) error {
	abs := filepath.Join(root, filepath.Clean(target))
	info, err := os.Stat(abs)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return appendGoFile(root, abs, seen, files)
	}
	return filepath.WalkDir(abs, func(filePath string, entry os.DirEntry, walkErr error) error {
		return collectWalkedFile(root, abs, filePath, entry, walkErr, seen, files)
	})
}

func collectWalkedFile(
	root string,
	walkRoot string,
	filePath string,
	entry os.DirEntry,
	walkErr error,
	seen map[string]bool,
	files *[]sourceFile,
) error {
	if walkErr != nil {
		return walkErr
	}
	if entry.IsDir() {
		if shouldSkipDir(entry.Name()) && filePath != walkRoot {
			return filepath.SkipDir
		}
		return nil
	}
	return appendGoFile(root, filePath, seen, files)
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "node_modules", ".venv", ".mypy_cache", ".pytest_cache", ".ruff_cache", "__pycache__",
		"vendor", "target", "dist", "build":
		return true
	default:
		return false
	}
}

func appendGoFile(root string, filePath string, seen map[string]bool, files *[]sourceFile) error {
	if !strings.HasSuffix(filePath, ".go") {
		return nil
	}
	rel, err := filepath.Rel(root, filePath)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	if seen[rel] {
		return nil
	}
	content, err := os.ReadFile(filePath) // #nosec G304 -- paths are explicit user-selected scan inputs.
	if err != nil {
		return err
	}
	seen[rel] = true
	*files = append(*files, sourceFile{Path: rel, Content: content})
	return nil
}
