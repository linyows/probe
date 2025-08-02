package probe

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	flatkey       = "__"
	tagMap        = "map"
	tagValidate   = "validate"
	labelRequired = "required"
)

// mustMarshalJSON attempts to unmarshal a JSON string into map[string]any.
// If unmarshaling fails, returns a map with an error message instead of panicking.
//
// Example:
//   result := mustMarshalJSON(`{"name": "John", "age": 30}`)
//   // result: map[string]any{"name": "John", "age": 30}
//   
//   result := mustMarshalJSON(`invalid json`)
//   // result: map[string]any{"error_message": "mustMarshalJSON error: ..."}
func mustMarshalJSON(st string) map[string]any {
	var re map[string]any
	if err := json.Unmarshal([]byte(st), &re); err != nil {
		return map[string]any{
			"error_message": fmt.Sprintf("mustMarshalJSON error: %s", err),
		}
	}
	return re
}

// isJSON checks if a string appears to be JSON by examining its first and last characters.
// This is a simple heuristic check and does not validate actual JSON syntax.
//
// Example:
//   isJSON(`{"key": "value"}`)  // true
//   isJSON(`["item1", "item2"]`) // true
//   isJSON(`{key: value}`)       // true (note: this is actually invalid JSON but has JSON-like brackets)
//   isJSON(`hello world`)        // false
func isJSON(st string) bool {
	trimmed := strings.TrimSpace(st)
	if len(trimmed) < 2 {
		return false
	}

	fChar := rune(trimmed[0])
	lChar := rune(trimmed[len(trimmed)-1])

	return (fChar == '{' && lChar == '}') || (fChar == '[' && lChar == ']')
}