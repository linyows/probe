package probe

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const (
	flatkey       = "__"
	tagMap        = "map"
	tagValidate   = "validate"
	labelRequired = "required"
)

// MergeStringMaps merges two string maps, where values from 'over' override values from 'base'.
// Only string values from 'over' are included; non-string values are ignored.
//
// Example:
//   base := map[string]string{"a": "1", "b": "2"}
//   over := map[string]any{"b": "overridden", "c": "3", "d": 123}
//   result := MergeStringMaps(base, over)
//   // result: map[string]string{"a": "1", "b": "overridden", "c": "3"}
//   // Note: "d": 123 is ignored because it's not a string
func MergeStringMaps(base map[string]string, over map[string]any) map[string]string {
	res := make(map[string]string)

	for k, v := range base {
		res[k] = v
	}

	for k, v := range over {
		if value, ok := v.(string); ok {
			res[k] = value
		}
	}

	return res
}

// MergeMaps recursively merges two maps of type map[string]any.
// If keys conflict, values from 'over' override those in 'base'.
// Nested maps are merged recursively, preserving data from both maps.
//
// Example:
//   base := map[string]any{
//     "a": 1,
//     "nested": map[string]any{"x": 1, "y": 2},
//   }
//   over := map[string]any{
//     "nested": map[string]any{"y": 3, "z": 4},
//     "c": 5,
//   }
//   result := MergeMaps(base, over)
//   // result: map[string]any{
//   //   "a": 1,
//   //   "nested": map[string]any{"x": 1, "y": 3, "z": 4},
//   //   "c": 5,
//   // }
func MergeMaps(base, over map[string]any) map[string]any {
	merged := make(map[string]any)

	// Copy all entries from base into the result
	for key, value := range base {
		merged[key] = value
	}

	// Merge entries from over, overriding base's values if keys conflict
	for key, value := range over {
		if existing, ok := merged[key]; ok {
			// If both values are maps, merge them recursively
			if map1Nested, ok1 := existing.(map[string]any); ok1 {
				if map2Nested, ok2 := value.(map[string]any); ok2 {
					merged[key] = MergeMaps(map1Nested, map2Nested)
					continue
				}
			}
		}
		// Otherwise, overwrite the value from over
		merged[key] = value
	}

	return merged
}

// MapToStructByTags converts a map[string]any to a struct using struct tags.
// Fields are mapped using the "map" tag, and validation is performed using the "validate" tag.
// Supports nested structs, []byte fields, and map[string]string fields.
//
// Example:
//   type User struct {
//     Name     string            `map:"name" validate:"required"`
//     Age      int               `map:"age"`
//     Metadata map[string]string `map:"metadata"`
//   }
//   
//   params := map[string]any{
//     "name": "John",
//     "age": 30,
//     "metadata": map[string]any{"role": "admin", "dept": "IT"},
//   }
//   
//   var user User
//   err := MapToStructByTags(params, &user)
//   // user.Name = "John", user.Age = 30, user.Metadata = {"role": "admin", "dept": "IT"}
func MapToStructByTags(params map[string]any, dest any) error {

	val := reflect.ValueOf(dest).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// get the map tag
		mapTag := fieldType.Tag.Get(tagMap)
		if mapTag == "" {
			continue
		}

		// get the validate tag
		validateTag := fieldType.Tag.Get(tagValidate)

		// when nested struct
		if field.Kind() == reflect.Struct {
			nestedParams, ok := params[mapTag].(map[string]any)
			if !ok && validateTag == labelRequired {
				return fmt.Errorf("required field '%s' is missing or not a map[string]any", mapTag)
			} else if ok {
				// recursively assigning a map
				err := MapToStructByTags(nestedParams, field.Addr().Interface())
				if err != nil {
					return err
				}
			}

			// when the field is a map[string]string
		} else if field.Type() == reflect.TypeOf(map[string]string{}) {
			v, ok := params[mapTag].(map[string]interface{})
			if !ok && validateTag == labelRequired {
				return fmt.Errorf("expected map[string]interface{} for field '%s'", mapTag)
			} else {
				existingMap := field.Interface().(map[string]string)
				mergedMap := MergeStringMaps(existingMap, v)
				field.Set(reflect.ValueOf(mergedMap))
			}

			// when the field is []byte
		} else if field.Type() == reflect.TypeOf([]byte{}) {
			v, ok := params[mapTag].(string)
			if !ok && validateTag == labelRequired {
				return fmt.Errorf("expected string for field '%s' to convert to []byte", mapTag)
			} else {
				field.Set(reflect.ValueOf([]byte(v)))
			}

		} else {
			// get the value corresponding to the key from the map
			if v, ok := params[mapTag]; ok {
				// set a value for a field
				if field.CanSet() {
					field.Set(reflect.ValueOf(v))
				}

				// error when required field is missing
			} else if validateTag == "required" {
				return fmt.Errorf("required field '%s' is missing", mapTag)
			}
		}
	}

	return nil
}

