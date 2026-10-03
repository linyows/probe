package imap

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSearchCriteria(t *testing.T) {
	r := NewReq()

	tests := []struct {
		name     string
		criteria Criteria
		want     func(*testing.T, *imap.SearchCriteria)
		wantErr  bool
	}{
		{
			name:     "empty criteria",
			criteria: Criteria{},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				assert.Empty(t, sc.SeqNum)
				assert.Empty(t, sc.UID)
				assert.Empty(t, sc.Flag)
				assert.Empty(t, sc.NotFlag)
				assert.Empty(t, sc.Header)
				assert.Empty(t, sc.Body)
				assert.Empty(t, sc.Text)
				assert.True(t, sc.Since.IsZero())
				assert.True(t, sc.Before.IsZero())
			},
			wantErr: false,
		},
		{
			name: "seq nums - single number",
			criteria: Criteria{
				SeqNums: []string{"42"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.SeqNum, 1)
				// SeqSetは内部実装のため、詳細チェックは省略
			},
			wantErr: false,
		},
		{
			name: "seq nums - range",
			criteria: Criteria{
				SeqNums: []string{"1:10"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.SeqNum, 1)
			},
			wantErr: false,
		},
		{
			name: "seq nums - all",
			criteria: Criteria{
				SeqNums: []string{"*"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.SeqNum, 1)
			},
			wantErr: false,
		},
		{
			name: "seq nums - range with asterisk",
			criteria: Criteria{
				SeqNums: []string{"100:*"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.SeqNum, 1)
			},
			wantErr: false,
		},
		{
			name: "UIDs - single UID",
			criteria: Criteria{
				UIDs: []string{"1000"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.UID, 1)
			},
			wantErr: false,
		},
		{
			name: "UIDs - range",
			criteria: Criteria{
				UIDs: []string{"1000:2000"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.UID, 1)
			},
			wantErr: false,
		},
		{
			name: "since - today",
			criteria: Criteria{
				Since: "today",
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				today := time.Now().Truncate(24 * time.Hour)
				assert.Equal(t, today, sc.Since)
			},
			wantErr: false,
		},
		{
			name: "since - yesterday",
			criteria: Criteria{
				Since: "yesterday",
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				yesterday := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
				assert.Equal(t, yesterday, sc.Since)
			},
			wantErr: false,
		},
		{
			name: "since - 2 hours ago",
			criteria: Criteria{
				Since: "2 hours ago",
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				assert.False(t, sc.Since.IsZero())
				// 時間の精密なテストは省略（実際の実装に依存）
			},
			wantErr: false,
		},
		{
			name: "since - RFC3339 format",
			criteria: Criteria{
				Since: "2023-12-01T10:00:00Z",
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				expected, _ := time.Parse(time.RFC3339, "2023-12-01T10:00:00Z")
				assert.Equal(t, expected, sc.Since)
			},
			wantErr: false,
		},
		{
			name: "since - invalid date",
			criteria: Criteria{
				Since: "invalid-date",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "before - date format",
			criteria: Criteria{
				Before: "2023-12-31",
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				expected, _ := time.Parse("2006-01-02", "2023-12-31")
				assert.Equal(t, expected, sc.Before)
			},
			wantErr: false,
		},
		{
			name: "sent_since - date format",
			criteria: Criteria{
				SentSince: "2023-01-01",
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				expected, _ := time.Parse("2006-01-02", "2023-01-01")
				assert.Equal(t, expected, sc.SentSince)
			},
			wantErr: false,
		},
		{
			name: "sent_before - date format",
			criteria: Criteria{
				SentBefore: "2023-12-31",
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				expected, _ := time.Parse("2006-01-02", "2023-12-31")
				assert.Equal(t, expected, sc.SentBefore)
			},
			wantErr: false,
		},
		{
			name: "headers - single header",
			criteria: Criteria{
				Headers: map[string]string{
					"From": "test@example.com",
				},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.Header, 1)
				assert.Equal(t, "From", sc.Header[0].Key)
				assert.Equal(t, "test@example.com", sc.Header[0].Value)
			},
			wantErr: false,
		},
		{
			name: "headers - multiple headers",
			criteria: Criteria{
				Headers: map[string]string{
					"From":    "test@example.com",
					"Subject": "Test Email",
				},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				assert.Len(t, sc.Header, 2)
			},
			wantErr: false,
		},
		{
			name: "bodies - single body text",
			criteria: Criteria{
				Bodies: []string{"important message"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.Body, 1)
				assert.Equal(t, "important message", sc.Body[0])
			},
			wantErr: false,
		},
		{
			name: "bodies - multiple body texts",
			criteria: Criteria{
				Bodies: []string{"urgent", "action required"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				assert.Len(t, sc.Body, 2)
				assert.Contains(t, sc.Body, "urgent")
				assert.Contains(t, sc.Body, "action required")
			},
			wantErr: false,
		},
		{
			name: "texts - single text",
			criteria: Criteria{
				Texts: []string{"search text"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.Text, 1)
				assert.Equal(t, "search text", sc.Text[0])
			},
			wantErr: false,
		},
		{
			name: "flags - seen flag (lowercase)",
			criteria: Criteria{
				Flags: []string{"seen"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.Flag, 1)
				assert.Equal(t, imap.FlagSeen, sc.Flag[0])
			},
			wantErr: false,
		},
		{
			name: "flags - seen flag (uppercase)",
			criteria: Criteria{
				Flags: []string{"SEEN"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.Flag, 1)
				assert.Equal(t, imap.FlagSeen, sc.Flag[0])
			},
			wantErr: false,
		},
		{
			name: "flags - seen flag (mixed case)",
			criteria: Criteria{
				Flags: []string{"Seen"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.Flag, 1)
				assert.Equal(t, imap.FlagSeen, sc.Flag[0])
			},
			wantErr: false,
		},
		{
			name: "flags - backslash prefixed flag",
			criteria: Criteria{
				Flags: []string{"\\Answered"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.Flag, 1)
				assert.Equal(t, imap.Flag("\\Answered"), sc.Flag[0])
			},
			wantErr: false,
		},
		{
			name: "flags - custom flag without backslash",
			criteria: Criteria{
				Flags: []string{"Custom"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.Flag, 1)
				assert.Equal(t, imap.Flag("\\Custom"), sc.Flag[0])
			},
			wantErr: false,
		},
		{
			name: "flags - multiple flags",
			criteria: Criteria{
				Flags: []string{"seen", "\\Answered", "Important"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				assert.Len(t, sc.Flag, 3)
				assert.Contains(t, sc.Flag, imap.FlagSeen)
				assert.Contains(t, sc.Flag, imap.Flag("\\Answered"))
				assert.Contains(t, sc.Flag, imap.Flag("\\Important"))
			},
			wantErr: false,
		},
		{
			name: "not_flags - seen flag",
			criteria: Criteria{
				NotFlags: []string{"seen"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.NotFlag, 1)
				assert.Equal(t, imap.FlagSeen, sc.NotFlag[0])
			},
			wantErr: false,
		},
		{
			name: "not_flags - custom flag",
			criteria: Criteria{
				NotFlags: []string{"\\Draft"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.NotFlag, 1)
				assert.Equal(t, imap.Flag("\\Draft"), sc.NotFlag[0])
			},
			wantErr: false,
		},
		{
			name: "complex criteria - multiple conditions",
			criteria: Criteria{
				Since: "today",
				Flags: []string{"seen"},
				Headers: map[string]string{
					"From": "noreply@example.com",
				},
				Bodies: []string{"verification code"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				// Since
				today := time.Now().Truncate(24 * time.Hour)
				assert.Equal(t, today, sc.Since)

				// Flags
				require.Len(t, sc.Flag, 1)
				assert.Equal(t, imap.FlagSeen, sc.Flag[0])

				// Headers
				require.Len(t, sc.Header, 1)
				assert.Equal(t, "From", sc.Header[0].Key)
				assert.Equal(t, "noreply@example.com", sc.Header[0].Value)

				// Bodies
				require.Len(t, sc.Body, 1)
				assert.Equal(t, "verification code", sc.Body[0])
			},
			wantErr: false,
		},
		{
			name: "imap.yml example - search for seen emails",
			criteria: Criteria{
				Flags: []string{"seen"},
			},
			want: func(t *testing.T, sc *imap.SearchCriteria) {
				require.Len(t, sc.Flag, 1)
				assert.Equal(t, imap.FlagSeen, sc.Flag[0])
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.buildSearchCriteria(tt.criteria)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			if tt.want != nil {
				tt.want(t, got)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	r := NewReq()

	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, time.Time)
	}{
		{
			name:  "today",
			input: "today",
			check: func(t *testing.T, result time.Time) {
				today := time.Now().Truncate(24 * time.Hour)
				assert.Equal(t, today, result)
			},
		},
		{
			name:  "yesterday",
			input: "yesterday",
			check: func(t *testing.T, result time.Time) {
				yesterday := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
				assert.Equal(t, yesterday, result)
			},
		},
		{
			name:  "2 hours ago",
			input: "2 hours ago",
			check: func(t *testing.T, result time.Time) {
				assert.False(t, result.IsZero())
			},
		},
		{
			name:  "30 minutes ago",
			input: "30 minutes ago",
			check: func(t *testing.T, result time.Time) {
				assert.False(t, result.IsZero())
			},
		},
		{
			name:  "RFC3339 format",
			input: "2023-12-01T10:00:00Z",
			check: func(t *testing.T, result time.Time) {
				expected, _ := time.Parse(time.RFC3339, "2023-12-01T10:00:00Z")
				assert.Equal(t, expected, result)
			},
		},
		{
			name:  "YYYY-MM-DD format",
			input: "2023-12-01",
			check: func(t *testing.T, result time.Time) {
				expected, _ := time.Parse("2006-01-02", "2023-12-01")
				assert.Equal(t, expected, result)
			},
		},
		{
			name:    "invalid format",
			input:   "invalid-date-format",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.parseDate(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestParseFetchItemsBodySection(t *testing.T) {
	tests := []struct {
		name     string
		dataitem string
		wantErr  bool
		validate func(t *testing.T, opts *imap.FetchOptions)
	}{
		{
			name:     "BODY[] - full message body",
			dataitem: "BODY[]",
			wantErr:  false,
			validate: func(t *testing.T, opts *imap.FetchOptions) {
				if len(opts.BodySection) == 0 {
					t.Error("Expected BodySection to be set")
				}
				if opts.BodySection[0].Peek {
					t.Error("Expected Peek to be false for BODY[]")
				}
			},
		},
		{
			name:     "BODY.PEEK[] - full message body without setting seen flag",
			dataitem: "BODY.PEEK[]",
			wantErr:  false,
			validate: func(t *testing.T, opts *imap.FetchOptions) {
				if len(opts.BodySection) == 0 {
					t.Error("Expected BodySection to be set")
				}
				if !opts.BodySection[0].Peek {
					t.Error("Expected Peek to be true for BODY.PEEK[]")
				}
			},
		},
		{
			name:     "BODY[HEADER] - message headers only",
			dataitem: "BODY[HEADER]",
			wantErr:  false,
			validate: func(t *testing.T, opts *imap.FetchOptions) {
				if len(opts.BodySection) == 0 {
					t.Error("Expected BodySection to be set")
				}
				if opts.BodySection[0].Specifier != imap.PartSpecifierHeader {
					t.Error("Expected Specifier to be PartSpecifierHeader")
				}
			},
		},
		{
			name:     "BODY[TEXT] - message text only",
			dataitem: "BODY[TEXT]",
			wantErr:  false,
			validate: func(t *testing.T, opts *imap.FetchOptions) {
				if len(opts.BodySection) == 0 {
					t.Error("Expected BodySection to be set")
				}
				if opts.BodySection[0].Specifier != imap.PartSpecifierText {
					t.Error("Expected Specifier to be PartSpecifierText")
				}
			},
		},
		{
			name:     "BODY[1] - specific message part",
			dataitem: "BODY[1]",
			wantErr:  false,
			validate: func(t *testing.T, opts *imap.FetchOptions) {
				if len(opts.BodySection) == 0 {
					t.Error("Expected BodySection to be set")
				}
				if len(opts.BodySection[0].Part) != 1 || opts.BodySection[0].Part[0] != 1 {
					t.Errorf("Expected Part to be [1], got %v", opts.BodySection[0].Part)
				}
			},
		},
		{
			name:     "BODY[1.2.HEADER] - specific part header",
			dataitem: "BODY[1.2.HEADER]",
			wantErr:  false,
			validate: func(t *testing.T, opts *imap.FetchOptions) {
				if len(opts.BodySection) == 0 {
					t.Error("Expected BodySection to be set")
				}
				expectedPart := []int{1, 2}
				if len(opts.BodySection[0].Part) != 2 || opts.BodySection[0].Part[0] != 1 || opts.BodySection[0].Part[1] != 2 {
					t.Errorf("Expected Part to be %v, got %v", expectedPart, opts.BodySection[0].Part)
				}
				if opts.BodySection[0].Specifier != imap.PartSpecifierHeader {
					t.Error("Expected Specifier to be PartSpecifierHeader")
				}
			},
		},
		{
			name:     "BODY[HEADER.FIELDS (FROM TO)] - specific header fields",
			dataitem: "BODY[HEADER.FIELDS (FROM TO)]",
			wantErr:  false,
			validate: func(t *testing.T, opts *imap.FetchOptions) {
				if len(opts.BodySection) == 0 {
					t.Error("Expected BodySection to be set")
				}
				expectedFields := []string{"FROM", "TO"}
				if len(opts.BodySection[0].HeaderFields) != 2 {
					t.Errorf("Expected HeaderFields length to be 2, got %d", len(opts.BodySection[0].HeaderFields))
				}
				for i, field := range expectedFields {
					if opts.BodySection[0].HeaderFields[i] != field {
						t.Errorf("Expected HeaderFields[%d] to be %s, got %s", i, field, opts.BodySection[0].HeaderFields[i])
					}
				}
				// Verify that Specifier is set to Header for HEADER.FIELDS
				if opts.BodySection[0].Specifier != imap.PartSpecifierHeader {
					t.Error("Expected Specifier to be PartSpecifierHeader for HEADER.FIELDS")
				}
			},
		},
		{
			name:     "BODY[HEADER.FIELDS (FROM)] - single header field",
			dataitem: "BODY[HEADER.FIELDS (FROM)]",
			wantErr:  false,
			validate: func(t *testing.T, opts *imap.FetchOptions) {
				if len(opts.BodySection) == 0 {
					t.Error("Expected BodySection to be set")
				}
				if len(opts.BodySection[0].HeaderFields) != 1 {
					t.Errorf("Expected HeaderFields length to be 1, got %d", len(opts.BodySection[0].HeaderFields))
				}
				if opts.BodySection[0].HeaderFields[0] != "FROM" {
					t.Errorf("Expected HeaderFields[0] to be FROM, got %s", opts.BodySection[0].HeaderFields[0])
				}
				// Verify that Specifier is set to Header for HEADER.FIELDS
				if opts.BodySection[0].Specifier != imap.PartSpecifierHeader {
					t.Error("Expected Specifier to be PartSpecifierHeader for HEADER.FIELDS")
				}
			},
		},
		{
			name:     "Multi-item fetch with BODY section",
			dataitem: "FLAGS UID BODY[HEADER]",
			wantErr:  false,
			validate: func(t *testing.T, opts *imap.FetchOptions) {
				if !opts.Flags {
					t.Error("Expected Flags to be true")
				}
				if !opts.UID {
					t.Error("Expected UID to be true")
				}
				if len(opts.BodySection) == 0 {
					t.Error("Expected BodySection to be set")
				}
				if opts.BodySection[0].Specifier != imap.PartSpecifierHeader {
					t.Error("Expected Specifier to be PartSpecifierHeader")
				}
			},
		},
	}

	r := NewReq()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := r.parseFetchItems(tt.dataitem)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFetchItems() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, opts)
			}
		})
	}
}

func TestParseBodySection(t *testing.T) {
	tests := []struct {
		name        string
		bodyItem    string
		wantErr     bool
		expectedErr string
		validate    func(t *testing.T, section *imap.FetchItemBodySection)
	}{
		{
			name:     "BODY[] - empty section",
			bodyItem: "BODY[]",
			wantErr:  false,
			validate: func(t *testing.T, section *imap.FetchItemBodySection) {
				if section.Peek {
					t.Error("Expected Peek to be false")
				}
			},
		},
		{
			name:     "BODY.PEEK[] - empty section with peek",
			bodyItem: "BODY.PEEK[]",
			wantErr:  false,
			validate: func(t *testing.T, section *imap.FetchItemBodySection) {
				if !section.Peek {
					t.Error("Expected Peek to be true")
				}
			},
		},
		{
			name:        "Invalid format",
			bodyItem:    "INVALID[SECTION]",
			wantErr:     true,
			expectedErr: "invalid BODY section format",
		},
	}

	r := NewReq()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section, err := r.parseBodySection(tt.bodyItem)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBodySection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.expectedErr != "" {
				if err == nil || err.Error()[:len(tt.expectedErr)] != tt.expectedErr {
					t.Errorf("Expected error to start with '%s', got '%v'", tt.expectedErr, err)
				}
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, section)
			}
		})
	}
}

