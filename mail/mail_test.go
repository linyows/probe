package mail

import (
	"bytes"
	"crypto/tls"
	"net"
	"net/smtp"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMail_New(t *testing.T) {
	mail := &Mail{
		Addr:         "localhost:25",
		MailFrom:     "sender@example.com",
		RcptTo:       []string{"recipient@example.com"},
		Data:         []byte("Subject: Test\n\nTest message"),
		MessageCount: 1,
	}

	if mail.Addr != "localhost:25" {
		t.Errorf("expected Addr to be 'localhost:25', got %s", mail.Addr)
	}
	if mail.MailFrom != "sender@example.com" {
		t.Errorf("expected MailFrom to be 'sender@example.com', got %s", mail.MailFrom)
	}
	if len(mail.RcptTo) != 1 || mail.RcptTo[0] != "recipient@example.com" {
		t.Errorf("expected RcptTo to contain 'recipient@example.com', got %v", mail.RcptTo)
	}
}

func TestGetFQDN(t *testing.T) {
	tests := []struct {
		name       string
		envValue   string
		setEnv     bool
		wantPrefix string
		wantEmpty  bool
	}{
		{
			name:       "with FQDN_DOMAIN env var",
			envValue:   "test.example.com",
			setEnv:     true,
			wantPrefix: "test.example.com",
			wantEmpty:  false,
		},
		{
			name:       "without FQDN_DOMAIN env var",
			setEnv:     false,
			wantEmpty:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv("FQDN_DOMAIN", tt.envValue)
				defer os.Unsetenv("FQDN_DOMAIN")
			} else {
				os.Unsetenv("FQDN_DOMAIN")
			}

			fqdn := getFQDN()

			if tt.wantEmpty && fqdn == "" {
				t.Error("expected FQDN to be non-empty")
			}
			if tt.wantPrefix != "" && fqdn != tt.wantPrefix {
				t.Errorf("expected FQDN to be '%s', got %s", tt.wantPrefix, fqdn)
			}
			if !tt.wantEmpty && fqdn == "" {
				t.Error("expected FQDN to be non-empty")
			}
		})
	}
}

func TestGenMsgID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		wantPrefix string
		wantSuffix string
	}{
		{
			name:       "normal id",
			id:         "test123",
			wantPrefix: "<test123@",
			wantSuffix: ">",
		},
		{
			name:       "empty id",
			id:         "",
			wantPrefix: "<@",
			wantSuffix: ">",
		},
		{
			name:       "special characters",
			id:         "abc-123_def",
			wantPrefix: "<abc-123_def@",
			wantSuffix: ">",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgID := genMsgID(tt.id)

			if !strings.HasPrefix(msgID, tt.wantPrefix) {
				t.Errorf("expected Message-ID to start with '%s', got %s", tt.wantPrefix, msgID)
			}
			if !strings.HasSuffix(msgID, tt.wantSuffix) {
				t.Errorf("expected Message-ID to end with '%s', got %s", tt.wantSuffix, msgID)
			}
		})
	}
}

func TestMail_AppendIDtoSubject(t *testing.T) {
	tests := []struct {
		name               string
		originalData       []byte
		wantMsgIDAtStart   bool
		wantSubjectModified bool
		wantDataUnchanged  bool
	}{
		{
			name:               "with subject header",
			originalData:       []byte("Subject: Original Subject\nFrom: sender@example.com\n\nBody content"),
			wantMsgIDAtStart:   true,
			wantSubjectModified: true,
			wantDataUnchanged:  false,
		},
		{
			name:               "without subject header",
			originalData:       []byte("From: sender@example.com\n\nBody content"),
			wantMsgIDAtStart:   true,
			wantSubjectModified: false,
			wantDataUnchanged:  true,
		},
		{
			name:               "empty data",
			originalData:       []byte(""),
			wantMsgIDAtStart:   true,
			wantSubjectModified: false,
			wantDataUnchanged:  true,
		},
		{
			name:               "subject at end",
			originalData:       []byte("From: sender@example.com\nSubject: Test Subject\n\nBody"),
			wantMsgIDAtStart:   true,
			wantSubjectModified: true,
			wantDataUnchanged:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mail := &Mail{}
			result := mail.appendIDtoSubject(tt.originalData)
			lines := bytes.Split(result, []byte("\n"))

			// Check if Message-ID is added at the beginning
			if tt.wantMsgIDAtStart {
				if len(lines) == 0 || !bytes.HasPrefix(lines[0], []byte("Message-ID: <")) {
					t.Error("expected Message-ID to be added at the beginning")
				}
			}

			// Check if subject is modified
			if tt.wantSubjectModified {
				found := false
				for _, line := range lines {
					if bytes.Contains(line, []byte("Subject:")) && bytes.Contains(line, []byte(" - ")) {
						found = true
						break
					}
				}
				if !found {
					t.Error("expected Subject to be modified with ID")
				}
			}

			// Check if original data remains unchanged (except Message-ID)
			if tt.wantDataUnchanged && len(lines) > 1 {
				resultWithoutMsgID := bytes.Join(lines[1:], []byte("\n"))
				if !bytes.Equal(resultWithoutMsgID, tt.originalData) {
					t.Error("expected original data to remain unchanged when no Subject modification needed")
				}
			}
		})
	}
}