// StructToMapByTags converts a struct to a map[string]any using struct tags.
// Fields are mapped using the "map" tag. Supports nested structs, []byte fields, and map[string]string fields.
// This is the inverse operation of MapToStructByTags.
//
// Example:
//   type User struct {
//     Name     string            `map:"name"`
//     Age      int               `map:"age"`
//     Metadata map[string]string `map:"metadata"`
//   }
//   
//   user := User{
//     Name: "John",
//     Age: 30,
//     Metadata: map[string]string{"role": "admin", "dept": "IT"},
//   }
//   
//   result, err := StructToMapByTags(user)
//   // result: map[string]any{
//   //   "name": "John",
//   //   "age": 30,
//   //   "metadata": map[string]string{"role": "admin", "dept": "IT"},
//   // }
func StructToMapByTags(src any) (map[string]any, error) {
	result := make(map[string]any)

	val := reflect.ValueOf(src)
	typ := reflect.TypeOf(src)

	// for pointers, access the actual value
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// get the map tag
		mapTag := fieldType.Tag.Get(tagMap)
		if mapTag == "" {
			continue
		}

		// when nested struct
		if field.Kind() == reflect.Struct {
			nestedMap, err := StructToMapByTags(field.Interface())
			if err != nil {
				return nil, err
			}
			result[mapTag] = nestedMap

			// when the field is []byte
		} else if field.Type() == reflect.TypeOf([]byte{}) {
			if b, ok := field.Interface().([]byte); ok {
				result[mapTag] = string(b)
			}

		} else if field.Type() == reflect.TypeOf(map[string]string{}) {
			// when the field is a map[string]string
			result[mapTag] = field.Interface()

		} else {
			// when the normal field
			result[mapTag] = field.Interface()
		}
	}

	return result, nil
}

// AssignStruct assigns values from an ActionsParams map to a struct using struct tags.
// Supports string and int fields with validation. Used for legacy action parameter assignment.
//
// Example:
//   type Config struct {
//     Name    string `map:"name" validate:"required"`
//     Timeout int    `map:"timeout"`
//   }
//   
//   params := ActionsParams{"name": "test", "timeout": "30"}
//   var config Config
//   err := AssignStruct(params, &config)
//   // config.Name = "test", config.Timeout = 30
func AssignStruct(pa ActionsParams, st any) error {
	v := reflect.ValueOf(st).Elem()
	t := v.Type()
	e := &ValidationError{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fType := field.Type
		mapKey := field.Tag.Get("map")
		va := field.Tag.Get("validate")
		required := strings.Contains(va, "required")

		if mapKey == "" {
			continue
		}

		value, ok := pa[mapKey]
		if ok {
			switch fType.String() {
			case "string":
				v.Field(i).SetString(value)
			case "int":
				intValue, err := strconv.Atoi(value)
				if err != nil {
					e.AddMessage(fmt.Sprintf("params '%s' can't convert to int: %s", mapKey, err))
				} else {
					v.Field(i).SetInt(int64(intValue))
				}
			default:
				e.AddMessage(fmt.Sprintf("params '%s' not found", mapKey))
			}
		}

		if required && v.Field(i).String() == "" {
			e.AddMessage(fmt.Sprintf("params '%s' is required", mapKey))
		}
	}

	if e.HasError() {
		return e
	}

	return nil
}

