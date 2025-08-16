package probe

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const (
	flatkey = "__"
	
	// Type prefixes for preserving type information in string encoding
	typePrefixInt   = "#i#"
	typePrefixFloat = "#f#"
	typePrefixBool  = "#b#"
)

// FlattenInterface recursively flattens a nested data structure into a flat map[string]string.
// Nested keys are separated by "__" (double underscore). Arrays and slices use numeric indices.
//
// Example:
//
//	input := map[string]any{
//	  "user": map[string]any{
//	    "name": "John",
//	    "tags": []string{"admin", "user"},
//	  },
//	  "count": 42,
//	}
//
//	result := FlattenInterface(input)
//	// result: map[string]string{
//	//   "user__name": "John",
//	//   "user__tags__0": "admin",
//	//   "user__tags__1": "user",
//	//   "count": "42",
//	// }
func FlattenInterface(i any) map[string]string {
	return flattenIf(i, "")
}

// flattenIf is the internal recursive function used by FlattenInterface.
// It handles the actual flattening logic with prefix accumulation.
func flattenIf(input any, prefix string) map[string]string {
	res := make(map[string]string)

	if input == nil {
		res[prefix] = ""
		return res
	}

	switch reflect.TypeOf(input).Kind() {
	case reflect.Map:
		// Traverse a map to get keys and values
		inputMap := reflect.ValueOf(input)
		for _, key := range inputMap.MapKeys() {
			// Convert the current key to a string
			strKey := fmt.Sprintf("%v", key)

			// Underscore prefixes when nested
			if prefix != "" {
				strKey = prefix + flatkey + strKey
			}

			// Recursive calls handle nesting
			for k, v := range flattenIf(inputMap.MapIndex(key).Interface(), strKey) {
				res[k] = v
			}
		}

	case reflect.Slice, reflect.Array:
		// Working with slices and arrays (using indexes as keys)
		inputSlice := reflect.ValueOf(input)
		for i := 0; i < inputSlice.Len(); i++ {
			strKey := fmt.Sprintf("%d", i)
			if prefix != "" {
				strKey = prefix + flatkey + strKey
			}
			for k, v := range flattenIf(inputSlice.Index(i).Interface(), strKey) {
				res[k] = v
			}
		}

	default:
		// If it is a basic type, encode with type prefix for type preservation
		res[prefix] = encodeValueWithTypePrefix(input)
	}

	return res
}

// UnflattenInterface converts a flattened map[string]string back to a nested map[string]any.
// This is the inverse operation of FlattenInterface. Automatically detects and converts:
// - Maps with sequential numeric keys (0, 1, 2...) to arrays
// - Numeric strings to numbers (int or float64)
// - Nested structures using "__" separator
//
// Example:
//
//	flatMap := map[string]string{
//	  "user__name": "John",
//	  "user__tags__0": "admin",
//	  "user__tags__1": "user",
//	  "count": "42",
//	  "price": "19.99",
//	}
//
//	result := UnflattenInterface(flatMap)
//	// result: map[string]any{
//	//   "user": map[string]any{
//	//     "name": "John",
//	//     "tags": []any{"admin", "user"},
//	//   },
//	//   "count": 42,
//	//   "price": 19.99,
//	// }
//
// Special case: If the root level forms an array, returns {"__array_root": []any{...}}
func UnflattenInterface(flatMap map[string]string) map[string]any {
	result := make(map[string]any)

	for key, value := range flatMap {
		keys := strings.Split(key, flatkey)
		nestMap(result, keys, value)
	}

	// Check if the root result should be converted to an array
	if shouldConvertToArray(result) {
		// Return a wrapper map containing the array since this function returns map[string]any
		arrayResult := convertMapToArrayWithNumericConversion(result)
		return map[string]any{"__array_root": arrayResult}
	}

	return convertMapsToArraysAndNumericStrings(result)
}

// convertMapsToArraysAndNumericStrings combines array conversion and numeric string conversion.
// This internal function is used by UnflattenInterface to apply both transformations recursively.
//
// Performs two main conversions:
// 1. Converts maps with sequential numeric keys (0,1,2...) to arrays
// 2. Converts numeric strings to actual numbers (int or float64)
func convertMapsToArraysAndNumericStrings(input map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range input {
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
			// Check if this nested map should be converted to an array
			if shouldConvertToArray(v) {
				result[key] = convertMapToArrayWithNumericConversion(v)
			} else {
				// Recursively process nested maps
				result[key] = convertMapsToArraysAndNumericStrings(v)
			}
		default:
			result[key] = v
		}
	}

	return result
}

