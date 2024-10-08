package http

import (
	"reflect"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestNewReq(t *testing.T) {
	p := map[string]string{
		"host": "http://localhost:8080",
		"path": "/foo/bar",
	}

	got, err := NewReq(p)
	if err != nil {
		t.Errorf("got error %s", err)
	}

	expects := &Req{
		Host:   "http://localhost:8080",
		Path:   "/foo/bar",
		Method: "GET",
		Ver:    "HTTP/1.1",
		Accept: "*/*",
		UA:     "probe/1.0.0",
	}

	if !reflect.DeepEqual(got, expects) {
		t.Errorf("\nExpected:\n%#v\nGot:\n%#v", expects, got)
	}
}

func TestReqDo(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	res := httpmock.NewStringResponder(200, "Hello World\n")
	httpmock.RegisterResponder("GET", "http://localhost:8080/foo/bar", res)

	p := map[string]string{
		"host": "http://localhost:8080",
		"path": "/foo/bar",
	}

	req, err := NewReq(p)
	if err != nil {
		t.Errorf("got error %s", err)
	}

	got, err := req.Do()
	if err != nil {
		t.Errorf("got error %s", err)
	}

	expects := "Hello World\n"

	if string(got) != expects {
		t.Errorf("\nExpected:\n%s\nGot:\n%s", expects, got)
	}
}