func TestMail_Send_Validation(t *testing.T) {
	tests := []struct {
		name           string
		mail           *Mail
		wantError      bool
		wantErrorMsg   string
	}{
		{
			name: "invalid MailFrom with newline",
			mail: &Mail{
				MailFrom: "invalid\nemail@example.com",
				RcptTo:   []string{"recipient@example.com"},
				Data:     []byte("Subject: Test\n\nTest message"),
			},
			wantError:    true,
			wantErrorMsg: "A line must not contain CR or LF",
		},
		{
			name: "invalid MailFrom with carriage return",
			mail: &Mail{
				MailFrom: "invalid\remail@example.com",
				RcptTo:   []string{"recipient@example.com"},
				Data:     []byte("Subject: Test\n\nTest message"),
			},
			wantError:    true,
			wantErrorMsg: "A line must not contain CR or LF",
		},
		{
			name: "invalid RcptTo with carriage return",
			mail: &Mail{
				MailFrom: "sender@example.com",
				RcptTo:   []string{"invalid\rrecipient@example.com"},
				Data:     []byte("Subject: Test\n\nTest message"),
			},
			wantError:    true,
			wantErrorMsg: "smtp rcptto validate error",
		},
		{
			name: "invalid RcptTo with newline",
			mail: &Mail{
				MailFrom: "sender@example.com",
				RcptTo:   []string{"invalid\nrecipient@example.com"},
				Data:     []byte("Subject: Test\n\nTest message"),
			},
			wantError:    true,
			wantErrorMsg: "smtp rcptto validate error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mail.Send()

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.wantErrorMsg) {
					t.Errorf("expected error message to contain '%s', got %s", tt.wantErrorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestMail_Send_Success(t *testing.T) {
	tests := []struct {
		name         string
		mail         *Mail
		wantError    bool
	}{
		{
			name: "successful single message",
			mail: &Mail{
				Addr:             "localhost:0", // Will be replaced with actual mock server port
				MailFrom:         "sender@example.com",
				RcptTo:           []string{"recipient@example.com"},
				Data:             []byte("Subject: Test\n\nTest message"),
				StartTLSDisabled: true,
				MessageCount:     1,
			},
			wantError: false,
		},
		{
			name: "successful multiple messages",
			mail: &Mail{
				Addr:             "localhost:0", // Will be replaced with actual mock server port
				MailFrom:         "sender@example.com",
				RcptTo:           []string{"recipient@example.com"},
				Data:             []byte("Subject: Test Multiple\n\nTest message"),
				StartTLSDisabled: true,
				MessageCount:     3,
			},
			wantError: false,
		},
		{
			name: "successful multiple recipients",
			mail: &Mail{
				Addr:             "localhost:0", // Will be replaced with actual mock server port
				MailFrom:         "sender@example.com",
				RcptTo:           []string{"recipient1@example.com", "recipient2@example.com"},
				Data:             []byte("Subject: Test Multiple Recipients\n\nTest message"),
				StartTLSDisabled: true,
				MessageCount:     1,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start mock SMTP server
			mockServer := &MockServer{
				Addr: "localhost:0",
				Name: "test.example.com",
			}

			// Create a listener to get a free port
			listener, err := net.Listen("tcp", "localhost:0")
			if err != nil {
				t.Fatalf("failed to create listener: %v", err)
			}
			addr := listener.Addr().String()
			listener.Close()

			mockServer.Addr = addr
			tt.mail.Addr = addr

			// Start server in goroutine
			go func() {
				if err := mockServer.Serve(); err != nil {
					t.Logf("mock server error: %v", err)
				}
			}()

			// Give server time to start
			time.Sleep(100 * time.Millisecond)

			// Run the test
			err = tt.mail.Send()

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestMail_Send_WithAuth(t *testing.T) {
	tests := []struct {
		name         string
		auth         smtp.Auth
		wantError    bool
		wantErrorMsg string
	}{
		{
			name:         "with auth but server doesn't support AUTH",
			auth:         smtp.PlainAuth("", "user", "pass", "test.example.com"),
			wantError:    true,
			wantErrorMsg: "server doesn't support AUTH",
		},
		{
			name:         "without auth",
			auth:         nil,
			wantError:    false,
			wantErrorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start mock SMTP server
			mockServer := &MockServer{
				Addr: "localhost:0",
				Name: "test.example.com",
			}

			// Create a listener to get a free port
			listener, err := net.Listen("tcp", "localhost:0")
			if err != nil {
				t.Fatalf("failed to create listener: %v", err)
			}
			addr := listener.Addr().String()
			listener.Close()

			mockServer.Addr = addr

			// Start server in goroutine
			go func() {
				if err := mockServer.Serve(); err != nil {
					t.Logf("mock server error: %v", err)
				}
			}()

			// Give server time to start
			time.Sleep(100 * time.Millisecond)

			mail := &Mail{
				Addr:             addr,
				MailFrom:         "sender@example.com",
				RcptTo:           []string{"recipient@example.com"},
				Data:             []byte("Subject: Test\n\nTest message"),
				Auth:             tt.auth,
				StartTLSDisabled: true,
				MessageCount:     1,
			}

			err = mail.Send()

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.wantErrorMsg) {
					t.Errorf("expected error message to contain '%s', got %s", tt.wantErrorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestMail_Send_NetworkErrors(t *testing.T) {
	tests := []struct {
		name         string
		addr         string
		wantErrorMsg string
	}{
		{
			name:         "connection refused",
			addr:         "localhost:9999", // Non-existent port
			wantErrorMsg: "tcp dial error",
		},
		{
			name:         "invalid address",
			addr:         "invalid-host:25",
			wantErrorMsg: "tcp dial error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mail := &Mail{
				Addr:             tt.addr,
				MailFrom:         "sender@example.com",
				RcptTo:           []string{"recipient@example.com"},
				Data:             []byte("Subject: Test\n\nTest message"),
				StartTLSDisabled: true,
				MessageCount:     1,
			}

			err := mail.Send()
			if err == nil {
				t.Error("expected error but got none")
				return
			}
			if !strings.Contains(err.Error(), tt.wantErrorMsg) {
				t.Errorf("expected error message to contain '%s', got %s", tt.wantErrorMsg, err.Error())
			}
		})
	}
}

func TestMail_Send_StartTLS(t *testing.T) {
	tests := []struct {
		name             string
		startTLSDisabled bool
		wantError        bool
	}{
		{
			name:             "STARTTLS disabled",
			startTLSDisabled: true,
			wantError:        false,
		},
		{
			name:             "STARTTLS enabled", 
			startTLSDisabled: false,
			wantError:        true, // Will fail due to TLS cert issues in test
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start mock SMTP server
			mockServer := &MockServer{
				Addr: "localhost:0",
				Name: "test.example.com",
			}

			// Create a listener to get a free port
			listener, err := net.Listen("tcp", "localhost:0")
			if err != nil {
				t.Fatalf("failed to create listener: %v", err)
			}
			addr := listener.Addr().String()
			listener.Close()

			mockServer.Addr = addr

			// Start server in goroutine
			go func() {
				if err := mockServer.Serve(); err != nil {
					t.Logf("mock server error: %v", err)
				}
			}()

			// Give server time to start
			time.Sleep(100 * time.Millisecond)

			mail := &Mail{
				Addr:             addr,
				MailFrom:         "sender@example.com",
				RcptTo:           []string{"recipient@example.com"},
				Data:             []byte("Subject: Test\n\nTest message"),
				StartTLSDisabled: tt.startTLSDisabled,
				MessageCount:     1,
			}

			err = mail.Send()

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestMail_Send_StartTLSHook(t *testing.T) {
	originalHook := testHookStartTLS
	defer func() { testHookStartTLS = originalHook }()

	hookCalled := false
	testHookStartTLS = func(config *tls.Config) {
		hookCalled = true
		if config == nil {
			t.Error("expected TLS config to be non-nil")
		}
		// Verify ServerName is set (will be IP address from listener)
		if config.ServerName == "" {
			t.Error("expected ServerName to be non-empty")
		}
	}

	// Start mock SMTP server
	mockServer := &MockServer{
		Addr: "localhost:0",
		Name: "test.example.com",
	}

	// Create a listener to get a free port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	mockServer.Addr = addr

	// Start server in goroutine
	go func() {
		if err := mockServer.Serve(); err != nil {
			t.Logf("mock server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	mail := &Mail{
		Addr:             addr,
		MailFrom:         "sender@example.com", 
		RcptTo:           []string{"recipient@example.com"},
		Data:             []byte("Subject: Test\n\nTest message"),
		StartTLSDisabled: false, // Enable STARTTLS to trigger hook
		MessageCount:     1,
	}

	// This will fail due to certificate issues, but hook should be called
	mail.Send()

	if !hookCalled {
		t.Error("expected testHookStartTLS to be called")
	}
}