// FlattenInterface recursively flattens a nested data structure into a flat map[string]string.
// Nested keys are separated by "__" (double underscore). Arrays and slices use numeric indices.
//
// Example:
//   input := map[string]any{
//     "user": map[string]any{
//       "name": "John",
//       "tags": []string{"admin", "user"},
//     },
//     "count": 42,
//   }
//   
//   result := FlattenInterface(input)
//   // result: map[string]string{
//   //   "user__name": "John",
//   //   "user__tags__0": "admin",
//   //   "user__tags__1": "user",
//   //   "count": "42",
//   // }
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
		// If it is a basic type, it is stored as is.
		res[prefix] = fmt.Sprintf("%v", input)
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
//   flatMap := map[string]string{
//     "user__name": "John",
//     "user__tags__0": "admin", 
//     "user__tags__1": "user",
//     "count": "42",
//     "price": "19.99",
//   }
//   
//   result := UnflattenInterface(flatMap)
//   // result: map[string]any{
//   //   "user": map[string]any{
//   //     "name": "John",
//   //     "tags": []any{"admin", "user"},
//   //   },
//   //   "count": 42,
//   //   "price": 19.99,
//   // }
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
//   input := map[string]any{
//     "0": map[string]any{"name": "item1", "count": "5"},
//     "1": map[string]any{"0": "nested1", "1": "nested2"},
//     "2": "42",
//   }
//   
//   result := convertMapToArrayWithNumericConversion(input)
//   // result: []any{
//   //   map[string]any{"name": "item1", "count": 5},
//   //   []any{"nested1", "nested2"},
//   //   42,
//   // }
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
//   shouldConvertToArray(map[string]any{"0": "a", "1": "b", "2": "c"}) // true
//   shouldConvertToArray(map[string]any{"0": "a", "2": "c"})           // false (missing 1)
//   shouldConvertToArray(map[string]any{"a": "a", "b": "b"})           // false (non-numeric keys)
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
//   isNumericSequence([]string{"0", "1", "2"})    // true
//   isNumericSequence([]string{"2", "0", "1"})    // true (order doesn't matter)
//   isNumericSequence([]string{"0", "2", "3"})    // false (missing 1)
//   isNumericSequence([]string{"a", "b", "c"})    // false (non-numeric)
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
// Creates nested map structure based on key path and converts numeric strings to integers.
//
// Example:
//   m := make(map[string]any)
//   nestMap(m, []string{"user", "profile", "age"}, "30")
//   // m becomes: {"user": {"profile": {"age": 30}}}
func nestMap(m map[string]any, keys []string, value string) {
	if len(keys) == 1 {
		// when it is the last key, set the value
		if intValue, err := strconv.Atoi(value); err == nil {
			m[keys[0]] = intValue
		} else {
			m[keys[0]] = value
		}
	} else {
		// when there are still keys remaining, create the next level map
		if _, exists := m[keys[0]]; !exists {
			m[keys[0]] = make(map[string]any)
		}
		// recursively set the next nested map
		nestMap(m[keys[0]].(map[string]any), keys[1:], value)
	}
}

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

// TitleCase converts a string to title case using a specified separator character.
// Each part separated by the character has its first letter capitalized.
//
// Example:
//   TitleCase("content-type", "-")     // "Content-Type"
//   TitleCase("user_name", "_")        // "User_Name"
//   TitleCase("hello-world-test", "-") // "Hello-World-Test"
func TitleCase(st string, char string) string {
	parts := strings.Split(st, char)
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, char)
}

// StrmapToAnymap converts a map[string]string to map[string]any.
// This is a simple type conversion utility function.
//
// Example:
//   input := map[string]string{"name": "John", "age": "30"}
//   result := StrmapToAnymap(input)
//   // result: map[string]any{"name": "John", "age": "30"}
func StrmapToAnymap(strmap map[string]string) map[string]any {
	anymap := make(map[string]any)
	for k, v := range strmap {
		anymap[k] = v
	}
	return anymap
}

