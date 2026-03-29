// Package scanner provides file walking and noise classification.
package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// NoiseLevel indicates how noisy a file/dir is for LLM context.
type NoiseLevel int

const (
	NoiseLevelNone   NoiseLevel = 0 // Signal — keep
	NoiseLevelYellow NoiseLevel = 1 // Optional — test files, migrations
	NoiseLevelRed    NoiseLevel = 2 // Strong noise — definitely ignore
)

// NoiseCategory names the reason a pattern is noisy.
type NoiseCategory string

const (
	CategoryLockfile      NoiseCategory = "lockfile, auto-generated"
	CategoryBuildArtifact NoiseCategory = "build output, generated"
	CategoryDependency    NoiseCategory = "dependencies, never edit"
	CategoryGenerated     NoiseCategory = "generated code, do not edit"
	CategoryBinary        NoiseCategory = "binary file, not readable"
	CategoryTestArtifact  NoiseCategory = "test artifacts"
	CategoryTestFile      NoiseCategory = "test files (optional)"
	CategoryIDEOS         NoiseCategory = "IDE/OS noise"
)

// Classification holds the result of classifying a file or directory.
type Classification struct {
	Level    NoiseLevel
	Category NoiseCategory
	// Pattern is the glob pattern for aggregation (e.g., "*_test.go", "*.min.js").
	// Empty for exact-match items like lockfiles or named directories.
	Pattern string
}

// Classifier classifies files/dirs by noise level.
type Classifier struct{}

// NewClassifier creates a new Classifier.
func NewClassifier() *Classifier {
	return &Classifier{}
}

// lockfileNames is the set of exact filenames that are lockfiles.
var lockfileNames = map[string]bool{
	"package-lock.json": true,
	"yarn.lock":         true,
	"Cargo.lock":        true,
	"Gemfile.lock":      true,
	"poetry.lock":       true,
	"go.sum":            true,
	"pnpm-lock.yaml":    true,
	"composer.lock":     true,
	"Pipfile.lock":      true,
	"bun.lockb":         true,
}

// noiseDirNames are directory names that are always noise.
var noiseDirNames = map[string]struct {
	level    NoiseLevel
	category NoiseCategory
}{
	"node_modules":  {NoiseLevelRed, CategoryDependency},
	"vendor":        {NoiseLevelRed, CategoryDependency},
	".venv":         {NoiseLevelRed, CategoryDependency},
	"venv":          {NoiseLevelRed, CategoryDependency},
	"dist":          {NoiseLevelRed, CategoryBuildArtifact},
	"build":         {NoiseLevelRed, CategoryBuildArtifact},
	"out":           {NoiseLevelRed, CategoryBuildArtifact},
	"target":        {NoiseLevelRed, CategoryBuildArtifact},
	"coverage":      {NoiseLevelRed, CategoryTestArtifact},
	"__snapshots__": {NoiseLevelRed, CategoryTestArtifact},
	".idea":         {NoiseLevelRed, CategoryIDEOS},
	".vscode":       {NoiseLevelRed, CategoryIDEOS},
}

// noiseFileExts maps extensions to noise info.
var noiseFileExts = map[string]struct {
	level    NoiseLevel
	category NoiseCategory
}{
	".min.js":  {NoiseLevelRed, CategoryBuildArtifact},
	".min.css": {NoiseLevelRed, CategoryBuildArtifact},
	".snap":    {NoiseLevelRed, CategoryTestArtifact},
	".swp":     {NoiseLevelRed, CategoryIDEOS},
	".log":     {NoiseLevelRed, CategoryIDEOS},
	".pb.go":   {NoiseLevelRed, CategoryGenerated},
}

// noiseFileNames maps exact filenames to noise info.
var noiseFileNames = map[string]struct {
	level    NoiseLevel
	category NoiseCategory
}{
	".DS_Store": {NoiseLevelRed, CategoryIDEOS},
}