func TestParseHeaderData(t *testing.T) {
	tests := []struct {
		name       string
		headerData string
		expected   map[string]string
	}{
		{
			name:       "Single header",
			headerData: "From: test@example.com",
			expected: map[string]string{
				"from": "test@example.com",
			},
		},
		{
			name:       "Multiple headers",
			headerData: "From: test@example.com\nTo: recipient@example.com\nSubject: Test Subject",
			expected: map[string]string{
				"from":    "test@example.com",
				"to":      "recipient@example.com",
				"subject": "Test Subject",
			},
		},
		{
			name:       "Header with continuation line",
			headerData: "Subject: This is a very long subject line\r\n that continues on the next line",
			expected: map[string]string{
				"subject": "This is a very long subject line that continues on the next line",
			},
		},
		{
			name:       "HEADER.FIELDS response format (FROM only)",
			headerData: "From: test@example.com\r\n\r\n",
			expected: map[string]string{
				"from": "test@example.com",
			},
		},
		{
			name:       "Empty header data",
			headerData: "",
			expected:   map[string]string{},
		},
	}

	r := NewReq()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.parseHeaderData(tt.headerData)

			// Check if all expected headers are present
			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Expected header %s not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("Header %s: expected %s, got %s", key, expectedValue, actualValue)
				}
			}

			// Check if there are any unexpected headers
			for key := range result {
				if _, expected := tt.expected[key]; !expected {
					t.Errorf("Unexpected header %s found: %s", key, result[key])
				}
			}
		})
	}
}

