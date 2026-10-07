package grpc

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/linyows/probe/actions/grpc/testserver/pb"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// userServer answers GetUser with one user, or NOT_FOUND.
type userServer struct {
	pb.UnimplementedUserServiceServer
}

func (userServer) GetUser(_ context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	if req.UserId != "123" {
		return nil, status.Errorf(codes.NotFound, "user %s not found", req.UserId)
	}
	return &pb.GetUserResponse{User: &pb.User{
		Id:          "123",
		Name:        "Test User",
		Email:       "test@example.com",
		Profile:     &pb.Profile{Age: 30, Location: "Tokyo"},
		Preferences: []string{"email"},
		CreatedAt:   "2026-10-08T00:00:00Z",
	}}, nil
}

func startUserServer(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := grpclib.NewServer()
	pb.RegisterUserServiceServer(s, userServer{})
	reflection.Register(s)
	go func() { _ = s.Serve(lis) }()
	t.Cleanup(s.Stop)
	return lis.Addr().String()
}

// userProto is the .proto file the server is built from.
func userProto(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testserver", "pb", "user_service.proto"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// writeProto writes content as users.proto in a directory of its own and
// returns its path.
func writeProto(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "users.proto")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRequestChecksAgainstProto(t *testing.T) {
	addr := startUserServer(t)
	spec := userProto(t)

	type want struct {
		in   string
		text string // matched by substring against the message and reason
	}
	tests := []struct {
		name  string
		proto string
		body  string
		opts  map[string]any // merged into proto
		want  []want
	}{
		{name: "the .proto files the server is built from", proto: spec},
		{name: "a call answered with an error status", proto: spec, body: `{"user_id": "999"}`},
		{
			name:  "a field of another type",
			proto: strings.Replace(spec, "string email = 3;", "int32 email = 3;", 1),
			want: []want{
				{"response", "the server's User declares string email = 3"},
				{"response", "field 3 of User is not encoded as int32 email"},
			},
		},
		{
			name:  "a field the server does not declare",
			proto: strings.Replace(spec, "string created_at = 6;", "string created_at = 6;\n  string phone = 7;", 1),
			want:  []want{{"response", "the server's User declares no field 7"}},
		},
		{
			name:  "a field the .proto files do not declare",
			proto: strings.Replace(spec, "string created_at = 6;", "", 1),
		},
		{
			name:  "a field the .proto files do not declare, strict",
			proto: strings.Replace(spec, "string created_at = 6;", "", 1),
			opts:  map[string]any{"strict": true},
			want: []want{
				{"response", "the server's User declares string created_at = 6"},
				{"response", "field 6 of User is not declared in the .proto files"},
			},
		},
		{
			name:  "a request body the request message does not take",
			proto: strings.Replace(spec, "string user_id = 1;", "string id = 1;", 1),
			want: []want{
				{"request", "request body does not keep to GetUserRequest"},
				{"response", "the server's GetUserRequest declares string user_id = 1"},
			},
		},
		{
			name:  "a request body left unchecked",
			proto: strings.Replace(spec, "string user_id = 1;", "string id = 1;", 1),
			opts:  map[string]any{"request": false},
			want:  []want{{"response", "the server's GetUserRequest declares string user_id = 1"}},
		},
		{
			name:  "a method the .proto files do not declare",
			proto: strings.Replace(spec, "rpc GetUser(GetUserRequest) returns (GetUserResponse);", "", 1),
			want:  []want{{"response", "the .proto files declare no method GetUser in UserService"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto := map[string]any{"files": []any{writeProto(t, tt.proto)}}
			for k, v := range tt.opts {
				proto[k] = v
			}
			path := proto["files"].([]any)[0].(string)
			proto["import_paths"] = []any{filepath.Dir(path)}
			body := tt.body
			if body == "" {
				body = `{"user_id": "123", "include_profile": true}`
			}
			ret, err := Request(map[string]any{
				"addr":    addr,
				"service": "UserService",
				"method":  "GetUser",
				"body":    body,
				"proto":   proto,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got, ok := ret["res"].(map[string]any)["violations"].([]any)
			if !ok {
				t.Fatalf("res.violations = %#v, want a list", ret["res"].(map[string]any)["violations"])
			}
			if len(got) != len(tt.want) {
				t.Fatalf("violations = %#v, want %d", got, len(tt.want))
			}
			for i, w := range tt.want {
				v := got[i].(map[string]any)
				reason, _ := v["reason"].(string)
				if v["in"] != w.in || !strings.Contains(v["message"].(string)+" "+reason, w.text) {
					t.Errorf("violations[%d] = %#v, want %s saying %q", i, v, w.in, w.text)
				}
			}
		})
	}
}

func TestRequestWithoutProtoChecksNothing(t *testing.T) {
	addr := startUserServer(t)
	for _, proto := range []any{nil, false} {
		with := map[string]any{"addr": addr, "service": "UserService", "method": "GetUser", "body": `{"user_id": "123"}`}
		if proto != nil {
			with["proto"] = proto
		}
		ret, err := Request(with)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v, ok := ret["res"].(map[string]any)["violations"]; ok {
			t.Errorf("with proto %v: res.violations = %#v, want none", proto, v)
		}
	}
}

func TestTakeProtoRejected(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.proto")
	if err := os.WriteFile(broken, []byte("syntax = \"proto3\";\nmessage {"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		proto   any
		wantErr string
	}{
		{name: "true", proto: true, wantErr: "proto must be a map with files, or false"},
		{name: "a path alone", proto: "users.proto", wantErr: "proto must be a map with files, or false"},
		{name: "an unknown key", proto: map[string]any{"files": []any{broken}, "spec": "x"}, wantErr: "proto takes files, import_paths, request and strict, not spec"},
		{name: "no files", proto: map[string]any{}, wantErr: "proto.files must list the .proto files"},
		{name: "files not a list", proto: map[string]any{"files": "users.proto"}, wantErr: "proto.files must be a list of paths"},
		{name: "strict not a boolean", proto: map[string]any{"files": []any{broken}, "strict": "yes"}, wantErr: "proto.strict must be true or false"},
		{name: "a file outside the import paths", proto: map[string]any{"files": []any{broken}, "import_paths": []any{t.TempDir()}}, wantErr: "is under none of the import paths"},
		{name: "a file that does not compile", proto: map[string]any{"files": []any{broken}, "import_paths": []any{dir}}, wantErr: "proto.files:"},
		{name: "a file that does not exist", proto: map[string]any{"files": []any{filepath.Join(dir, "none.proto")}, "import_paths": []any{dir}}, wantErr: "proto.files:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := takeProto(map[string]any{"proto": tt.proto})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("err = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestCompareMessagesCardinality(t *testing.T) {
	compile := func(label string) *contract {
		dir := t.TempDir()
		path := filepath.Join(dir, "m.proto")
		if err := os.WriteFile(path, []byte("syntax = \"proto2\";\nmessage M { "+label+" string a = 1; }\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		c, err := loadContract([]string{path}, []string{dir})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	message := func(c *contract) protoreflect.MessageDescriptor {
		return c.files[0].Messages().ByName("M")
	}
	spec, server := compile("required"), compile("optional")

	out := spec.compareMessages(message(spec), message(server), "$", map[[2]protoreflect.FullName]bool{})
	if len(out) != 1 || !strings.Contains(out[0].(map[string]any)["message"].(string), "declares string a as optional") {
		t.Errorf("violations = %v, want the cardinality told apart", out)
	}
	if out := spec.compareMessages(message(spec), message(spec), "$", map[[2]protoreflect.FullName]bool{}); len(out) != 0 {
		t.Errorf("violations = %v, want none for the same definition", out)
	}
}

// A request body the server's definition cannot take either cannot be sent,
// so the step fails as a request that breaks its contract, with nothing
// sent, rather than as an action that could not run.
func TestRequestBodyNeitherDefinitionTakes(t *testing.T) {
	addr := startUserServer(t)
	path := writeProto(t, userProto(t))
	with := func(proto map[string]any) map[string]any {
		w := map[string]any{"addr": addr, "service": "UserService", "method": "GetUser", "body": `{"user_id": 123}`}
		if proto != nil {
			w["proto"] = proto
		}
		return w
	}
	files := map[string]any{"files": []any{path}, "import_paths": []any{filepath.Dir(path)}}

	ret, err := Request(with(files))
	if err != nil {
		t.Fatalf("err = %v, want the violation as a result", err)
	}
	res := ret["res"].(map[string]any)
	if res["status_code"] != "" {
		t.Errorf("status_code = %v, want none: nothing should have been sent", res["status_code"])
	}
	vs, _ := res["violations"].([]any)
	if len(vs) != 1 || vs[0].(map[string]any)["in"] != "request" {
		t.Errorf("violations = %v, want the request's", vs)
	}
	if ret["status"] != 1 {
		t.Errorf("status = %v, want 1", ret["status"])
	}

	// Without a contract, or with the request left unchecked, it stays an
	// error of the action.
	unchecked := map[string]any{"files": []any{path}, "import_paths": []any{filepath.Dir(path)}, "request": false}
	for _, proto := range []map[string]any{nil, unchecked} {
		if _, err := Request(with(proto)); err == nil || !strings.Contains(err.Error(), "failed to unmarshal request JSON") {
			t.Errorf("with proto %v: err = %v, want the action's error", proto, err)
		}
	}
}

func TestCheckRequestOfAnEmptyBody(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.proto")
	content := "syntax = \"proto2\";\nmessage Req { required string id = 1; }\nmessage Res {}\nservice S { rpc Get(Req) returns (Res); }\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := loadContract([]string{path}, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	c.request = true
	spec := c.method("S", "Get")

	// An empty body is an empty message, without the required field.
	out := c.checkRequest(spec, "")
	if len(out) != 1 || !strings.Contains(out[0].(map[string]any)["reason"].(string), "required field") {
		t.Errorf("violations = %v, want the required field missing", out)
	}
	if out := c.checkRequest(spec, `{"id": "1"}`); len(out) != 0 {
		t.Errorf("violations = %v, want none", out)
	}
}

func TestImportName(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		file string
		want string // empty when the file is under none of the import paths
	}{
		{filepath.Join(dir, "users.proto"), "users.proto"},
		{filepath.Join(dir, "user", "v1", "user.proto"), "user/v1/user.proto"},
		// A name that starts with two dots is under the directory.
		{filepath.Join(dir, "..schemas", "users.proto"), "..schemas/users.proto"},
		{filepath.Join(filepath.Dir(dir), "other.proto"), ""},
	}
	for _, tt := range tests {
		got, err := importName(tt.file, []string{dir})
		if tt.want == "" {
			if err == nil {
				t.Errorf("importName(%s) = %q, want an error", tt.file, got)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("importName(%s) = %q, %v; want %q", tt.file, got, err, tt.want)
		}
	}
}
