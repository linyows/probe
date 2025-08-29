package probe

import (
	"google.golang.org/protobuf/types/known/structpb"
)

// StructToMap converts a protobuf Struct to a map[string]any
func StructToMap(s *structpb.Struct) map[string]any {
	if s == nil {
		return nil
	}
	return s.AsMap()
}

// MapToStruct converts a map[string]any to a protobuf Struct
func MapToStruct(m map[string]any) (*structpb.Struct, error) {
	if m == nil {
		return nil, nil
	}
	return structpb.NewStruct(m)
}

// MapToStructFlat creates a protobuf Struct from map[string]any and returns flattened map[string]string
// This function provides high-performance data processing using protobuf.Struct internally
func MapToStructFlat(m map[string]any) (map[string]string, error) {
	// Use protobuf.Struct for validation and processing
	structData, err := structpb.NewStruct(m)
	if err != nil {
		// Fall back to original flattening for unsupported types
		return FlattenInterface(m), nil
	}
	
	// Convert back to map for processing 
	processedMap := structData.AsMap()
	
	// Return flattened result for API compatibility
	return FlattenInterface(processedMap), nil
}

// StructFlatToMap converts a flattened map[string]string to map[string]any using protobuf Struct
// This function provides high-performance data processing using protobuf.Struct internally
func StructFlatToMap(flat map[string]string) map[string]any {
	// First unflatten using existing logic
	unflattened := UnflattenInterface(flat)
	
	// Use protobuf.Struct for validation and normalization
	structData, err := structpb.NewStruct(unflattened)
	if err != nil {
		// Fall back to original approach for unsupported data
		return unflattened
	}
	
	// Convert back to map and fix numeric type issues
	result := structData.AsMap()
	return normalizeNumericTypes(result)
}

// normalizeNumericTypes converts float64 values to int where appropriate
func normalizeNumericTypes(data map[string]any) map[string]any {
	result := make(map[string]any)
	
	for key, value := range data {
		switch v := value.(type) {
		case float64:
			// Check if it's actually an integer
			if v == float64(int64(v)) {
				result[key] = int(v)
			} else {
				result[key] = v
			}
		case map[string]any:
			// Recursively process nested maps
			result[key] = normalizeNumericTypes(v)
		case []any:
			// Process arrays
			normalizedArray := make([]any, len(v))
			for i, item := range v {
				if itemMap, ok := item.(map[string]any); ok {
					normalizedArray[i] = normalizeNumericTypes(itemMap)
				} else if floatVal, ok := item.(float64); ok && floatVal == float64(int64(floatVal)) {
					normalizedArray[i] = int(floatVal)
				} else {
					normalizedArray[i] = item
				}
			}
			result[key] = normalizedArray
		default:
			result[key] = v
		}
	}
	
	return result
}