// startIMAPServer runs an in-memory IMAP server for user/pass with an INBOX.
// With useTLS it listens with a freshly made self-signed certificate, the
// kind a test or staging mail server often has.
func startIMAPServer(t *testing.T, useTLS bool) (host string, port int, user *imapmemserver.User) {
	t.Helper()

	mem := imapmemserver.New()
	user = imapmemserver.NewUser("user", "pass")
	if err := user.Create("INBOX", nil); err != nil {
		t.Fatal(err)
	}
	mem.AddUser(user)

	srv := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}},
		InsecureAuth: !useTLS,
	})

	var ln net.Listener
	var err error
	if useTLS {
		ln, err = tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{selfSignedCert(t)}})
	} else {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	addr := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", addr.Port, user
}

func selfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "imap.test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func TestRequest_InsecureSkipTLS(t *testing.T) {
	host, port, _ := startIMAPServer(t, true)

	request := func(skip any) (map[string]any, error) {
		return Request(map[string]any{
			"host":              host,
			"port":              port,
			"username":          "user",
			"password":          "pass",
			"tls":               true,
			"insecure_skip_tls": skip,
			"commands":          []any{map[string]any{"name": "select", "mailbox": "INBOX"}},
		})
	}

	for _, skip := range []any{true, "true"} {
		if _, err := request(skip); err != nil {
			t.Errorf("insecure_skip_tls=%v should accept a self-signed certificate: %v", skip, err)
		}
	}

	_, err := request(false)
	if err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Errorf("without insecure_skip_tls the self-signed certificate must be refused, got %v", err)
	}
}