// testFilePatterns are suffix patterns that indicate test files.
var testFileSuffixes = []string{
	".test.ts", ".test.js", ".test.tsx", ".test.jsx",
	".spec.ts", ".spec.js", ".spec.tsx", ".spec.jsx",
	"_test.go", ".test.py",
}

// binaryExtensions are file extensions that are typically binary.
var binaryExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".ico": true, ".svg": false, // svg is text
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".zip": true, ".tar": true, ".gz": true, ".bz2": true,
	".7z": true, ".rar": true, ".jar": true, ".war": true,
	".pdf": true, ".docx": true, ".xlsx": true, ".pptx": true,
	".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
	".wasm": true, ".bin": true, ".dat": true,
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true,
	".pyc": true, ".class": true, ".o": true, ".a": true,
}

// ClassifyPath classifies a single path (file or dir).
func (c *Classifier) ClassifyPath(path string, isDir bool) Classification {
	base := filepath.Base(path)

	if isDir {
		return c.classifyDir(base)
	}
	return c.classifyFile(path, base)
}

func (c *Classifier) classifyDir(name string) Classification {
	if info, ok := noiseDirNames[name]; ok {
		return Classification{Level: info.level, Category: info.category}
	}
	return Classification{}
}

func (c *Classifier) classifyFile(path, base string) Classification {
	// Check exact filename matches
	if info, ok := noiseFileNames[base]; ok {
		return Classification{Level: info.level, Category: info.category}
	}

	// Check lockfiles
	if lockfileNames[base] {
		return Classification{Level: NoiseLevelRed, Category: CategoryLockfile}
	}

	// Check generated patterns: *.generated.*
	if isGeneratedName(base) {
		return Classification{Level: NoiseLevelRed, Category: CategoryGenerated, Pattern: "*.generated.*"}
	}

	// Check test file suffixes (yellow)
	for _, suffix := range testFileSuffixes {
		if strings.HasSuffix(base, suffix) {
			return Classification{Level: NoiseLevelYellow, Category: CategoryTestFile, Pattern: "*" + suffix}
		}
	}

	// Check binary extensions
	ext := strings.ToLower(filepath.Ext(base))
	if ext != "" {
		// Check compound extensions like .min.js, .pb.go
		lower := strings.ToLower(base)
		for compExt, info := range noiseFileExts {
			if strings.HasSuffix(lower, compExt) {
				return Classification{Level: info.level, Category: info.category, Pattern: "*" + compExt}
			}
		}

		if binary, ok := binaryExtensions[ext]; ok && binary {
			return Classification{Level: NoiseLevelRed, Category: CategoryBinary, Pattern: "*" + ext}
		}
	}

	// Check file content for generated markers
	if isGeneratedContent(path) {
		return Classification{Level: NoiseLevelRed, Category: CategoryGenerated}
	}

	// Check binary content via sniff
	if isBinaryFile(path) {
		return Classification{Level: NoiseLevelRed, Category: CategoryBinary}
	}

	return Classification{}
}

func isGeneratedName(base string) bool {
	lower := strings.ToLower(base)
	// *.generated.* pattern
	parts := strings.Split(lower, ".")
	for i, p := range parts {
		if p == "generated" && i > 0 {
			return true
		}
	}
	return false
}

// isGeneratedContent checks the first few lines of a file for generated markers.
func isGeneratedContent(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lines := 0
	for scanner.Scan() && lines < 5 {
		line := scanner.Text()
		if strings.Contains(line, "Code generated") ||
			strings.Contains(line, "DO NOT EDIT") ||
			strings.Contains(line, "DO NOT MODIFY") ||
			strings.Contains(line, "auto-generated") ||
			strings.Contains(line, "autogenerated") {
			return true
		}
		lines++
	}
	return false
}

// isBinaryFile sniffs the first 512 bytes for null bytes (binary indicator).
func isBinaryFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil || n == 0 {
		return false
	}

	for _, b := range buf[:n] {
		if b == 0 {
			return true
		}
	}
	return false
}
