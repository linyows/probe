package http

import (
	hp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/linyows/probe/actionrpc"
)

// writeSpec writes an OpenAPI document of one operation, GET /users/{id},
// served under /v1 of base.
func writeSpec(t *testing.T, base string) string {
	t.Helper()
	spec := `openapi: 3.0.3
info: {title: users, version: "1"}
servers:
- url: ` + base + `/v1
paths:
  /users/{id}:
    get:
      parameters:
      - {name: id, in: path, required: true, schema: {type: integer}}
      responses:
        "200":
          description: a user
          content:
            application/json:
              schema:
                type: object
                required: [id, name]
                properties:
                  id: {type: integer}
                  name: {type: string}
`
	path := filepath.Join(t.TempDir(), "openapi.yml")
	if err := os.WriteFile(path, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func usersServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(hp.HandlerFunc(func(w hp.ResponseWriter, r *hp.Request) {
		if r.URL.Path == "/v1/old" {
			hp.Redirect(w, r, "/v1/users/1", hp.StatusFound)
			return
		}
		code, body := hp.StatusNotFound, `{}`
		switch r.URL.Path {
		case "/v1/users/1":
			code, body = hp.StatusOK, `{"id":1,"name":"probe"}`
		case "/v1/users/2":
			code, body = hp.StatusOK, `{"id":"2"}`
		case "/v1/users/3":
			code, body = hp.StatusInternalServerError, `{"error":"down"}`
		case "/v1/teams":
			code, body = hp.StatusOK, `[]`
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRequestStepChecksResponseAgainstOpenAPI(t *testing.T) {
	srv := usersServer(t)
	spec := writeSpec(t, srv.URL)

	tests := []struct {
		name string
		path string
		// want are the reasons expected, one for each violation, matched
		// by substring; none when the response keeps to the document.
		want []string
		// field is the field the first violation names, when it names one.
		field string
	}{
		{name: "a response the document allows", path: "/users/1"},
		{name: "a redirect is matched by where it leads", path: "/old"},
		{name: "a body that breaks the schema", path: "/users/2", want: []string{"missing property 'name'", "got string, want integer"}},
		{name: "a status the operation does not declare", path: "/users/3", want: []string{"'500'"}},
		{name: "a path the document does not have", path: "/teams", want: []string{"'/v1/teams'"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ret, _, err := RequestStep(actionrpc.Call{With: map[string]any{
				"url":     srv.URL + "/v1",
				"get":     tt.path,
				"openapi": map[string]any{"spec": spec},
			}})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			res := ret["res"].(map[string]any)
			got, ok := res["violations"].([]any)
			if !ok {
				t.Fatalf("res.violations = %#v, want a list", res["violations"])
			}
			if len(got) != len(tt.want) {
				t.Fatalf("violations = %#v, want %d", got, len(tt.want))
			}
			for i, w := range tt.want {
				v := got[i].(map[string]any)
				if v["in"] != "response" {
					t.Errorf("violations[%d].in = %v, want response", i, v["in"])
				}
				text := v["message"].(string) + " " + v["reason"].(string)
				if !strings.Contains(text, w) {
					t.Errorf("violations[%d] = %#v, want it to say %q", i, v, w)
				}
			}
			if _, ok := ret["req"].(map[string]any)["openapi"]; ok {
				t.Error("openapi should not be carried into req")
			}
		})
	}
}

func TestRequestStepNamesTheFieldThatBreaksTheSchema(t *testing.T) {
	srv := usersServer(t)
	ret, _, err := RequestStep(actionrpc.Call{With: map[string]any{
		"url":     srv.URL + "/v1",
		"get":     "/users/2",
		"openapi": map[string]any{"spec": writeSpec(t, srv.URL)},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range ret["res"].(map[string]any)["violations"].([]any) {
		if v.(map[string]any)["field"] == "$.id" {
			return
		}
	}
	t.Errorf("violations = %#v, want one for $.id", ret["res"].(map[string]any)["violations"])
}

func TestRequestStepWithoutOpenAPIChecksNothing(t *testing.T) {
	srv := usersServer(t)

	for _, openapi := range []any{nil, false} {
		with := map[string]any{"url": srv.URL + "/v1", "get": "/teams"}
		if openapi != nil {
			with["openapi"] = openapi
		}
		ret, _, err := RequestStep(actionrpc.Call{With: with})
		if err != nil {
			t.Fatalf("with openapi %v: unexpected error: %v", openapi, err)
		}
		if v, ok := ret["res"].(map[string]any)["violations"]; ok {
			t.Errorf("with openapi %v: res.violations = %#v, want none", openapi, v)
		}
	}
}

func TestRequestStepOpenAPIRejected(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(hp.HandlerFunc(func(w hp.ResponseWriter, r *hp.Request) {
		hits.Add(1)
	}))
	defer srv.Close()

	broken := filepath.Join(t.TempDir(), "broken.yml")
	if err := os.WriteFile(broken, []byte("openapi: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		openapi any
		wantErr string
	}{
		{name: "true", openapi: true, wantErr: "openapi must be a map with spec, or false"},
		{name: "a path alone", openapi: "openapi.yml", wantErr: "openapi must be a map with spec, or false"},
		{name: "an unknown key", openapi: map[string]any{"spec": broken, "strict": true}, wantErr: "openapi takes spec, not strict"},
		{name: "no spec", openapi: map[string]any{}, wantErr: "openapi.spec must be the path of an OpenAPI document"},
		{name: "a missing file", openapi: map[string]any{"spec": filepath.Join(t.TempDir(), "none.yml")}, wantErr: "openapi.spec:"},
		{name: "a document that cannot be parsed", openapi: map[string]any{"spec": broken}, wantErr: "openapi.spec: " + broken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := RequestStep(actionrpc.Call{With: map[string]any{
				"url":     srv.URL,
				"get":     "/",
				"openapi": tt.openapi,
			}})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
	// A step whose document cannot be used sends nothing.
	if n := hits.Load(); n != 0 {
		t.Errorf("the server was sent %d requests, want 0", n)
	}
}
