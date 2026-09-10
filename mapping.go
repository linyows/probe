package probe

import (
	"maps"
	"os"
	"strings"
)

// mergeMaps recursively merges two maps of type map[string]any.
// If keys conflict, values from 'over' override those in 'base'.
// Nested maps are merged recursively, preserving data from both maps.
//
// Example:
//
//	base := map[string]any{
//	  "a": 1,
//	  "nested": map[string]any{"x": 1, "y": 2},
//	}
//	over := map[string]any{
//	  "nested": map[string]any{"y": 3, "z": 4},
//	  "c": 5,
//	}
//	result := mergeMaps(base, over)
//	// result: map[string]any{
//	//   "a": 1,
//	//   "nested": map[string]any{"x": 1, "y": 3, "z": 4},
//	//   "c": 5,
//	// }
func mergeMaps(base, over map[string]any) map[string]any {
	merged := make(map[string]any)

	// Copy all entries from base into the result
	maps.Copy(merged, base)

	// Merge entries from over, overriding base's values if keys conflict
	for key, value := range over {
		if existing, ok := merged[key]; ok {
			// If both values are maps, merge them recursively
			if map1Nested, ok1 := existing.(map[string]any); ok1 {
				if map2Nested, ok2 := value.(map[string]any); ok2 {
					merged[key] = mergeMaps(map1Nested, map2Nested)
					continue
				}
			}
		}
		// Otherwise, overwrite the value from over
		merged[key] = value
	}

	return merged
}

// strmapToAnymap converts a map[string]string to map[string]any.
// This is a simple type conversion utility function.
//
// Example:
//
//	input := map[string]string{"name": "John", "age": "30"}
//	result := strmapToAnymap(input)
//	// result: map[string]any{"name": "John", "age": "30"}
func strmapToAnymap(strmap map[string]string) map[string]any {
	anymap := make(map[string]any)
	for k, v := range strmap {
		anymap[k] = v
	}
	return anymap
}

// envMap returns all environment variables as a map[string]string.
// Each environment variable is parsed from "KEY=VALUE" format.
//
// Example:
//
//	env := envMap()
//	// env contains all environment variables like:
//	// {"PATH": "/usr/bin:/bin", "HOME": "/home/user", "USER": "username", ...}
func envMap() map[string]string {
	env := make(map[string]string)
	for _, v := range os.Environ() {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return env
}
