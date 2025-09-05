package probe

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

const (
	tagMap        = "map"
	tagValidate   = "validate"
	labelRequired = "required"
)

// MergeStringMaps merges two string maps, where values from 'over' override values from 'base'.
// Only string values from 'over' are included; non-string values are ignored.
//
// Example:
//
//	base := map[string]string{"a": "1", "b": "2"}
//	over := map[string]any{"b": "overridden", "c": "3", "d": 123}
//	result := MergeStringMaps(base, over)
//	// result: map[string]string{"a": "1", "b": "overridden", "c": "3"}
//	// Note: "d": 123 is ignored because it's not a string
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
//
//	base := map[string]any{
//	  "a": 1,
//	  "nested": map[string]any{"x": 1, "y": 2},
//	}
//	over := map[string]any{
//	  "nested": map[string]any{"y": 3, "z": 4},
//	  "c": 5,
//	}
//	result := MergeMaps(base, over)
//	// result: map[string]any{
//	//   "a": 1,
//	//   "nested": map[string]any{"x": 1, "y": 3, "z": 4},
//	//   "c": 5,
//	// }
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

func ToAnySlice[T any](s []T) []any {
	res := make([]any, len(s))
	for i, v := range s {
		res[i] = v
	}
	return res
}

func FromAnySlice[T any](s []any) ([]T, error) {
	res := make([]T, len(s))
	for i, v := range s {
		val, ok := v.(T)
		if !ok {
			return nil, fmt.Errorf("element %d has type %T, expected %T", i, v, *new(T))
		}
		res[i] = val
	}
	return res, nil
}