// plainIMAPServer runs an in-memory IMAP server without TLS, with INBOX
// holding n messages and an empty Archive, and returns its address.
func plainIMAPServer(t *testing.T, n int) (string, int) {
	t.Helper()

	mem := imapmemserver.New()
	user := imapmemserver.NewUser("user", "pass")
	for _, mb := range []string{"INBOX", "Archive"} {
		if err := user.Create(mb, nil); err != nil {
			t.Fatal(err)
		}
	}
	mem.AddUser(user)

	srv := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}, imap.CapUIDPlus: {}},
		InsecureAuth: true,
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	// Put the messages in through a real client, as a mail server would
	// receive them.
	c, err := imapclient.DialInsecure(ln.Addr().String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	if err := c.Login("user", "pass").Wait(); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= n; i++ {
		msg := fmt.Sprintf("From: a@example.test\r\nSubject: message %d\r\n\r\nbody %d\r\n", i, i)
		cmd := c.Append("INBOX", int64(len(msg)), nil)
		if _, err := cmd.Write([]byte(msg)); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := cmd.Wait(); err != nil {
			t.Fatal(err)
		}
	}

	addr := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", addr.Port
}

// runCommands runs the imap action's commands against the server.
func runCommands(t *testing.T, host string, port int, commands ...map[string]any) (map[string]any, error) {
	t.Helper()
	cmds := make([]any, len(commands))
	for i, c := range commands {
		cmds[i] = c
	}
	return Request(map[string]any{
		"host": host, "port": port, "username": "user", "password": "pass",
		"tls": false, "commands": cmds,
	})
}

