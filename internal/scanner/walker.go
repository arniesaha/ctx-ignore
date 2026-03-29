package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/arniesaha/ctx-ignore/internal/tokens"
)

// ScanResult is the full result of scanning a directory.
type ScanResult struct {
	Root        string
	TotalFiles  int
	TotalTokens int64
	NoiseItems  []NoiseItem
	SignalItems []SignalItem
	// Aggregated by pattern for noise
	NoisePatterns []NoisePattern
}

// NoiseItem is a single noisy file.
type NoiseItem struct {
	Path     string
	Tokens   int64
	Level    NoiseLevel
	Category NoiseCategory
}

// SignalItem is a file worth keeping.
type SignalItem struct {
	Path   string
	Tokens int64
}

// NoisePattern is an aggregated noise pattern for the report.
type NoisePattern struct {
	Pattern  string
	Tokens   int64
	Count    int
	Level    NoiseLevel
	Category NoiseCategory
}

// Scanner walks a directory and classifies files.
type Scanner struct {
	root       string
	classifier *Classifier
}

// New creates a new Scanner.
func New(root string) *Scanner {
	return &Scanner{
		root:       root,
		classifier: NewClassifier(),
	}
}

// Scan walks the root directory and returns a ScanResult.
func (s *Scanner) Scan() (*ScanResult, error) {
	result := &ScanResult{Root: s.root}

	// Track which dirs are noise so we can skip their contents
	noiseDirs := map[string]bool{}

	var noiseItems []NoiseItem
	signalMap := map[string]*SignalItem{}

	// patternMap aggregates noise by pattern/dir
	type patternKey struct {
		pattern  string
		level    NoiseLevel
		category NoiseCategory
	}
	patternMap := map[patternKey]*NoisePattern{}

	addNoiseDir := func(path, base string, cl Classification) {
		dirTokens := countDirTokens(path)
		noiseDirs[path] = true
		key := patternKey{base + "/", cl.Level, cl.Category}
		if p, ok := patternMap[key]; ok {
			p.Tokens += dirTokens
			p.Count++
		} else {
			patternMap[key] = &NoisePattern{
				Pattern:  base + "/",
				Tokens:   dirTokens,
				Count:    1,
				Level:    cl.Level,
				Category: cl.Category,
			}
		}
		result.TotalTokens += dirTokens
		result.TotalFiles++
	}

	err := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}

		// Skip hidden dirs (except .idea, .vscode which we handle explicitly)
		base := filepath.Base(path)
		if d.IsDir() && strings.HasPrefix(base, ".") {
			cl := s.classifier.ClassifyPath(path, true)
			if cl.Level == NoiseLevelNone {
				return filepath.SkipDir // skip other hidden dirs
			}
			addNoiseDir(path, base, cl)
			return filepath.SkipDir
		}

		// Check if this path is inside a noise dir
		for noiseDir := range noiseDirs {
			if strings.HasPrefix(path, noiseDir+string(os.PathSeparator)) {
				return nil
			}
		}

		if d.IsDir() {
			if path == s.root {
				return nil
			}
			cl := s.classifier.ClassifyPath(path, true)
			if cl.Level > NoiseLevelNone {
				addNoiseDir(path, base, cl)
				return filepath.SkipDir
			}
			return nil
		}

		// It's a file
		result.TotalFiles++
		fileTokens, _ := tokens.CountFile(path)
		result.TotalTokens += fileTokens

		cl := s.classifier.ClassifyPath(path, false)
		if cl.Level > NoiseLevelNone {
			noiseItems = append(noiseItems, NoiseItem{
				Path:     path,
				Tokens:   fileTokens,
				Level:    cl.Level,
				Category: cl.Category,
			})
			// Use glob pattern for aggregation when available, otherwise filename
			patternStr := base
			if cl.Pattern != "" {
				patternStr = cl.Pattern
			}
			key := patternKey{patternStr, cl.Level, cl.Category}
			if p, ok := patternMap[key]; ok {
				p.Tokens += fileTokens
				p.Count++
			} else {
				patternMap[key] = &NoisePattern{
					Pattern:  patternStr,
					Tokens:   fileTokens,
					Count:    1,
					Level:    cl.Level,
					Category: cl.Category,
				}
			}
		} else {
			rel, _ := filepath.Rel(s.root, path)
			signalMap[rel] = &SignalItem{Path: rel, Tokens: fileTokens}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	result.NoiseItems = noiseItems
	for _, v := range signalMap {
		result.SignalItems = append(result.SignalItems, *v)
	}

	// Convert patternMap to slice, sort by tokens desc
	for _, p := range patternMap {
		result.NoisePatterns = append(result.NoisePatterns, *p)
	}
	sort.Slice(result.NoisePatterns, func(i, j int) bool {
		if result.NoisePatterns[i].Level != result.NoisePatterns[j].Level {
			return result.NoisePatterns[i].Level > result.NoisePatterns[j].Level
		}
		return result.NoisePatterns[i].Tokens > result.NoisePatterns[j].Tokens
	})

	sort.Slice(result.SignalItems, func(i, j int) bool {
		return result.SignalItems[i].Tokens > result.SignalItems[j].Tokens
	})

	return result, nil
}

// countDirTokens sums token counts for all files in a directory tree.
func countDirTokens(dir string) int64 {
	var total int64
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		t, err := tokens.CountFile(path)
		if err == nil {
			total += t
		}
		return nil
	})
	return total
}
