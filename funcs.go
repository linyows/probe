package probe

import (
	"fmt"
	"regexp"
	"strings"
)

// MatchJSON compares two `map[string]any` objects strictly.
// All fields in `src` and `target` must match, including structure and values.
func MatchJSON(src, target map[string]any) bool {
	var diffs []string
	return deepMatch(src, target, &diffs, "")
}

// DiffJSON compares two `map[string]any` objects strictly and collects differences.
func DiffJSON(src, target map[string]any) string {
	var diffs []string
	if match := deepMatch(src, target, &diffs, ""); match {
		return "No diff"
	}
	return strings.Join(diffs, "\n")
}

// deepMatch recursively compares `src` and `target`.
func deepMatch(src, target any, diffs *[]string, path string) bool {
	// extendPath constructs a new path for nested keys.
	extendPath := func(path, key string) string {
		if path == "" {
			return key
		}
		return fmt.Sprintf("%s.%s", path, key)
	}

	switch targetVal := target.(type) {
	case map[string]any:
		// Check if src is also a map
		srcMap, ok := src.(map[string]any)
		if !ok {
			*diffs = append(*diffs, fmt.Sprintf("Key '%s': Expected a map, got %T", path, src))
			return false
		}
		// Check for missing or mismatched keys
		for key, targetValue := range targetVal {
			newPath := extendPath(path, key)
			srcValue, exists := srcMap[key]
			if !exists {
				*diffs = append(*diffs, fmt.Sprintf("Key '%s': Missing in source", newPath))
				return false
			}
			if !deepMatch(srcValue, targetValue, diffs, newPath) {
				return false
			}
		}
		// Check for extra keys in src
		for key := range srcMap {
			if _, exists := targetVal[key]; !exists {
				newPath := extendPath(path, key)
				*diffs = append(*diffs, fmt.Sprintf("Key '%s': Extra field in source", newPath))
				return false
			}
		}
		return true

	case []any:
		// Check if src is also a slice
		srcSlice, ok := src.([]any)
		if !ok {
			*diffs = append(*diffs, fmt.Sprintf("Key '%s': Expected a slice, got %T", path, src))
			return false
		}
		if len(srcSlice) != len(targetVal) {
			*diffs = append(*diffs, fmt.Sprintf("Key '%s': Slice length mismatch (expected %d, got %d)", path, len(targetVal), len(srcSlice)))
			return false
		}
		// Recursively compare each element in the slice
		for i, targetElem := range targetVal {
			newPath := fmt.Sprintf("%s[%d]", path, i)
			if !deepMatch(srcSlice[i], targetElem, diffs, newPath) {
				return false
			}
		}
		return true

	case int, int64, uint, uint64, float64:
		targetStr, ok := AnyToString(target)
		if !ok {
			*diffs = append(*diffs, fmt.Sprintf("Key '%s': Expected number, got %T", path, target))
			return false
		}
		srcStr, ok := AnyToString(src)
		if !ok {
			*diffs = append(*diffs, fmt.Sprintf("Key '%s': Expected %#v, got %#v", path, target, src))
			return false
		}
		if srcStr != targetStr {
			*diffs = append(*diffs, fmt.Sprintf("Key '%s': Expected %s, got %s", path, targetStr, srcStr))
			return false
		}
		return true

	case string:
		// If the target value is a regex (e.g., "/regex/")
		if len(targetVal) > 2 && targetVal[0] == '/' && targetVal[len(targetVal)-1] == '/' {
			pattern := targetVal[1 : len(targetVal)-1] // Extract the regex pattern
			re := regexp.MustCompile(pattern)
			srcStr, ok := src.(string)
			if ok {
				if !re.MatchString(srcStr) {
					*diffs = append(*diffs, fmt.Sprintf("Key '%s': Regex mismatch (pattern: %s, value: %v)", path, pattern, srcStr))
					return false
				}
			} else {
				srcStr, ok := AnyToString(src)
				if !ok || !re.MatchString(srcStr) {
					*diffs = append(*diffs, fmt.Sprintf("Key '%s': Regex mismatch (pattern: %s, value: %v)", path, pattern, srcStr))
					return false
				}
			}
			return true
		}
		// Otherwise, compare as a regular string
		srcStr, ok := src.(string)
		if !ok || srcStr != targetVal {
			*diffs = append(*diffs, fmt.Sprintf("Key '%s': Expected '%s', got '%v'", path, targetVal, src))
			return false
		}
		return true

	default:
		// Compare all other types directly
		if src != target {
			*diffs = append(*diffs, fmt.Sprintf("Key '%s': Expected '%v', got '%v'", path, target, src))
			return false
		}
		return true
	}
}

// deepMatchWithDiffs recursively compares `src` and `target` and collects differences.
//
//nolint:unused // Reserved for future use
func deepMatchWithDiffs(src, target interface{}, diffs *[]string, path string) bool {
	// extendPath constructs a new path for nested keys.
	extendPath := func(path, key string) string {
		if path == "" {
			return key
		}
		return fmt.Sprintf("%s.%s", path, key)
	}

	switch targetVal := target.(type) {
	case map[string]any:
		srcMap, ok := src.(map[string]any)
		if !ok {
			*diffs = append(*diffs, fmt.Sprintf("Key '%s': Expected a map, got %T", path, src))
			return false
		}

		// Check for missing or mismatched keys
		for key, targetValue := range targetVal {
			newPath := extendPath(path, key)
			srcValue, exists := srcMap[key]
			if !exists {
				*diffs = append(*diffs, fmt.Sprintf("Key '%s': Missing in source", newPath))
				return false
			}
			if !deepMatchWithDiffs(srcValue, targetValue, diffs, newPath) {
				return false
			}
		}

		// Check for extra keys in src
		for key := range srcMap {
			if _, exists := targetVal[key]; !exists {
				newPath := extendPath(path, key)
				*diffs = append(*diffs, fmt.Sprintf("Key '%s': Extra field in source", newPath))
				return false
			}
		}

		return true

	case []any:
		srcSlice, ok := src.([]any)
		if !ok {
			*diffs = append(*diffs, fmt.Sprintf("Key '%s': Expected a slice, got %T", path, src))
			return false
		}
		if len(srcSlice) != len(targetVal) {
			*diffs = append(*diffs, fmt.Sprintf("Key '%s': Slice length mismatch (expected %d, got %d)", path, len(targetVal), len(srcSlice)))
			return false
		}
		for i := range targetVal {
			newPath := fmt.Sprintf("%s[%d]", path, i)
			if !deepMatchWithDiffs(srcSlice[i], targetVal[i], diffs, newPath) {
				return false
			}
		}
		return true

	case string:
		// If the target is a regex
		if len(targetVal) > 2 && targetVal[0] == '/' && targetVal[len(targetVal)-1] == '/' {
			pattern := targetVal[1 : len(targetVal)-1]
			re := regexp.MustCompile(pattern)
			srcStr, ok := src.(string)
			if !ok || !re.MatchString(srcStr) {
				*diffs = append(*diffs, fmt.Sprintf("Key '%s': Regex mismatch (pattern: %s, value: %v)", path, pattern, src))
				return false
			}
			return true
		}

		// Regular string comparison
		srcStr, ok := src.(string)
		if !ok || srcStr != targetVal {
			*diffs = append(*diffs, fmt.Sprintf("Key '%s': Expected '%s', got '%v'", path, targetVal, src))
			return false
		}
		return true

	default:
		if src != target {
			*diffs = append(*diffs, fmt.Sprintf("Key '%s': Expected '%v', got '%v'", path, target, src))
			return false
		}
		return true
	}
}
