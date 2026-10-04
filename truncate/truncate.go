// Package truncate shortens long values before they are logged or printed,
// marking where they were cut.
package truncate

import (
	"fmt"

	"github.com/fatih/color"
)

const (
	// MaxLogLength is the maximum length for log output to prevent log bloat
	MaxLogLength = 200
	// MaxLength is the maximum length for general string processing
	MaxLength = 1000000
)

// Message returns the colored marker appended to a truncated value.
func Message() string {
	return "... [" + color.New(color.FgYellow).Sprintf("⚠︎ probe truncated") + "]"
}

// String truncates a string if it exceeds the maximum length
func String(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + Message()
}

// MapString truncates long values in map[string]string for logging
func MapString(params map[string]string, maxLen int) map[string]string {
	truncated := make(map[string]string)
	for key, value := range params {
		truncated[key] = String(value, maxLen)
	}
	return truncated
}

// Map truncates long values in map[string]any for logging
func Map(params map[string]any, maxLen int) map[string]any {
	truncated := make(map[string]any)
	for key, value := range params {
		switch v := value.(type) {
		case string:
			truncated[key] = String(v, maxLen)
		default:
			// For non-string values, convert to string first, then truncate
			str := fmt.Sprintf("%v", v)
			truncated[key] = String(str, maxLen)
		}
	}
	return truncated
}
