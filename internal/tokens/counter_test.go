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
		{999, "999"},
		{1000, "1K"},
		{1500, "1.5K"},
		{10577, "10.5K"},
		{1_000_000, "1M"},
		{2_100_000, "2.1M"},
	}
	for _, tc := range cases {
		got := FormatTokens(tc.n)
		if got != tc.expected {
			t.Errorf("FormatTokens(%d) = %q, want %q", tc.n, got, tc.expected)
		}
	}
}

func TestFormatTokensViaCountBytes(t *testing.T) {
	cases := []struct {
		bytes    int64
		expected string
	}{
		{0, "0"},
		{500 * 4, "500"},
		{4_000_000, "1M"},
		{8_400_000, "2.1M"},
	}
	for _, tc := range cases {
		got := FormatTokens(CountBytes(tc.bytes))
		if got != tc.expected {
			t.Errorf("FormatTokens(CountBytes(%d)) = %q, want %q", tc.bytes, got, tc.expected)
		}
	}
}
