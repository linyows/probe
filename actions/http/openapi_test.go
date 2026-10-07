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
	"github.com/pb33f/libopenapi"
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

func TestRequestStepChecksBodiesBeyondJSON(t *testing.T) {
	srv := httptest.NewServer(hp.HandlerFunc(func(w hp.ResponseWriter, r *hp.Request) {
		bodies := map[string]struct{ contentType, body string }{
			"/text/long":     {"text/plain", "abcd"},
			"/text/short":    {"text/plain", "abc"},
			"/json/null":     {"application/json", "null"},
			"/json/nullable": {"application/json", "null"},
			"/json/empty":    {"application/json", ""},
			"/pdf":           {"application/pdf", "%PDF-1.7"},
		}
		b, ok := bodies[r.URL.Path]
		if !ok {
			w.WriteHeader(hp.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", b.contentType)
		_, _ = w.Write([]byte(b.body))
	}))
	defer srv.Close()

	object := `{type: object, required: [id], properties: {id: {type: integer}}}`
	spec := `openapi: 3.0.3
info: {title: bodies, version: "1"}
paths:
  /text/long:
    get:
      responses:
        "200": {description: text, content: {text/plain: {schema: {type: string, maxLength: 3}}}}
  /text/short:
    get:
      responses:
        "200": {description: text, content: {text/plain: {schema: {type: string, maxLength: 3}}}}
  /json/null:
    get:
      responses:
        "200": {description: an object, content: {application/json: {schema: ` + object + `}}}
  /json/nullable:
    get:
      responses:
        "200": {description: an object or null, content: {application/json: {schema: {type: object, nullable: true}}}}
  /json/empty:
    get:
      responses:
        "200": {description: an object, content: {application/json: {schema: ` + object + `}}}
  /pdf:
    get:
      responses:
        "200": {description: a PDF, content: {application/pdf: {schema: {type: string, format: binary}}}}
  /none:
    get:
      responses:
        "204": {description: nothing}
`
	path := filepath.Join(t.TempDir(), "openapi.yml")
	if err := os.WriteFile(path, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path string
		want string // the violation expected, matched by substring; empty for none
	}{
		{"/text/long", "maxLength: got 4, want 3"},
		{"/text/short", ""},
		{"/json/null", "the schema does not allow null"},
		{"/json/nullable", ""},
		{"/json/empty", "the response declares a schema for application/json"},
		// A body of a type the validator cannot decode is left unchecked.
		{"/pdf", ""},
		{"/none", ""},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			ret, _, err := RequestStep(actionrpc.Call{With: map[string]any{
				"url":     srv.URL,
				"get":     tt.path,
				"openapi": map[string]any{"spec": path},
			}})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := ret["res"].(map[string]any)["violations"].([]any)
			if tt.want == "" {
				if len(got) != 0 {
					t.Errorf("violations = %#v, want none", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("violations = %#v, want 1", got)
			}
			v := got[0].(map[string]any)
			if text := v["message"].(string) + " " + v["reason"].(string); !strings.Contains(text, tt.want) {
				t.Errorf("violation = %#v, want it to say %q", v, tt.want)
			}
		})
	}
}

func TestAllowsNull(t *testing.T) {
	const doc = `openapi: 3.1.0
info: {title: null, version: "1"}
paths: {}
components:
  schemas:
    Object: {type: object}
    Nullable: {type: object, nullable: true}
    TypeNull: {type: [object, "null"]}
    EnumWithNull: {enum: [a, null]}
    EnumWithoutNull: {enum: [a, b]}
    Untyped: {description: anything}
    AllOfObject: {allOf: [{$ref: "#/components/schemas/Object"}]}
    AllOfNullable: {allOf: [{$ref: "#/components/schemas/Object"}], nullable: true}
    OneOfWithNull: {oneOf: [{$ref: "#/components/schemas/Object"}, {type: "null"}]}
    AnyOfObjects: {anyOf: [{$ref: "#/components/schemas/Object"}, {type: string}]}
`
	d, err := libopenapi.NewDocument([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	m, err := d.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]bool{
		"Object":          false,
		"Nullable":        true,
		"TypeNull":        true,
		"EnumWithNull":    true,
		"EnumWithoutNull": false,
		"Untyped":         true,
		"AllOfObject":     false,
		"AllOfNullable":   true,
		"OneOfWithNull":   true,
		"AnyOfObjects":    false,
	}
	for name, want := range tests {
		s := m.Model.Components.Schemas.GetOrZero(name).Schema()
		if got := allowsNull(s, 0); got != want {
			t.Errorf("allowsNull(%s) = %v, want %v", name, got, want)
		}
	}
}