// convertMapToArrayWithNumericConversion converts a map with numeric keys to an array.
// Also applies numeric string conversion to array elements recursively.
//
// Example:
//
//	input := map[string]any{
//	  "0": map[string]any{"name": "item1", "count": "5"},
//	  "1": map[string]any{"0": "nested1", "1": "nested2"},
//	  "2": "42",
//	}
//
//	result := convertMapToArrayWithNumericConversion(input)
//	// result: []any{
//	//   map[string]any{"name": "item1", "count": 5},
//	//   []any{"nested1", "nested2"},
//	//   42,
//	// }
func convertMapToArrayWithNumericConversion(m map[string]any) []any {
	result := make([]any, len(m))

	for key, value := range m {
		index, _ := strconv.Atoi(key)
		switch v := value.(type) {
		case string:
			// Try to convert numeric strings to numbers
			if num, err := strconv.Atoi(v); err == nil {
				result[index] = num
			} else if floatNum, err := strconv.ParseFloat(v, 64); err == nil {
				result[index] = floatNum
			} else {
				result[index] = v
			}
		case map[string]any:
			// Recursively process nested structures
			if shouldConvertToArray(v) {
				result[index] = convertMapToArrayWithNumericConversion(v)
			} else {
				result[index] = convertMapsToArraysAndNumericStrings(v)
			}
		default:
			result[index] = value
		}
	}

	return result
}

// shouldConvertToArray checks if a map should be converted to an array.
// Returns true if all keys are numeric strings forming a complete sequence from 0 to len-1.
//
// Example:
//
//	shouldConvertToArray(map[string]any{"0": "a", "1": "b", "2": "c"}) // true
//	shouldConvertToArray(map[string]any{"0": "a", "2": "c"})           // false (missing 1)
//	shouldConvertToArray(map[string]any{"a": "a", "b": "b"})           // false (non-numeric keys)
func shouldConvertToArray(m map[string]any) bool {
	if len(m) == 0 {
		return false
	}

	// Check if all keys are numeric and form a complete sequence from 0 to len-1
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	return isNumericSequence(keys)
}

