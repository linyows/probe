package probe

import (
	"encoding/json"
	"fmt"
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

// merge string maps
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

// converting from a map[string]any to a struct
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

// converting from a struct to a map[string]any
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

func FlattenInterface(i any) map[string]string {
	return flattenIf(i, "")
}

func flattenIf(input any, prefix string) map[string]string {
	res := make(map[string]string)

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

// Recursively convert a map[string]string to a map[string]any
func UnflattenInterface(flatMap map[string]string) map[string]any {
	result := make(map[string]any)

	for key, value := range flatMap {
		keys := strings.Split(key, flatkey)
		nestMap(result, keys, value)
	}

	return result
}

// A helper to set values for nested keys
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

func mustMarshalJSON(st string) map[string]any {
	var re map[string]any
	if err := json.Unmarshal([]byte(st), &re); err != nil {
		return map[string]any{
			"error_message": fmt.Sprintf("mustMarshalJSON error: %s", err),
		}
	}
	return re
}

func isJSON(st string) bool {
	trimmed := strings.TrimSpace(st)
	if len(trimmed) < 2 {
		return false
	}

	fChar := rune(trimmed[0])
	lChar := rune(trimmed[len(trimmed)-1])

	return (fChar == '{' && lChar == '}') || (fChar == '[' && lChar == ']')
}
