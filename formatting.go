package probe

import (
	"encoding/json"
	"fmt"
	"strings"
)

// mustMarshalJSON attempts to unmarshal a JSON string into either a
// map[string]any (object input) or []any (array input). On failure it
// returns a map[string]any with an error_message instead of panicking
// so callers can surface the parse error.
//
// (The "Marshal" in the name is a legacy misnomer: this is an Unmarshal
// helper. Renaming is intentionally left for a follow-up to keep this
// change focused on the array-parsing fix.)
//
// Example:
//
//	result := mustMarshalJSON(`{"name": "John"}`)
//	// result: map[string]any{"name": "John"}
//
//	result := mustMarshalJSON(`[{"id": 1}, {"id": 2}]`)
//	// result: []any{map[string]any{"id": float64(1)}, map[string]any{"id": float64(2)}}
//
//	result := mustMarshalJSON(`invalid json`)
//	// result: map[string]any{"error_message": "mustMarshalJSON error: ..."}
func mustMarshalJSON(st string) any {
	// Pick the target type from the first non-space byte so callers
	// that handed us an array body don't get their data replaced by
	// the unmarshal error from the (object-only) default branch.
	trimmed := strings.TrimLeft(st, " \t\r\n")
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var arr []any
		if err := json.Unmarshal([]byte(st), &arr); err != nil {
			return map[string]any{
				"error_message": fmt.Sprintf("mustMarshalJSON error: %s", err),
			}
		}
		return arr
	}

	var obj map[string]any
	if err := json.Unmarshal([]byte(st), &obj); err != nil {
		return map[string]any{
			"error_message": fmt.Sprintf("mustMarshalJSON error: %s", err),
		}
	}
	return obj
}

// isJSON checks if a string appears to be JSON by examining its first and last characters.
// This is a simple heuristic check and does not validate actual JSON syntax.
//
// Example:
//
//	isJSON(`{"key": "value"}`)  // true
//	isJSON(`["item1", "item2"]`) // true
//	isJSON(`{key: value}`)       // true (note: this is actually invalid JSON but has JSON-like brackets)
//	isJSON(`hello world`)        // false
func isJSON(st string) bool {
	trimmed := strings.TrimSpace(st)
	if len(trimmed) < 2 {
		return false
	}

	fChar := rune(trimmed[0])
	lChar := rune(trimmed[len(trimmed)-1])

	return (fChar == '{' && lChar == '}') || (fChar == '[' && lChar == ']')
}