// isNumericSequence checks if the keys form a numeric sequence starting from 0.
// Used by shouldConvertToArray to validate array conversion eligibility.
//
// Example:
//
//	isNumericSequence([]string{"0", "1", "2"})    // true
//	isNumericSequence([]string{"2", "0", "1"})    // true (order doesn't matter)
//	isNumericSequence([]string{"0", "2", "3"})    // false (missing 1)
//	isNumericSequence([]string{"a", "b", "c"})    // false (non-numeric)
func isNumericSequence(keys []string) bool {
	if len(keys) == 0 {
		return false
	}

	// Convert all keys to integers and check if they form a sequence
	nums := make([]int, len(keys))
	for i, key := range keys {
		num, err := strconv.Atoi(key)
		if err != nil {
			return false
		}
		nums[i] = num
	}

	// Check if it's a complete sequence from 0 to len-1
	for i := 0; i < len(nums); i++ {
		found := false
		for _, num := range nums {
			if num == i {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// nestMap is a helper function to set values for nested keys in UnflattenInterface.
// Creates nested map structure based on key path and decodes values with type prefixes.
//
// Example:
//
//	m := make(map[string]any)
//	nestMap(m, []string{"user", "profile", "age"}, "#i#30")
//	// m becomes: {"user": {"profile": {"age": 30}}}
func nestMap(m map[string]any, keys []string, value string) {
	if len(keys) == 1 {
		// when it is the last key, set the value with type decoding
		m[keys[0]] = decodeValueWithTypePrefix(value)
	} else {
		// when there are still keys remaining, create the next level map
		if _, exists := m[keys[0]]; !exists {
			m[keys[0]] = make(map[string]any)
		}
		// recursively set the next nested map
		nestMap(m[keys[0]].(map[string]any), keys[1:], value)
	}
}

// ConvertNumericStrings recursively converts numeric strings to numbers in nested structures.
// This function processes maps and converts string values that represent numbers to actual numeric types.
//
// Example:
//
//	input := map[string]any{
//	  "count": "42",
//	  "price": "19.99",
//	  "name": "product",
//	  "nested": map[string]any{
//	    "quantity": "10",
//	    "available": "true",
//	  },
//	}
//
//	result := ConvertNumericStrings(input)
//	// result: map[string]any{
//	//   "count": 42,
//	//   "price": 19.99,
//	//   "name": "product",
//	//   "nested": map[string]any{
//	//     "quantity": 10,
//	//     "available": "true", // non-numeric strings remain unchanged
//	//   },
//	// }
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

// ConvertBodyToJson converts flat body data to properly nested JSON structure.
// This function processes body__ prefixed keys and converts them to a JSON string.
// Supports both object and array structures based on key patterns.
//
// Example:
//
//	data := map[string]string{
//	  "method": "POST",
//	  "body__name": "John",
//	  "body__tags__0": "admin",
//	  "body__tags__1": "user",
//	}
//
//	err := ConvertBodyToJson(data)
//	// data becomes:
//	// {
//	//   "method": "POST",
//	//   "body": `{"name":"John","tags":["admin","user"]}`,
//	// }
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
		// Note: Expression expansion should already be done by this point
		// For HTTP, use legacy numeric conversion for backward compatibility
		unflattenedData := UnflattenInterface(bodyData)
		// Apply numeric conversion for backward compatibility with HTTP actions
		if arrayRoot, ok := unflattenedData["__array_root"]; ok {
			// Handle root array case
			if arrayData, isArray := arrayRoot.([]any); isArray {
				convertedArray := make([]any, len(arrayData))
				for i, item := range arrayData {
					if mapItem, isMap := item.(map[string]any); isMap {
						convertedArray[i] = convertMapsToArraysAndNumericStrings(mapItem)
					} else {
						convertedArray[i] = item
					}
				}
				unflattenedData = map[string]any{"__array_root": convertedArray}
			}
		} else {
			unflattenedData = convertMapsToArraysAndNumericStrings(unflattenedData)
		}

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

// encodeValueWithTypePrefix encodes a value with appropriate type prefix for type preservation.
// This allows accurate type restoration in UnflattenInterface.
//
// Encoding format:
//   - int: "#i#123"
//   - float64: "#f#19.99"
//   - bool: "#b#true" or "#b#false"
//   - string: "value" (no prefix for backward compatibility)
//
// Example:
//
//	encodeValueWithTypePrefix(42)       // "#i#42"
//	encodeValueWithTypePrefix(19.99)    // "#f#19.99"
//	encodeValueWithTypePrefix(true)     // "#b#true"
//	encodeValueWithTypePrefix("hello")  // "hello"
func encodeValueWithTypePrefix(input any) string {
	switch v := input.(type) {
	case int:
		return typePrefixInt + strconv.Itoa(v)
	case int8:
		return typePrefixInt + strconv.Itoa(int(v))
	case int16:
		return typePrefixInt + strconv.Itoa(int(v))
	case int32:
		return typePrefixInt + strconv.Itoa(int(v))
	case int64:
		return typePrefixInt + strconv.FormatInt(v, 10)
	case uint:
		return typePrefixInt + strconv.FormatUint(uint64(v), 10)
	case uint8:
		return typePrefixInt + strconv.FormatUint(uint64(v), 10)
	case uint16:
		return typePrefixInt + strconv.FormatUint(uint64(v), 10)
	case uint32:
		return typePrefixInt + strconv.FormatUint(uint64(v), 10)
	case uint64:
		return typePrefixInt + strconv.FormatUint(v, 10)
	case float32:
		return typePrefixFloat + strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return typePrefixFloat + strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return typePrefixBool + strconv.FormatBool(v)
	case string:
		// No prefix for strings to maintain backward compatibility
		return v
	default:
		// For unknown types, convert to string without prefix
		return fmt.Sprintf("%v", input)
	}
}

// decodeValueWithTypePrefix decodes a string value that may contain type prefix information.
// This is the inverse operation of encodeValueWithTypePrefix.
//
// Decoding format:
//   - "#i#123" → int(123)
//   - "#f#19.99" → float64(19.99)
//   - "#b#true" → bool(true)
//   - "#b#false" → bool(false)
//   - "hello" → string("hello") (no prefix, default string)
//
// Example:
//
//	decodeValueWithTypePrefix("#i#42")      // 42 (int)
//	decodeValueWithTypePrefix("#f#19.99")   // 19.99 (float64)
//	decodeValueWithTypePrefix("#b#true")    // true (bool)
//	decodeValueWithTypePrefix("hello")      // "hello" (string)
func decodeValueWithTypePrefix(value string) any {
	// Check for type prefixes
	if strings.HasPrefix(value, typePrefixInt) {
		valueStr := strings.TrimPrefix(value, typePrefixInt)
		if intValue, err := strconv.Atoi(valueStr); err == nil {
			return intValue
		}
		// If parsing fails, treat as string
		return valueStr
	}
	
	if strings.HasPrefix(value, typePrefixFloat) {
		valueStr := strings.TrimPrefix(value, typePrefixFloat)
		if floatValue, err := strconv.ParseFloat(valueStr, 64); err == nil {
			return floatValue
		}
		// If parsing fails, treat as string
		return valueStr
	}
	
	if strings.HasPrefix(value, typePrefixBool) {
		valueStr := strings.TrimPrefix(value, typePrefixBool)
		if boolValue, err := strconv.ParseBool(valueStr); err == nil {
			return boolValue
		}
		// If parsing fails, treat as string
		return valueStr
	}
	
	// No prefix - default to string (type prefix approach: explicit typing only)
	return value
}





