// Package tokens provides token counting utilities.
// For v0.1, we use a character-based approximation: 1 token ≈ 4 chars.
package tokens

import "os"

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
		return formatFloat(float64(n)/1_000_000) + "M"
	case n >= 1_000:
		return formatFloat(float64(n)/1_000) + "K"
	default:
		return formatInt(n)
	}
}

func formatFloat(f float64) string {
	// Simple formatting: up to 1 decimal place
	i := int64(f)
	d := int64((f - float64(i)) * 10)
	if d == 0 {
		return formatInt(i)
	}
	return formatInt(i) + "." + formatInt(d)
}

func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