func fetchedFlags(t *testing.T, ret map[string]any) [][]string {
	t.Helper()
	res := ret["res"].(map[string]any)
	data := res["data"].(map[string]any)
	fetch := data["fetch"].(map[string]any)
	var out [][]string
	for _, m := range fetch["messages"].([]any) {
		flags, _ := m.(map[string]any)["flags"].([]string)
		out = append(out, flags)
	}
	return out
}

func TestStore_ChangesFlags(t *testing.T) {
	host, port := plainIMAPServer(t, 3)

	ret, err := runCommands(t, host, port,
		map[string]any{"name": "select", "mailbox": "INBOX"},
		map[string]any{"name": "store", "sequence": "1:2", "dataitem": "+FLAGS", "value": `\Seen \Flagged`},
		map[string]any{"name": "store", "sequence": "2", "dataitem": "-FLAGS.SILENT", "value": `(\Flagged)`},
		map[string]any{"name": "fetch", "sequence": "1:3", "dataitem": "FLAGS"},
	)
	require.NoError(t, err)

	flags := fetchedFlags(t, ret)
	require.Len(t, flags, 3)
	assert.ElementsMatch(t, []string{`\Seen`, `\Flagged`}, flags[0], "message 1 got both flags")
	assert.ElementsMatch(t, []string{`\Seen`}, flags[1], "message 2 lost \\Flagged again")
	assert.Empty(t, flags[2], "message 3 was not in the set")
}

func TestStore_UIDAndReplace(t *testing.T) {
	host, port := plainIMAPServer(t, 2)

	ret, err := runCommands(t, host, port,
		map[string]any{"name": "select", "mailbox": "INBOX"},
		map[string]any{"name": "uid store", "sequence": "2", "dataitem": "+FLAGS", "value": `\Seen`},
		map[string]any{"name": "uid store", "sequence": "2", "dataitem": "FLAGS", "value": `\Answered`},
		map[string]any{"name": "fetch", "sequence": "1:2", "dataitem": "FLAGS"},
	)
	require.NoError(t, err)

	flags := fetchedFlags(t, ret)
	assert.Empty(t, flags[0])
	assert.ElementsMatch(t, []string{`\Answered`}, flags[1], "FLAGS replaces the set")

	store := ret["res"].(map[string]any)["data"].(map[string]any)["store"].(map[string]any)
	assert.Equal(t, 1, store["count"], "the server reported one message")
}

func TestStore_UsesLatestSearch(t *testing.T) {
	host, port := plainIMAPServer(t, 3)

	ret, err := runCommands(t, host, port,
		map[string]any{"name": "select", "mailbox": "INBOX"},
		map[string]any{"name": "search", "criteria": map[string]any{"headers": map[string]any{"Subject": "message 2"}}},
		map[string]any{"name": "store", "dataitem": "+FLAGS", "value": `\Deleted`},
		map[string]any{"name": "fetch", "sequence": "1:3", "dataitem": "FLAGS"},
	)
	require.NoError(t, err)

	flags := fetchedFlags(t, ret)
	assert.Empty(t, flags[0])
	assert.ElementsMatch(t, []string{`\Deleted`}, flags[1], "store without a sequence acts on the search result")
	assert.Empty(t, flags[2])
}

func TestCopy_CopiesMessages(t *testing.T) {
	host, port := plainIMAPServer(t, 3)

	ret, err := runCommands(t, host, port,
		map[string]any{"name": "select", "mailbox": "INBOX"},
		map[string]any{"name": "copy", "sequence": "1:2", "mailbox": "Archive"},
		map[string]any{"name": "examine", "mailbox": "Archive"},
	)
	require.NoError(t, err)

	data := ret["res"].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, 2, data["copy"].(map[string]any)["count"])
	assert.Equal(t, 2, data["examine"].(map[string]any)["exists"], "Archive now holds the copies")

	ret, err = runCommands(t, host, port,
		map[string]any{"name": "select", "mailbox": "INBOX"},
		map[string]any{"name": "uid copy", "sequence": "3", "mailbox": "Archive"},
		map[string]any{"name": "examine", "mailbox": "Archive"},
	)
	require.NoError(t, err)
	data = ret["res"].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, 3, data["examine"].(map[string]any)["exists"])
}

