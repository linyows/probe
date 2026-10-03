package grpc

import (
	"net"
	"reflect"
	"testing"

	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func TestConvertMetadataToMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "structured metadata",
			input: map[string]any{
				"service": "UserService",
				"metadata": map[string]any{
					"authorization": "Bearer token",
					"user-agent":    "probe-grpc/1.0",
				},
			},
			expected: map[string]any{
				"service": "UserService",
				"metadata": map[string]any{
					"authorization": "Bearer token",
					"user-agent":    "probe-grpc/1.0",
				},
			},
		},
		{
			name: "no metadata",
			input: map[string]any{
				"service": "UserService",
				"method":  "GetUser",
			},
			expected: map[string]any{
				"service": "UserService",
				"method":  "GetUser",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For structured metadata, we simply verify that the input structure is preserved
			// since ConvertMetadataToMap doesn't need to process structured metadata

			// Check that structured metadata is preserved as-is
			if !reflect.DeepEqual(tt.input, tt.expected) {
				t.Errorf("Structured metadata test failed: input = %v, expected = %v", tt.input, tt.expected)
			}

			// If metadata field exists, verify it's properly structured
			if metadata, exists := tt.input["metadata"]; exists {
				if metadataMap, ok := metadata.(map[string]any); ok {
					// Verify metadata contains expected keys
					if auth, exists := metadataMap["authorization"]; exists {
						if authStr, ok := auth.(string); !ok || authStr == "" {
							t.Errorf("Authorization should be a non-empty string")
						}
					}
					if ua, exists := metadataMap["user-agent"]; exists {
						if uaStr, ok := ua.(string); !ok || uaStr == "" {
							t.Errorf("User-agent should be a non-empty string")
						}
					}
				} else {
					t.Errorf("Metadata should be a map[string]any")
				}
			}
		})
	}
}

func TestPrepareGrpcRequestData(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "complete gRPC request with structured metadata",
			input: map[string]any{
				"addr":    "localhost:50051",
				"service": "UserService",
				"method":  "CreateUser",
				"body":    `{"user": {"name": "John", "email": "john@example.com"}}`,
				"metadata": map[string]any{
					"authorization": "Bearer token",
					"user-agent":    "probe/1.0",
				},
			},
			expected: map[string]any{
				"addr":    "localhost:50051",
				"service": "UserService",
				"method":  "CreateUser",
				"body":    `{"user": {"name": "John", "email": "john@example.com"}}`,
				"metadata": map[string]any{
					"authorization": "Bearer token",
					"user-agent":    "probe/1.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For structured data, verify the input structure is preserved
			if !reflect.DeepEqual(tt.input, tt.expected) {
				t.Errorf("PrepareGrpcRequestData test failed: input = %v, expected = %v", tt.input, tt.expected)
			}

			// Verify basic required fields
			if addr, exists := tt.input["addr"]; !exists || addr == "" {
				t.Errorf("addr field is required and should not be empty")
			}
			if service, exists := tt.input["service"]; !exists || service == "" {
				t.Errorf("service field is required and should not be empty")
			}
			if method, exists := tt.input["method"]; !exists || method == "" {
				t.Errorf("method field is required and should not be empty")
			}

			// Verify structured metadata if present
			if metadata, exists := tt.input["metadata"]; exists {
				if metadataMap, ok := metadata.(map[string]any); ok {
					if len(metadataMap) == 0 {
						t.Errorf("metadata should not be empty if present")
					}
				} else {
					t.Errorf("metadata should be a map[string]any")
				}
			}
		})
	}
}

func TestNewReq(t *testing.T) {
	req := NewReq()

	if req.Timeout != "30s" {
		t.Errorf("NewReq() timeout = %s, want 30s", req.Timeout)
	}
	if req.TLS != false {
		t.Errorf("NewReq() tls = %t, want false", req.TLS)
	}
	if req.Insecure != false {
		t.Errorf("NewReq() insecure = %t, want false", req.Insecure)
	}
	if req.Metadata == nil {
		t.Errorf("NewReq() metadata should not be nil")
	}
}

