package http

import (
	"reflect"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestNewReq(t *testing.T) {
	got := NewReq()

	expects := &Req{
		URL:    "",
		Method: "GET",
		Proto:  "HTTP/1.1",
		Header: map[string]string{
			"Accept":     "*/*",
			"User-Agent": "probe-http/1.0.0",
		},
	}

	if !reflect.DeepEqual(got, expects) {
		t.Errorf("\nExpected:\n%#v\nGot:\n%#v", expects, got)
	}
}

func TestDo(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	res := httpmock.NewStringResponder(200, "Hello World\n")
	httpmock.RegisterResponder("GET", "http://localhost:8080/foo/bar", res)

	req := NewReq()
	req.URL = "http://localhost:8080/foo/bar"

	got, err := req.Do()
	if err != nil {
		t.Errorf("got error %s", err)
	}

	expects := "Hello World\n"

	if string(got.Res.Body) != expects {
		t.Errorf("\nExpected:\n%s\nGot:\n%s", expects, got.Res.Body)
	}
}