func TestStoreAndCopy_Errors(t *testing.T) {
	host, port := plainIMAPServer(t, 1)

	tests := []struct {
		name    string
		command map[string]any
		want    string
	}{
		{"unknown data item", map[string]any{"name": "store", "sequence": "1", "dataitem": "LABELS", "value": `\Seen`}, "unsupported STORE data item"},
		{"nothing to add", map[string]any{"name": "store", "sequence": "1", "dataitem": "+FLAGS", "value": ""}, "value must list the flags"},
		{"missing mailbox", map[string]any{"name": "copy", "sequence": "1", "mailbox": "Nowhere"}, "failed to Copy"},
	}
	// A failed command is not an action error: the step sees res.code 1 and
	// the reason in res.error, as it does for every other command.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ret, err := runCommands(t, host, port, map[string]any{"name": "select", "mailbox": "INBOX"}, tt.command)
			require.NoError(t, err)
			res := ret["res"].(map[string]any)
			assert.Equal(t, 1, res["code"])
			assert.Contains(t, res["error"], tt.want)
		})
	}
}

func TestParseStoreFlags(t *testing.T) {
	tests := []struct {
		dataitem, value string
		op              imap.StoreFlagsOp
		silent          bool
		flags           []imap.Flag
	}{
		{"FLAGS", `\Seen`, imap.StoreFlagsSet, false, []imap.Flag{imap.FlagSeen}},
		{"+flags", `(\Seen \Flagged)`, imap.StoreFlagsAdd, false, []imap.Flag{imap.FlagSeen, imap.FlagFlagged}},
		{"-FLAGS.SILENT", `\Deleted`, imap.StoreFlagsDel, true, []imap.Flag{imap.FlagDeleted}},
		{"FLAGS", "", imap.StoreFlagsSet, false, nil}, // clears every flag
	}
	for _, tt := range tests {
		got, err := parseStoreFlags(tt.dataitem, tt.value)
		require.NoError(t, err, tt.dataitem)
		assert.Equal(t, tt.op, got.Op, tt.dataitem)
		assert.Equal(t, tt.silent, got.Silent, tt.dataitem)
		assert.Equal(t, tt.flags, got.Flags, tt.dataitem)
	}
}

func TestParseNumRanges(t *testing.T) {
	tests := []struct {
		in      string
		want    []numRange
		wantErr bool
	}{
		{"*", []numRange{{0, 0}}, false}, // the last message, not 1:*
		{"1:3,5,9:*", []numRange{{1, 3}, {5, 5}, {9, 0}}, false},
		{" 2 ", []numRange{{2, 2}}, false},
		{"1,*", []numRange{{1, 1}, {0, 0}}, false},
		{"4294967295", []numRange{{4294967295, 4294967295}}, false},
		{"0", nil, true},          // would become "*"
		{"1:0", nil, true},        // would become 1:*, the whole mailbox
		{"-1", nil, true},         // would wrap to a huge number
		{"4294967296", nil, true}, // overflows 32 bits, wrapping to "*"
		{"1:4294967296", nil, true},
		{"*:5", nil, true},
		{"1,,2", nil, true},
		{"a", nil, true},
	}
	for _, tt := range tests {
		got, err := parseNumRanges(tt.in, "sequence number")
		if tt.wantErr {
			assert.Error(t, err, "%q", tt.in)
			continue
		}
		require.NoError(t, err, "%q", tt.in)
		assert.Equal(t, tt.want, got, "%q", tt.in)
	}
}

func TestStoreAndCopy_StarIsTheLastMessage(t *testing.T) {
	host, port := plainIMAPServer(t, 3)

	ret, err := runCommands(t, host, port,
		map[string]any{"name": "select", "mailbox": "INBOX"},
		map[string]any{"name": "store", "sequence": "*", "dataitem": "+FLAGS", "value": `\Flagged`},
		map[string]any{"name": "uid store", "sequence": "*", "dataitem": "+FLAGS", "value": `\Seen`},
		map[string]any{"name": "fetch", "sequence": "1:3", "dataitem": "FLAGS"},
	)
	require.NoError(t, err)
	flags := fetchedFlags(t, ret)
	assert.Empty(t, flags[0], "message 1 must be untouched")
	assert.Empty(t, flags[1], "message 2 must be untouched")
	assert.ElementsMatch(t, []string{`\Flagged`, `\Seen`}, flags[2], "only the last message")

	ret, err = runCommands(t, host, port,
		map[string]any{"name": "select", "mailbox": "INBOX"},
		map[string]any{"name": "copy", "sequence": "*", "mailbox": "Archive"},
		map[string]any{"name": "uid copy", "sequence": "*", "mailbox": "Archive"},
		map[string]any{"name": "examine", "mailbox": "Archive"},
	)
	require.NoError(t, err)
	data := ret["res"].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, 2, data["examine"].(map[string]any)["exists"], "one message per copy, not the whole mailbox")
}

func TestStore_RejectsOutOfRangeNumbers(t *testing.T) {
	host, port := plainIMAPServer(t, 2)

	for _, seq := range []string{"0", "1:0", "-1", "4294967296", "1:4294967296"} {
		ret, err := runCommands(t, host, port,
			map[string]any{"name": "select", "mailbox": "INBOX"},
			map[string]any{"name": "store", "sequence": seq, "dataitem": "+FLAGS", "value": `\Deleted`},
		)
		require.NoError(t, err, seq)
		res := ret["res"].(map[string]any)
		assert.Equal(t, 1, res["code"], "sequence %q must be refused", seq)
		assert.Contains(t, res["error"], "use a number from 1 to 4294967295", seq)
	}

	// And nothing was flagged on the way.
	ret, err := runCommands(t, host, port,
		map[string]any{"name": "select", "mailbox": "INBOX"},
		map[string]any{"name": "fetch", "sequence": "1:2", "dataitem": "FLAGS"},
	)
	require.NoError(t, err)
	for _, f := range fetchedFlags(t, ret) {
		assert.Empty(t, f)
	}
}

