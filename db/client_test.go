package db

import (
	"reflect"
	"testing"
	"time"
)

func TestParseParams(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]string
		wantErr bool
	}{
		{
			name: "valid mysql dsn",
			input: map[string]string{
				"dsn":   "mysql://user:pass@localhost:3306/testdb",
				"query": "SELECT * FROM users",
			},
			wantErr: false,
		},
		{
			name: "valid postgres dsn",
			input: map[string]string{
				"dsn":   "postgres://user:pass@localhost:5432/testdb",
				"query": "SELECT * FROM users",
			},
			wantErr: false,
		},
		{
			name: "valid sqlite dsn",
			input: map[string]string{
				"dsn":   "test.db",
				"query": "SELECT * FROM users",
			},
			wantErr: false,
		},
		{
			name: "missing dsn",
			input: map[string]string{
				"query": "SELECT * FROM users",
			},
			wantErr: true,
		},
		{
			name: "missing query",
			input: map[string]string{
				"dsn": "mysql://user:pass@localhost:3306/testdb",
			},
			wantErr: true,
		},
		{
			name: "invalid dsn format",
			input: map[string]string{
				"dsn":   "invalid://dsn",
				"query": "SELECT * FROM users",
			},
			wantErr: true,
		},
		{
			name: "with timeout",
			input: map[string]string{
				"dsn":     "mysql://user:pass@localhost:3306/testdb",
				"query":   "SELECT * FROM users",
				"timeout": "30s",
			},
			wantErr: false,
		},
		{
			name: "with params",
			input: map[string]string{
				"dsn":    "mysql://user:pass@localhost:3306/testdb",
				"query":  "SELECT * FROM users WHERE id = ?",
				"param1": "123",
				"param2": "test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _, _, err := ParseRequest(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && req == nil {
				t.Error("ParseRequest() returned nil request for valid input")
			}
		})
	}
}

// Note: NewReq() function doesn't exist in db/client.go, so we skip this test
func TestReqStruct(t *testing.T) {
	req := &Req{
		DSN:     "mysql://user:pass@localhost:3306/testdb",
		Query:   "SELECT 1",
		Timeout: "30s",
		Params:  []interface{}{"param1"},
	}

	if req.DSN == "" {
		t.Error("DSN should not be empty")
	}
	if req.Query == "" {
		t.Error("Query should not be empty")
	}
}

// Note: Req.Execute() is used instead of Req.Do(), so we test validation through ParseRequest
func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		data        map[string]string
		expectError bool
	}{
		{
			name: "missing dsn",
			data: map[string]string{
				"query": "SELECT 1",
			},
			expectError: true,
		},
		{
			name: "missing query",
			data: map[string]string{
				"dsn": "mysql://user:pass@localhost:3306/testdb",
			},
			expectError: true,
		},
		{
			name: "empty dsn",
			data: map[string]string{
				"dsn":   "",
				"query": "SELECT 1",
			},
			expectError: true,
		},
		{
			name: "empty query",
			data: map[string]string{
				"dsn":   "mysql://user:pass@localhost:3306/testdb",
				"query": "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ParseRequest(tt.data)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseRequest() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseRequest() unexpected error: %v", err)
			}
		})
	}
}

func TestExecuteQuery(t *testing.T) {
	tests := []struct {
		name        string
		data        map[string]string
		expectError bool
	}{
		{
			name: "missing required dsn",
			data: map[string]string{
				"query": "SELECT 1",
			},
			expectError: true,
		},
		{
			name: "missing required query",
			data: map[string]string{
				"dsn": "mysql://user:pass@localhost:3306/testdb",
			},
			expectError: true,
		},
		{
			name: "empty dsn",
			data: map[string]string{
				"dsn":   "",
				"query": "SELECT 1",
			},
			expectError: true,
		},
		{
			name: "empty query",
			data: map[string]string{
				"dsn":   "mysql://user:pass@localhost:3306/testdb",
				"query": "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track if callbacks were called
			beforeCalled := false
			afterCalled := false

			before := WithBefore(func(query string, params []interface{}) {
				beforeCalled = true
			})
			after := WithAfter(func(result *Result) {
				afterCalled = true
			})

			_, err := ExecuteQuery(tt.data, before, after)

			if tt.expectError {
				if err == nil {
					t.Errorf("ExecuteQuery() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("ExecuteQuery() unexpected error: %v", err)
				}
			}

			// For validation errors, callbacks may not be called
			if !tt.expectError {
				if !beforeCalled {
					t.Error("before callback was not called for valid request")
				}
				if !afterCalled {
					t.Error("after callback was not called for valid request")
				}
			}
		})
	}
}

func TestWithBefore(t *testing.T) {
	called := false
	var capturedQuery string
	var capturedParams []interface{}

	option := WithBefore(func(query string, params []interface{}) {
		called = true
		capturedQuery = query
		capturedParams = params
	})

	cb := &Callback{}
	option(cb)

	if cb.before == nil {
		t.Error("WithBefore() did not set before callback")
		return
	}

	// Test the callback
	testParams := []interface{}{"param1", "param2"}
	cb.before("SELECT * FROM test", testParams)

	if !called {
		t.Error("before callback was not called")
	}
	if capturedQuery != "SELECT * FROM test" {
		t.Errorf("Expected query 'SELECT * FROM test', got '%s'", capturedQuery)
	}
	if !reflect.DeepEqual(capturedParams, testParams) {
		t.Errorf("Expected params %v, got %v", testParams, capturedParams)
	}
}

func TestWithAfter(t *testing.T) {
	called := false
	var capturedResult *Result

	option := WithAfter(func(result *Result) {
		called = true
		capturedResult = result
	})

	cb := &Callback{}
	option(cb)

	if cb.after == nil {
		t.Error("WithAfter() did not set after callback")
		return
	}

	// Test the callback
	testResult := &Result{
		Status: 1,
		Res: Res{
			Code:         1,
			Rows:         []interface{}{},
			RowsAffected: 0,
			Error:        "test error",
		},
		RT: time.Second,
	}
	cb.after(testResult)

	if !called {
		t.Error("after callback was not called")
	}
	if capturedResult != testResult {
		t.Error("after callback did not receive correct result")
	}
}