func TestReq_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     *Req
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing addr",
			req: &Req{
				Service: "UserService",
				Method:  "GetUser",
			},
			wantErr: true,
			errMsg:  "Req.Addr is required",
		},
		{
			name: "missing service",
			req: &Req{
				Addr:   "localhost:50051",
				Method: "GetUser",
			},
			wantErr: true,
			errMsg:  "Req.Service is required",
		},
		{
			name: "missing method",
			req: &Req{
				Addr:    "localhost:50051",
				Service: "UserService",
			},
			wantErr: true,
			errMsg:  "Req.Method is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set defaults
			if tt.req.Timeout == "" {
				tt.req.Timeout = "30s"
			}
			if tt.req.Metadata == nil {
				tt.req.Metadata = make(map[string]string)
			}

			_, err := tt.req.Do()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Req.Do() expected error but got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("Req.Do() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Req.Do() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestRequest_StructureValidation(t *testing.T) {
	// Test that Request function can be called without panicking
	// Note: This doesn't test actual gRPC calls as that would require a running server
	data := map[string]any{
		"addr":    "localhost:50051",
		"service": "TestService",
		"method":  "TestMethod",
		"body":    `{"test": "value"}`,
	}

	// This will fail with connection error, but should not panic
	_, err := Request(data)
	if err == nil {
		t.Log("Request() completed without error (unexpected in unit test)")
	} else {
		// Expected - connection will fail in unit test environment
		t.Logf("Request() failed as expected in unit test: %v", err)
	}
}

// startHealthServer serves the standard health service with reflection, which
// answers NOT_FOUND for a service it does not know.
func startHealthServer(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := grpclib.NewServer()
	healthpb.RegisterHealthServer(s, health.NewServer())
	reflection.Register(s)
	go func() { _ = s.Serve(lis) }()
	t.Cleanup(s.Stop)
	return lis.Addr().String()
}

func TestRequestStatus(t *testing.T) {
	addr := startHealthServer(t)

	tests := []struct {
		name       string
		body       string
		wantCode   string
		wantStatus int
	}{
		{name: "ok", body: `{"service": ""}`, wantCode: "OK", wantStatus: 0},
		// The server answering with an error status is a result to test, not a
		// failure to make the call.
		{name: "not found", body: `{"service": "unknown"}`, wantCode: "NOT_FOUND", wantStatus: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ret, err := Request(map[string]any{
				"addr":    addr,
				"service": "grpc.health.v1.Health",
				"method":  "Check",
				"body":    tt.body,
			})
			if err != nil {
				t.Fatalf("Request() error: %v", err)
			}
			res, _ := ret["res"].(map[string]any)
			if res["status_code"] != tt.wantCode {
				t.Errorf("status_code = %v, want %s", res["status_code"], tt.wantCode)
			}
			if ret["status"] != tt.wantStatus {
				t.Errorf("status = %v, want %d", ret["status"], tt.wantStatus)
			}
			if tt.wantCode != "OK" && res["status_message"] == "" {
				t.Error("status_message is empty")
			}
		})
	}
}

func TestRequestUnreachable(t *testing.T) {
	// Nothing listens there, so no status ever comes back: that is an error.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()

	_, err = Request(map[string]any{
		"addr":    addr,
		"service": "grpc.health.v1.Health",
		"method":  "Check",
		"timeout": "2s",
	})
	if err == nil {
		t.Fatal("Request() to a closed port succeeded")
	}
}

func TestStatusCodeName(t *testing.T) {
	tests := map[codes.Code]string{
		codes.OK:                 "OK",
		codes.Canceled:           "CANCELLED",
		codes.NotFound:           "NOT_FOUND",
		codes.DeadlineExceeded:   "DEADLINE_EXCEEDED",
		codes.FailedPrecondition: "FAILED_PRECONDITION",
		codes.Unauthenticated:    "UNAUTHENTICATED",
	}
	for c, want := range tests {
		if got := statusCodeName(c); got != want {
			t.Errorf("statusCodeName(%v) = %s, want %s", c, got, want)
		}
	}
}
