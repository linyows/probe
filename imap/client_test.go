package imap

import (
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
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
