package probe

import (
	"reflect"
	"testing"
)

type TestStruct struct {
	String      string            `map:"string"`
	Number      int               `map:"number"`
	Bool        bool              `map:"bool"`
	Bytes       []byte            `map:"bytes"`
	Required    string            `map:"required" validate:"required"`
	MapStrStr   map[string]string `map:"map_str_str"`
	EmbedStruct TestEmbedStruct   `map:"embed_struct"`
}

type TestEmbedStruct struct {
	Name string `map:"name"`
}

func TestMapToStructByTags(t *testing.T) {
	got := TestStruct{
		String: "hello, world!",
		MapStrStr: map[string]string{
			"foo":   "bar",
			"hello": "world",
		},
	}

	params := map[string]any{
		"string":   "s-t-r-i-n-g",
		"number":   123,
		"bool":     false,
		"bytes":    "b-y-t-e-s",
		"required": "required!",
		"map_str_str": map[string]any{
			"foo": "f-o-o",
			"bar": "b-a-r",
			"baz": "b-a-z",
		},
		"embed_struct": map[string]any{
			"name": "probe",
		},
	}

	expects := TestStruct{
		String:   "s-t-r-i-n-g",
		Number:   123,
		Bool:     false,
		Bytes:    []byte("b-y-t-e-s"),
		Required: "required!",
		MapStrStr: map[string]string{
			"foo":   "f-o-o",
			"bar":   "b-a-r",
			"baz":   "b-a-z",
			"hello": "world",
		},
		EmbedStruct: TestEmbedStruct{
			Name: "probe",
		},
	}

	if err := MapToStructByTags(params, &got); err != nil {
		t.Errorf("MapToStructByTags error %s", err)
	}

	if !reflect.DeepEqual(got, expects) {
		t.Errorf("\nExpected:\n%#v\nGot:\n%#v", expects, got)
	}
}

func TestMapToStructByTags_Required(t *testing.T) {
	got := TestStruct{}
	params := map[string]any{"string": "yo"}
	err := MapToStructByTags(params, &got)

	if err.Error() != "required field 'required' is missing" {
		t.Errorf("MapToStructByTags error is wrong: %s", err)
	}
}

func TestFlattenInterface(t *testing.T) {
	expects := map[string]string{
		"map_str_str__foo": "f-o-o",
		"map_str_str__bar": "b-a-r",
		"string":           "s-t-r-i-n-g",
	}

	data := map[string]any{
		"map_str_str": map[string]any{
			"foo": "f-o-o",
			"bar": "b-a-r",
		},
		"string": "s-t-r-i-n-g",
	}

	got := FlattenInterface(data)

	if !reflect.DeepEqual(got, expects) {
		t.Errorf("\nExpected:\n%#v\nGot:\n%#v", expects, got)
	}
}

func TestUnflattenInterface(t *testing.T) {
	expects := map[string]any{
		"map_str_str": map[string]any{
			"foo": "f-o-o",
			"bar": "b-a-r",
		},
		"string": "s-t-r-i-n-g",
	}

	data := map[string]string{
		"map_str_str__foo": "f-o-o",
		"map_str_str__bar": "b-a-r",
		"string":           "s-t-r-i-n-g",
	}

	got := UnflattenInterface(data)

	if !reflect.DeepEqual(got, expects) {
		t.Errorf("\nExpected:\n%#v\nGot:\n%#v", expects, got)
	}
}
