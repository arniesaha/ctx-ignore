// Package tokens provides token counting utilities.
// For v0.1, we use a character-based approximation: 1 token ≈ 4 chars.
package tokens

import (
	"fmt"
	"math"
	"os"
	"strconv"
)

const charsPerToken = 4

// CountFile returns the approximate token count for a file.
func CountFile(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return CountBytes(info.Size()), nil
}

// CountBytes returns the approximate token count for a byte count.
func CountBytes(bytes int64) int64 {
	if bytes == 0 {
		return 0
	}
	tokens := bytes / charsPerToken
	if tokens == 0 {
		return 1
	}
	return tokens
}

// FormatTokens returns a human-readable token count string.
func FormatTokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return formatCompact(float64(n)/1_000_000) + "M"
	case n >= 1_000:
		return formatCompact(float64(n)/1_000) + "K"
	default:
		return strconv.FormatInt(n, 10)
	}
}

// formatCompact formats a float with up to 1 decimal place, trimming ".0".
func formatCompact(f float64) string {
	// Truncate to 1 decimal place
	f = math.Floor(f*10) / 10
	if f == math.Floor(f) {
		return strconv.FormatInt(int64(f), 10)
	}
	return fmt.Sprintf("%.1f", f)
}
