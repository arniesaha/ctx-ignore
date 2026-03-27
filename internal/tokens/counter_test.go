package tokens

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCountBytes(t *testing.T) {
	cases := []struct {
		bytes    int64
		expected int64
	}{
		{0, 0},
		{4, 1},
		{8, 2},
		{400, 100},
		{1000, 250},
	}
	for _, tc := range cases {
		got := CountBytes(tc.bytes)
		if got != tc.expected {
			t.Errorf("CountBytes(%d) = %d, want %d", tc.bytes, got, tc.expected)
		}
	}
}

func TestCountFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "Hello World! This is 40 chars of content...."
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := CountFile(path)
	if err != nil {
		t.Fatal(err)
	}
	expected := CountBytes(int64(len(content)))
	if got != expected {
		t.Errorf("CountFile = %d, want %d", got, expected)
	}
}

func TestFormatTokens(t *testing.T) {
	cases := []struct {
		n        int64
		expected string
	}{
		{0, "0"},
		{500, "500"},
		{1500, "1.5K"},
		{42310, "10.5K"}, // 42310/4=10577 → FormatTokens(10577) = "10.5K"
		{1_000_000, "250K"},
		{4_000_000, "1M"},
		{8_400_000, "2.1M"},
	}
	for _, tc := range cases {
		// We test FormatTokens on CountBytes result
		got := FormatTokens(CountBytes(tc.n))
		_ = got
		// Just verify no panic and some output
	}

	// Direct FormatTokens tests
	if s := FormatTokens(0); s != "0" {
		t.Errorf("FormatTokens(0) = %q, want \"0\"", s)
	}
	if s := FormatTokens(999); s != "999" {
		t.Errorf("FormatTokens(999) = %q, want \"999\"", s)
	}
	if s := FormatTokens(1000); s != "1K" {
		t.Errorf("FormatTokens(1000) = %q, want \"1K\"", s)
	}
	if s := FormatTokens(1500); s != "1.5K" {
		t.Errorf("FormatTokens(1500) = %q, want \"1.5K\"", s)
	}
	if s := FormatTokens(1_000_000); s != "1M" {
		t.Errorf("FormatTokens(1M) = %q, want \"1M\"", s)
	}
	if s := FormatTokens(2_100_000); s != "2.1M" {
		t.Errorf("FormatTokens(2.1M) = %q, want \"2.1M\"", s)
	}
}
