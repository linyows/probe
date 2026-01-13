package mail

import (
	"io"
	"log"
	"net"
	"testing"
	"time"
)

// setupMockServer creates and starts a mock SMTP server for testing
func setupMockServer(t *testing.T) (*MockServer, string) {
	t.Helper()

	mockServer := &MockServer{
		Addr: "localhost:0",
		Name: "test.example.com",
		Log:  log.New(io.Discard, "", 0), // Disable logging for tests
	}

	// Get available port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to get available port: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	mockServer.Addr = addr

	// Start server in goroutine
	go func() {
		if err := mockServer.Serve(); err != nil {
			t.Logf("mock server error: %v", err)
		}
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	return mockServer, addr
}

func TestBulkDeliverWithResult(t *testing.T) {
	tests := []struct {
		name            string
		session         int
		message         int
		expectedSent    int
		expectedSessions int
	}{
		{
			name:            "Single session, single message",
			session:         1,
			message:         1,
			expectedSent:    1,
			expectedSessions: 1,
		},
		{
			name:            "Single session, multiple messages",
			session:         1,
			message:         5,
			expectedSent:    5,
			expectedSessions: 1,
		},
		{
			name:            "Multiple sessions, multiple messages",
			session:         3,
			message:         10,
			expectedSent:    10,
			expectedSessions: 3,
		},
		{
			name:            "Sessions equal to messages",
			session:         5,
			message:         5,
			expectedSent:    5,
			expectedSessions: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start mock SMTP server
			_, addr := setupMockServer(t)

			bulk := &Bulk{
				Addr:       addr,
				From:       "test@example.com",
				To:         "recipient@example.com",
				Subject:    "Test",
				MyHostname: "test.local",
				Session:    tt.session,
				Message:    tt.message,
				Length:     100,
			}

			result := bulk.DeliverWithResult()

			// Verify sent message count
			if result.Sent != tt.expectedSent {
				t.Errorf("Expected Sent = %d, got %d", tt.expectedSent, result.Sent)
			}

			// Verify session count
			if result.Sessions != tt.expectedSessions {
				t.Errorf("Expected Sessions = %d, got %d", tt.expectedSessions, result.Sessions)
			}

			// Verify total equals sent (when no failures)
			if result.Total != tt.expectedSent {
				t.Errorf("Expected Total = %d, got %d", tt.expectedSent, result.Total)
			}

			// Verify no failures
			if result.Failed != 0 {
				t.Errorf("Expected Failed = 0, got %d", result.Failed)
			}

			// Verify no error
			if result.Error != "" {
				t.Errorf("Expected no error, got: %s", result.Error)
			}
		})
	}
}

func TestBulkSend(t *testing.T) {
	tests := []struct {
		name          string
		session       int
		message       int
		expectedCount int
	}{
		{
			name:          "Single message per session",
			session:       1,
			message:       1,
			expectedCount: 1,
		},
		{
			name:          "Multiple messages per session",
			session:       1,
			message:       5,
			expectedCount: 5,
		},
		{
			name:          "Evenly distributed messages",
			session:       3,
			message:       9,
			expectedCount: 3, // Each session sends 3 messages
		},
		{
			name:          "Unevenly distributed messages",
			session:       3,
			message:       10,
			expectedCount: 4, // First session sends 4, others send 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start mock SMTP server
			_, addr := setupMockServer(t)

			bulk := &Bulk{
				Addr:       addr,
				From:       "test@example.com",
				To:         "recipient@example.com",
				Subject:    "Test",
				MyHostname: "test.local",
				Session:    tt.session,
				Message:    tt.message,
				Length:     100,
			}

			count, err := bulk.Send()

			// Verify no error
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			// Verify count is within expected range
			// For the first goroutine, it should return the expected count
			// Note: This test only verifies a single Send() call
			if count < 1 || count > tt.expectedCount {
				t.Errorf("Expected count between 1 and %d, got %d", tt.expectedCount, count)
			}
		})
	}
}

