package grpc

import (
	"testing"
)

// Note: ConvertBodyToJson functionality is now handled by probe.ConvertBodyToJson

func TestConvertMetadataToMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name: "simple metadata",
			input: map[string]string{
				"metadata__authorization": "Bearer token",
				"metadata__user-agent":    "probe-grpc/1.0",
				"service":                 "UserService",
			},
			expected: map[string]string{
				"metadata__authorization": "Bearer token",
				"metadata__user-agent":    "probe-grpc/1.0",
				"service":                 "UserService",
			},
		},
		{
			name: "no metadata",
			input: map[string]string{
				"service": "UserService",
				"method":  "GetUser",
			},
			expected: map[string]string{
				"service": "UserService",
				"method":  "GetUser",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of input to avoid modifying the test case
			data := make(map[string]string)
			for k, v := range tt.input {
				data[k] = v
			}

			err := ConvertMetadataToMap(data)
			if err != nil {
				t.Errorf("ConvertMetadataToMap() error = %v", err)
				return
			}

			// Check that the result matches expected
			if len(data) != len(tt.expected) {
				t.Errorf("ConvertMetadataToMap() result length = %d, want %d", len(data), len(tt.expected))
			}

			for key, expected := range tt.expected {
				if actual, exists := data[key]; !exists || actual != expected {
					t.Errorf("ConvertMetadataToMap() data[%s] = %s, want %s", key, actual, expected)
				}
			}
		})
	}
}

func TestPrepareGrpcRequestData(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name: "complete gRPC request",
			input: map[string]string{
				"addr":                    "localhost:50051",
				"service":                 "UserService",
				"method":                  "CreateUser",
				"body__user__name":        "John",
				"body__user__email":       "john@example.com",
				"metadata__authorization": "Bearer token",
				"metadata__user-agent":    "probe/1.0",
			},
			expected: map[string]string{
				"addr":                    "localhost:50051",
				"service":                 "UserService",
				"method":                  "CreateUser",
				"body":                    `{"user":{"email":"john@example.com","name":"John"}}`,
				"metadata__authorization": "Bearer token",
				"metadata__user-agent":    "probe/1.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of input to avoid modifying the test case
			data := make(map[string]string)
			for k, v := range tt.input {
				data[k] = v
			}

			err := PrepareGrpcRequestData(data)
			if err != nil {
				t.Errorf("PrepareGrpcRequestData() error = %v", err)
				return
			}

			// Check key fields
			for key, expected := range tt.expected {
				if actual, exists := data[key]; !exists || actual != expected {
					t.Errorf("PrepareGrpcRequestData() data[%s] = %s, want %s", key, actual, expected)
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
	data := map[string]string{
		"addr":    "localhost:50051",
		"service": "TestService",
		"method":  "TestMethod",
		"body": `{"test": "value"}`,
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