// MapToStructByTags converts a map[string]any to a struct using struct tags.
// Fields are mapped using the "map" tag, and validation is performed using the "validate" tag.
// Supports nested structs, []byte fields, and map[string]string fields.
//
// Example:
//
//	type User struct {
//	  Name     string            `map:"name" validate:"required"`
//	  Age      int               `map:"age"`
//	  Metadata map[string]string `map:"metadata"`
//	}
//
//	params := map[string]any{
//	  "name": "John",
//	  "age": 30,
//	  "metadata": map[string]any{"role": "admin", "dept": "IT"},
//	}
//
//	var user User
//	err := MapToStructByTags(params, &user)
//	// user.Name = "John", user.Age = 30, user.Metadata = {"role": "admin", "dept": "IT"}
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

			// when the field is []???(slice)
		} else if field.Kind() == reflect.Slice {
			v, ok := params[mapTag].([]string)
			if !ok && validateTag == labelRequired {
				return fmt.Errorf("required field '%s' is missing or not a []string", mapTag)
			} else if fieldType.Type.Elem().Kind() == reflect.String {
				if ok {
					// []string ===> []string
					field.Set(reflect.ValueOf(v))

					// []any ===> []string
				} else if params[mapTag] != nil && reflect.TypeOf(params[mapTag]).String() == "[]interface {}" {
					vv, okk := params[mapTag].([]any)
					if !okk && validateTag == labelRequired {
						return fmt.Errorf("required field '%s' is missing or not a []string", mapTag)
					} else {
						stSlice, err := FromAnySlice[string](vv)
						if err != nil {
							return err
						}
						field.Set(reflect.ValueOf(stSlice))
					}
				}

			} else if fieldType.Type.Elem().Kind() == reflect.Struct || fieldType.Type.Elem().Kind() == reflect.Ptr {
				sliceParams, ok := params[mapTag].([]any)
				if !ok && validateTag == labelRequired {
					return fmt.Errorf("required field '%s' is missing or not a map[string]any", mapTag)
				} else if ok {
					elemType := field.Type().Elem()
					for _, prms := range sliceParams {
						nestedParams, okk := prms.(map[string]any)
						if okk {
							var p reflect.Value
							if elemType.Kind() == reflect.Ptr {
								// []*Struct: create new struct and get pointer
								structType := elemType.Elem()
								p = reflect.New(structType)
							} else {
								// []Struct: create new struct
								p = reflect.New(elemType)
							}

							err := MapToStructByTags(nestedParams, p.Interface())
							if err != nil {
								return err
							}

							if elemType.Kind() == reflect.Ptr {
								// []any{map[string]any{}} ===> []*Struct
								field.Set(reflect.Append(field, p))
							} else {
								// []any{map[string]any{}} ===> []Struct
								field.Set(reflect.Append(field, p.Elem()))
							}
						}
					}
				}
			}

		} else {
			// get the value corresponding to the key from the map
			if v, ok := params[mapTag]; ok {
				// set a value for a field
				if field.CanSet() {
					// Special handling for bool fields to handle string "false"/"true" from YAML
					if field.Kind() == reflect.Bool {
						switch val := v.(type) {
						case bool:
							field.SetBool(val)
						case string:
							if boolVal, err := strconv.ParseBool(val); err == nil {
								field.SetBool(boolVal)
							} else {
								return fmt.Errorf("cannot convert '%s' to bool for field '%s'", val, mapTag)
							}
						default:
							return fmt.Errorf("cannot convert %T to bool for field '%s'", v, mapTag)
						}
						// Special handling for int fields to handle float64 from YAML/JSON
					} else if field.Kind() == reflect.Int {
						switch val := v.(type) {
						case int:
							field.SetInt(int64(val))
						case int64:
							field.SetInt(val)
						case float64:
							// YAML/JSON numbers are parsed as float64, convert to int
							field.SetInt(int64(val))
						case string:
							if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
								field.SetInt(intVal)
							} else {
								return fmt.Errorf("cannot convert '%s' to int for field '%s'", val, mapTag)
							}
						default:
							return fmt.Errorf("cannot convert %T to int for field '%s'", v, mapTag)
						}
					} else {
						// Type-safe assignment based on field type
						fieldType := field.Type()
						valueType := reflect.TypeOf(v)

						if valueType == fieldType {
							// Direct assignment if types match
							field.Set(reflect.ValueOf(v))
						} else if fieldType.Kind() == reflect.String {
							// Convert to string if field expects string
							if v != nil {
								field.SetString(fmt.Sprintf("%v", v))
							}
						} else if fieldType.Kind() == reflect.Map && valueType.Kind() == reflect.Map {
							// Handle map type conversion
							if fieldType.Key().Kind() == reflect.String && fieldType.Elem().Kind() == reflect.String {
								// Convert map[string]interface{} to map[string]string
								if mapVal, ok := v.(map[string]interface{}); ok {
									newMap := make(map[string]string)
									for k, val := range mapVal {
										newMap[k] = fmt.Sprintf("%v", val)
									}
									field.Set(reflect.ValueOf(newMap))
								} else {
									field.Set(reflect.ValueOf(v))
								}
							} else {
								field.Set(reflect.ValueOf(v))
							}
						} else if valueType.ConvertibleTo(fieldType) {
							// Use Go's type conversion if possible
							field.Set(reflect.ValueOf(v).Convert(fieldType))
						} else {
							return fmt.Errorf("cannot assign %T to field '%s' of type %s", v, mapTag, fieldType)
						}
					}
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
//
//	type User struct {
//	  Name     string            `map:"name"`
//	  Age      int               `map:"age"`
//	  Metadata map[string]string `map:"metadata"`
//	}
//
//	user := User{
//	  Name: "John",
//	  Age: 30,
//	  Metadata: map[string]string{"role": "admin", "dept": "IT"},
//	}
//
//	result, err := StructToMapByTags(user)
//	// result: map[string]any{
//	//   "name": "John",
//	//   "age": 30,
//	//   "metadata": map[string]string{"role": "admin", "dept": "IT"},
//	// }
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

			// when the field is []???(slice)
		} else if field.Kind() == reflect.Slice {
			if fieldType.Type.Elem().Kind() == reflect.Struct {
				// []struct ===> []any{map[string]any{}}
				sliceValue := field
				var anySlice []any
				for i := 0; i < sliceValue.Len(); i++ {
					structElem := sliceValue.Index(i)
					structMap, err := StructToMapByTags(structElem.Interface())
					if err != nil {
						return nil, err
					}
					anySlice = append(anySlice, structMap)
				}
				result[mapTag] = anySlice
			} else {
				// other slice types (string, etc.)
				result[mapTag] = field.Interface()
			}

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
//
//	type Config struct {
//	  Name    string `map:"name" validate:"required"`
//	  Timeout int    `map:"timeout"`
//	}
//
//	params := ActionsParams{"name": "test", "timeout": "30"}
//	var config Config
//	err := AssignStruct(params, &config)
//	// config.Name = "test", config.Timeout = 30
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
				if strValue, ok := value.(string); ok {
					v.Field(i).SetString(strValue)
				} else {
					v.Field(i).SetString(fmt.Sprintf("%v", value))
				}
			case "int":
				var strValue string
				if str, ok := value.(string); ok {
					strValue = str
				} else {
					strValue = fmt.Sprintf("%v", value)
				}
				intValue, err := strconv.Atoi(strValue)
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

// StrmapToAnymap converts a map[string]string to map[string]any.
// This is a simple type conversion utility function.
//
// Example:
//
//	input := map[string]string{"name": "John", "age": "30"}
//	result := StrmapToAnymap(input)
//	// result: map[string]any{"name": "John", "age": "30"}
func StrmapToAnymap(strmap map[string]string) map[string]any {
	anymap := make(map[string]any)
	for k, v := range strmap {
		anymap[k] = v
	}
	return anymap
}

// HeaderToStringValue converts header values to strings for HTTP processing.
// This function ensures all header values are strings, converting numbers and other types as needed.
//
// Example:
//
//	data := map[string]any{
//	  "headers": map[string]any{
//	    "Content-Length": 1024,
//	    "X-Rate-Limit": 100.5,
//	    "Authorization": "Bearer token",
//	  },
//	}
//
//	result := HeaderToStringValue(data)
//	// result["headers"] = map[string]any{
//	//   "Content-Length": "1024",
//	//   "X-Rate-Limit": "100.5",
//	//   "Authorization": "Bearer token",
//	// }
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

// AnyToString attempts to convert any type to a string.
// Returns the string representation and a boolean indicating success.
//
// Example:
//
//	str, ok := AnyToString(42)        // "42", true
//	str, ok := AnyToString(3.14)      // "3.14", true
//	str, ok := AnyToString("hello")   // "hello", true
//	str, ok := AnyToString(nil)       // "nil", true
//	str, ok := AnyToString([]int{1})  // "", false
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

// EnvMap returns all environment variables as a map[string]string.
// Each environment variable is parsed from "KEY=VALUE" format.
//
// Example:
//
//	env := EnvMap()
//	// env contains all environment variables like:
//	// {"PATH": "/usr/bin:/bin", "HOME": "/home/user", "USER": "username", ...}
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

// TitleCase converts a string to title case using a specified separator character.
// Each part separated by the character has its first letter capitalized.
//
// Example:
//
//	TitleCase("content-type", "-")     // "Content-Type"
//	TitleCase("user_name", "_")        // "User_Name"
//	TitleCase("hello-world-test", "-") // "Hello-World-Test"
func TitleCase(st string, char string) string {
	parts := strings.Split(st, char)
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, char)
}

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