func TestStore_ForgetsSearchAfterSwitchingMailbox(t *testing.T) {
	host, port := plainIMAPServer(t, 3)

	// Put a message in Archive, so a stale INBOX result could hit it.
	_, err := runCommands(t, host, port,
		map[string]any{"name": "select", "mailbox": "INBOX"},
		map[string]any{"name": "copy", "sequence": "1:3", "mailbox": "Archive"},
	)
	require.NoError(t, err)

	for _, cmd := range []map[string]any{
		{"name": "store", "dataitem": "+FLAGS", "value": `\Deleted`},
		{"name": "uid store", "dataitem": "+FLAGS", "value": `\Deleted`},
		{"name": "copy", "mailbox": "INBOX"},
	} {
		ret, err := runCommands(t, host, port,
			map[string]any{"name": "select", "mailbox": "INBOX"},
			map[string]any{"name": "search", "criteria": map[string]any{"headers": map[string]any{"Subject": "message 2"}}},
			map[string]any{"name": "uid search", "criteria": map[string]any{"headers": map[string]any{"Subject": "message 2"}}},
			map[string]any{"name": "select", "mailbox": "Archive"},
			cmd,
		)
		require.NoError(t, err)
		res := ret["res"].(map[string]any)
		assert.Equal(t, 1, res["code"], "%s must not reuse the INBOX search in Archive", cmd["name"])
		assert.Contains(t, res["error"], "sequence is required", cmd["name"])
	}

	ret, err := runCommands(t, host, port,
		map[string]any{"name": "examine", "mailbox": "Archive"},
		map[string]any{"name": "fetch", "sequence": "1:3", "dataitem": "FLAGS"},
	)
	require.NoError(t, err)
	for _, f := range fetchedFlags(t, ret) {
		assert.Empty(t, f, "no message in Archive was touched")
	}
	assert.Equal(t, 3, ret["res"].(map[string]any)["data"].(map[string]any)["examine"].(map[string]any)["exists"])
}

// TestStore_UsesOnlyTheLatestSearch covers a search of one kind followed by
// one of the other kind with a different result: a command without a
// sequence must not fall back to the older search.
func TestStore_UsesOnlyTheLatestSearch(t *testing.T) {
	subject := func(s string) map[string]any {
		return map[string]any{"headers": map[string]any{"Subject": s}}
	}

	tests := []struct {
		name    string
		first   map[string]any
		second  map[string]any
		command map[string]any
		wantErr string
		flagged int // 1-based message flagged, 0 for none
	}{
		{
			name:    "store after search then uid search",
			first:   map[string]any{"name": "search", "criteria": subject("message 1")},
			second:  map[string]any{"name": "uid search", "criteria": subject("message 2")},
			command: map[string]any{"name": "store", "dataitem": "+FLAGS", "value": `\Flagged`},
			wantErr: "sequence is required",
		},
		{
			name:    "uid store after uid search then search",
			first:   map[string]any{"name": "uid search", "criteria": subject("message 1")},
			second:  map[string]any{"name": "search", "criteria": subject("message 2")},
			command: map[string]any{"name": "uid store", "dataitem": "+FLAGS", "value": `\Flagged`},
			wantErr: "sequence is required",
		},
		{
			name:    "uid store after search then uid search uses the latest",
			first:   map[string]any{"name": "search", "criteria": subject("message 1")},
			second:  map[string]any{"name": "uid search", "criteria": subject("message 2")},
			command: map[string]any{"name": "uid store", "dataitem": "+FLAGS", "value": `\Flagged`},
			flagged: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port := plainIMAPServer(t, 3)
			ret, err := runCommands(t, host, port,
				map[string]any{"name": "select", "mailbox": "INBOX"},
				tt.first, tt.second, tt.command,
				map[string]any{"name": "fetch", "sequence": "1:3", "dataitem": "FLAGS"},
			)
			require.NoError(t, err)
			res := ret["res"].(map[string]any)
			if tt.wantErr != "" {
				assert.Equal(t, 1, res["code"])
				assert.Contains(t, res["error"], tt.wantErr)
				return
			}
			require.Equal(t, 0, res["code"], res["error"])
			for i, f := range fetchedFlags(t, ret) {
				if i+1 == tt.flagged {
					assert.ElementsMatch(t, []string{`\Flagged`}, f, "message %d", i+1)
				} else {
					assert.Empty(t, f, "message %d", i+1)
				}
			}
		})
	}
}

func TestParseStoreFlags_Parentheses(t *testing.T) {
	for _, bad := range []string{"(", ")", `(\Seen`, `\Seen)`, " ( "} {
		_, err := parseStoreFlags("FLAGS", bad)
		assert.Error(t, err, "%q must be refused, not read as clearing every flag", bad)
	}
	for _, clear := range []string{"", "()", " ( ) "} {
		got, err := parseStoreFlags("FLAGS", clear)
		require.NoError(t, err, "%q", clear)
		assert.Empty(t, got.Flags, "%q clears the flags on purpose", clear)
	}
}

