package probe

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	
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
	
	// Return flattened result for API compatibility using direct implementation
	return flattenMap(processedMap, ""), nil
}

// StructFlatToMap converts a flattened map[string]string to map[string]any using protobuf Struct
// This function provides high-performance data processing using protobuf.Struct internally
func StructFlatToMap(flat map[string]string) map[string]any {
	// First unflatten using direct implementation
	unflattened := unflattenMap(flat)
	
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

// Internal flattening implementation (copied from flattening.go)
const flatkey = "__"

// FlattenInterface provides backward compatibility with the old API
func FlattenInterface(input any) map[string]string {
	return flattenMap(input, "")
}

// UnflattenInterface provides backward compatibility with the old API  
func UnflattenInterface(flatMap map[string]string) map[string]any {
	return unflattenMap(flatMap)
}

func flattenMap(input any, prefix string) map[string]string {
	res := make(map[string]string)

	if input == nil {
		res[prefix] = ""
		return res
	}

	switch reflect.TypeOf(input).Kind() {
	case reflect.Map:
		inputMap := reflect.ValueOf(input)
		for _, key := range inputMap.MapKeys() {
			strKey := fmt.Sprintf("%v", key)
			if prefix != "" {
				strKey = prefix + flatkey + strKey
			}
			for k, v := range flattenMap(inputMap.MapIndex(key).Interface(), strKey) {
				res[k] = v
			}
		}
	case reflect.Slice, reflect.Array:
		inputSlice := reflect.ValueOf(input)
		for i := 0; i < inputSlice.Len(); i++ {
			strKey := fmt.Sprintf("%d", i)
			if prefix != "" {
				strKey = prefix + flatkey + strKey
			}
			for k, v := range flattenMap(inputSlice.Index(i).Interface(), strKey) {
				res[k] = v
			}
		}
	default:
		res[prefix] = fmt.Sprintf("%v", input)
	}
	return res
}

func unflattenMap(flatMap map[string]string) map[string]any {
	result := make(map[string]any)

	for key, value := range flatMap {
		keys := strings.Split(key, flatkey)
		nestMapValue(result, keys, value)
	}

	converted := convertMapsToArraysAndNumericStrings(result)
	
	// Check if the root level forms an array
	if shouldConvertToArray(converted) {
		// Return a wrapper map containing the array
		arrayResult := convertMapToArray(converted)
		return map[string]any{"__array_root": arrayResult}
	}

	return converted
}

func nestMapValue(m map[string]any, keys []string, value string) {
	if len(keys) == 1 {
		// Convert numeric strings
		if num, err := strconv.Atoi(value); err == nil {
			m[keys[0]] = num
		} else if floatNum, err := strconv.ParseFloat(value, 64); err == nil {
			m[keys[0]] = floatNum
		} else {
			m[keys[0]] = value
		}
	} else {
		if _, exists := m[keys[0]]; !exists {
			m[keys[0]] = make(map[string]any)
		}
		nestMapValue(m[keys[0]].(map[string]any), keys[1:], value)
	}
}

func convertMapsToArraysAndNumericStrings(input map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range input {
		switch v := value.(type) {
		case map[string]any:
			if shouldConvertToArray(v) {
				result[key] = convertMapToArray(v)
			} else {
				result[key] = convertMapsToArraysAndNumericStrings(v)
			}
		default:
			result[key] = v
		}
	}
	return result
}

func shouldConvertToArray(m map[string]any) bool {
	if len(m) == 0 {
		return false
	}
	for i := 0; i < len(m); i++ {
		if _, exists := m[strconv.Itoa(i)]; !exists {
			return false
		}
	}
	return true
}

func convertMapToArray(m map[string]any) []any {
	result := make([]any, len(m))
	for key, value := range m {
		index, _ := strconv.Atoi(key)
		if mapVal, ok := value.(map[string]any); ok {
			if shouldConvertToArray(mapVal) {
				result[index] = convertMapToArray(mapVal)
			} else {
				result[index] = convertMapsToArraysAndNumericStrings(mapVal)
			}
		} else {
			result[index] = value
		}
	}
	return result
}

// ConvertBodyToJson provides backward compatibility for body processing
func ConvertBodyToJson(data map[string]string) error {
	bodyData := map[string]string{}

	// Extract all body__ prefixed keys
	for key, value := range data {
		if strings.HasPrefix(key, "body__") {
			newKey := strings.TrimPrefix(key, "body__")
			bodyData[newKey] = value
			delete(data, key)
		}
	}

	if len(bodyData) > 0 {
		// Use new unflatten implementation
		unflattenedData := unflattenMap(bodyData)
		
		// Check if the result is a root array (indicated by __array_root key)
		var dataToMarshal any = unflattenedData
		if arrayRoot, ok := unflattenedData["__array_root"]; ok {
			dataToMarshal = arrayRoot
		}

		j, err := json.Marshal(dataToMarshal)
		if err != nil {
			return err
		}
		data["body"] = string(j)
	}

	return nil
}

// ConvertNumericStrings provides backward compatibility for numeric conversion
func ConvertNumericStrings(data map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range data {
		switch v := value.(type) {
		case string:
			// Try to convert numeric strings to numbers
			if num, err := strconv.Atoi(v); err == nil {
				result[key] = num
			} else if floatNum, err := strconv.ParseFloat(v, 64); err == nil {
				result[key] = floatNum
			} else {
				result[key] = v
			}
		case map[string]any:
			// Recursively process nested maps
			result[key] = ConvertNumericStrings(v)
		default:
			result[key] = v
		}
	}

	return result
}