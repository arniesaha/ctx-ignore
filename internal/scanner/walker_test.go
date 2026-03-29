package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestTree creates a temp directory with known noise and signal files.
func setupTestTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// Signal files
	writeFile(t, root, "main.go", "package main\nfunc main() {}\n")
	writeFile(t, root, "README.md", "# Test project\n")
	writeFile(t, root, "src/handler.go", "package src\nfunc Handle() {}\n")

	// Lockfile (noise)
	writeFile(t, root, "go.sum", "github.com/foo/bar v1.0.0 h1:abc=\n")

	// Test files (yellow noise)
	writeFile(t, root, "main_test.go", "package main\nfunc TestMain() {}\n")
	writeFile(t, root, "src/handler_test.go", "package src\nfunc TestHandle() {}\n")

	// Dependency dir (red noise)
	writeFile(t, root, "vendor/module/lib.go", "package module\n")

	// Build dir (red noise)
	writeFile(t, root, "dist/bundle.js", "var x = 1;\n")

	return root
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestScanBasicTree(t *testing.T) {
	root := setupTestTree(t)
	s := New(root)
	result, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	// Should have signal files: main.go, README.md, src/handler.go
	if len(result.SignalItems) != 3 {
		t.Errorf("expected 3 signal items, got %d", len(result.SignalItems))
		for _, si := range result.SignalItems {
			t.Logf("  signal: %s", si.Path)
		}
	}

	// Should have noise patterns for: go.sum, *_test.go, vendor/, dist/
	if len(result.NoisePatterns) < 3 {
		t.Errorf("expected at least 3 noise patterns, got %d", len(result.NoisePatterns))
		for _, np := range result.NoisePatterns {
			t.Logf("  noise: %s (count=%d)", np.Pattern, np.Count)
		}
	}

	// Check that test files are aggregated under *_test.go pattern
	foundTestPattern := false
	for _, np := range result.NoisePatterns {
		if np.Pattern == "*_test.go" {
			foundTestPattern = true
			if np.Count != 2 {
				t.Errorf("expected 2 test files aggregated, got %d", np.Count)
			}
			if np.Level != NoiseLevelYellow {
				t.Errorf("test files should be yellow, got %d", np.Level)
			}
		}
	}
	if !foundTestPattern {
		t.Error("expected *_test.go noise pattern")
	}

	// Check vendor/ is detected
	foundVendor := false
	for _, np := range result.NoisePatterns {
		if np.Pattern == "vendor/" {
			foundVendor = true
			if np.Category != CategoryDependency {
				t.Errorf("vendor/ should be CategoryDependency, got %q", np.Category)
			}
		}
	}
	if !foundVendor {
		t.Error("expected vendor/ noise pattern")
	}
}

func TestScanTotalTokensPositive(t *testing.T) {
	root := setupTestTree(t)
	s := New(root)
	result, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if result.TotalTokens <= 0 {
		t.Errorf("expected positive total tokens, got %d", result.TotalTokens)
	}

	// Total tokens should equal signal + noise
	var signalTokens int64
	for _, si := range result.SignalItems {
		signalTokens += si.Tokens
	}
	var noiseTokens int64
	for _, np := range result.NoisePatterns {
		noiseTokens += np.Tokens
	}
	if signalTokens+noiseTokens != result.TotalTokens {
		t.Errorf("signal(%d) + noise(%d) = %d, but TotalTokens = %d",
			signalTokens, noiseTokens, signalTokens+noiseTokens, result.TotalTokens)
	}
}

func TestScanNoisePatternsAreSorted(t *testing.T) {
	root := setupTestTree(t)
	s := New(root)
	result, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	// Red patterns should come before yellow
	seenYellow := false
	for _, np := range result.NoisePatterns {
		if np.Level == NoiseLevelYellow {
			seenYellow = true
		}
		if seenYellow && np.Level == NoiseLevelRed {
			t.Error("red patterns should come before yellow patterns in sorted output")
			break
		}
	}
}

func TestScanEmptyDir(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	result, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if result.TotalFiles != 0 {
		t.Errorf("expected 0 files, got %d", result.TotalFiles)
	}
	if len(result.NoisePatterns) != 0 {
		t.Errorf("expected 0 noise patterns, got %d", len(result.NoisePatterns))
	}
}