// stallingIMAPServer accepts connections and answers like an IMAP server up
// to the point given, then goes silent: "greeting" never sends the greeting,
// "select" logs the client in but never answers SELECT, and "logout" refuses
// SELECT and then never answers LOGOUT.
func stallingIMAPServer(t *testing.T, stallAt string) (string, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				if stallAt == "greeting" {
					_, _ = io.Copy(io.Discard, c)
					return
				}
				_, _ = c.Write([]byte("* OK [CAPABILITY IMAP4rev1 AUTH=PLAIN] ready\r\n"))
				rd := bufio.NewReader(c)
				for {
					line, err := rd.ReadString('\n')
					if err != nil {
						return
					}
					fields := strings.Fields(line)
					if len(fields) < 2 {
						continue
					}
					tag, cmd := fields[0], strings.ToUpper(fields[1])
					switch {
					case cmd == "LOGOUT" && stallAt == "logout":
						// Never answer: the session has to be cut off.
					case cmd == "LOGIN" || cmd == "LOGOUT" || cmd == "CAPABILITY" || cmd == "NOOP":
						_, _ = c.Write([]byte(tag + " OK done\r\n"))
					case cmd == "SELECT" && stallAt == "logout":
						_, _ = c.Write([]byte(tag + " NO no such mailbox\r\n"))
					default:
						// Never answer: the session has to be cut off.
					}
				}
			}(conn)
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", addr.Port
}

func TestRequest_TimeoutBoundsTheSession(t *testing.T) {
	tests := []struct {
		stallAt string
		want    string
	}{
		{"greeting", "IMAP session timed out after 300ms while logging in"},
		{"select", "IMAP session timed out after 300ms while running commands"},
		// The command fails first; the session then runs out on LOGOUT.
		{"logout", "IMAP session timed out after 300ms while logging out"},
	}
	for _, tt := range tests {
		t.Run(tt.stallAt, func(t *testing.T) {
			host, port := stallingIMAPServer(t, tt.stallAt)

			start := time.Now()
			_, err := Request(map[string]any{
				"host": host, "port": port, "username": "user", "password": "pass",
				"tls": false, "timeout": "300ms",
				"commands": []any{map[string]any{"name": "select", "mailbox": "INBOX"}},
			})
			elapsed := time.Since(start)

			require.Error(t, err, "a silent server must not leave the step hanging")
			assert.Contains(t, err.Error(), tt.want)
			assert.Less(t, elapsed, 3*time.Second, "the timeout has to cut the session short")
		})
	}
}

func TestRequest_TimeoutAsNumberIsSeconds(t *testing.T) {
	host, port := plainIMAPServer(t, 1)

	// 5 used to mean 5 nanoseconds, which no session survives.
	for _, timeout := range []any{5, 5.0, "5"} {
		ret, err := Request(map[string]any{
			"host": host, "port": port, "username": "user", "password": "pass",
			"tls": false, "timeout": timeout,
			// Ignored: the setting was removed and unknown keys are skipped.
			"strict_host_check": true,
			"commands":          []any{map[string]any{"name": "select", "mailbox": "INBOX"}},
		})
		require.NoError(t, err, "timeout %#v", timeout)
		req := ret["req"].(map[string]any)
		assert.Equal(t, 5*time.Second, req["timeout"], "timeout %#v", timeout)
	}
}

func TestParseTimeout(t *testing.T) {
	tests := []struct {
		in      any
		want    time.Duration
		wantErr bool
	}{
		{"30s", 30 * time.Second, false},
		{"1m30s", 90 * time.Second, false},
		{" 2 ", 2 * time.Second, false},
		{"1.5", 1500 * time.Millisecond, false},
		{2, 2 * time.Second, false},
		{int64(3), 3 * time.Second, false},
		{0.25, 250 * time.Millisecond, false},
		{10 * time.Second, 10 * time.Second, false},
		{"soon", 0, true},
		{0, 0, true},
		{"-1s", 0, true},
		{true, 0, true},
		{20_000_000_000, 0, true}, // would wrap to about 49 years
		{int64(maxTimeoutSeconds) + 1, 0, true},
		{1e20, 0, true},
		{"1e20", 0, true},
		{"NaN", 0, true},
		{-5, 0, true},
		{time.Duration(0), 0, true},
		{int64(maxTimeoutSeconds), time.Duration(maxTimeoutSeconds) * time.Second, false},
	}
	for _, tt := range tests {
		got, err := parseTimeout(tt.in)
		if tt.wantErr {
			assert.Error(t, err, "%#v", tt.in)
			continue
		}
		require.NoError(t, err, "%#v", tt.in)
		assert.Equal(t, tt.want, got, "%#v", tt.in)
	}
}

func TestRequest_TimeoutWhileLoggingOutKeepsTheCommandError(t *testing.T) {
	host, port := stallingIMAPServer(t, "logout")
	_, err := Request(map[string]any{
		"host": host, "port": port, "username": "user", "password": "pass",
		"tls": false, "timeout": "300ms",
		"commands": []any{map[string]any{"name": "select", "mailbox": "Nowhere"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "while logging out")
	assert.Contains(t, err.Error(), "after the command failed")
	assert.Contains(t, err.Error(), "no such mailbox")
}
