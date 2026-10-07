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