// EnvMap returns all environment variables as a map[string]string.
// Each environment variable is parsed from "KEY=VALUE" format.
//
// Example:
//   env := EnvMap()
//   // env contains all environment variables like:
//   // {"PATH": "/usr/bin:/bin", "HOME": "/home/user", "USER": "username", ...}
func EnvMap() map[string]string {
	env := make(map[string]string)
	for _, v := range os.Environ() {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return env
}

// AnyToString attempts to convert any type to a string.
func AnyToString(value any) (string, bool) {
	if value == nil {
		return "nil", true
	}
	
	switch v := value.(type) {
	case string:
		return v, true
	case bool:
		return strconv.FormatBool(v), true
	case int, int8, int16, int32, int64:
		return strconv.FormatInt(reflect.ValueOf(v).Int(), 10), true
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(reflect.ValueOf(v).Uint(), 10), true
	case float32, float64:
		return strconv.FormatFloat(reflect.ValueOf(v).Float(), 'f', -1, 64), true
	case []byte:
		return string(v), true
	case fmt.Stringer:
		return v.String(), true
	default:
		if reflect.ValueOf(value).IsZero() {
			return "nil", true
		}
		return "", false
	}
}

// HeaderToStringValue converts header values to strings for HTTP processing.
// This function ensures all header values are strings, converting numbers and other types as needed.
//
// Example:
//   data := map[string]any{
//     "headers": map[string]any{
//       "Content-Length": 1024,
//       "X-Rate-Limit": 100.5,
//       "Authorization": "Bearer token",
//     },
//   }
//   
//   result := HeaderToStringValue(data)
//   // result["headers"] = map[string]any{
//   //   "Content-Length": "1024",
//   //   "X-Rate-Limit": "100.5", 
//   //   "Authorization": "Bearer token",
//   // }
func HeaderToStringValue(data map[string]any) map[string]any {
	v, exists := data["headers"]
	if !exists {
		return data
	}

	newHeaders := make(map[string]any)
	if headers, ok := v.(map[string]any); ok {
		for key, value := range headers {
			switch v := value.(type) {
			case string:
				newHeaders[key] = v
			case int:
				newHeaders[key] = strconv.Itoa(v)
			case float64:
				newHeaders[key] = strconv.FormatFloat(v, 'f', -1, 64)
			default:
				newHeaders[key] = fmt.Sprintf("%v", v)
			}
		}
	}

	if len(newHeaders) > 0 {
		data["headers"] = newHeaders
	}

	return data
}

// ConvertBodyToJson converts flat body data to properly nested JSON structure.
// This function processes body__ prefixed keys and converts them to a JSON string.
// Supports both object and array structures based on key patterns.
//
// Example:
//   data := map[string]string{
//     "method": "POST",
//     "body__name": "John",
//     "body__tags__0": "admin",
//     "body__tags__1": "user",
//   }
//   
//   err := ConvertBodyToJson(data)
//   // data becomes:
//   // {
//   //   "method": "POST",
//   //   "body": `{"name":"John","tags":["admin","user"]}`,
//   // }
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
		// UnflattenInterface now handles both array conversion and numeric conversion
		unflattenedData := UnflattenInterface(bodyData)

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

// ConvertNumericStrings recursively converts numeric strings to numbers in nested structures.
// This function processes maps and converts string values that represent numbers to actual numeric types.
//
// Example:
//   input := map[string]any{
//     "count": "42",
//     "price": "19.99", 
//     "name": "product",
//     "nested": map[string]any{
//       "quantity": "10",
//       "available": "true",
//     },
//   }
//   
//   result := ConvertNumericStrings(input)
//   // result: map[string]any{
//   //   "count": 42,
//   //   "price": 19.99,
//   //   "name": "product",
//   //   "nested": map[string]any{
//   //     "quantity": 10,
//   //     "available": "true", // non-numeric strings remain unchanged
//   //   },
//   // }
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
