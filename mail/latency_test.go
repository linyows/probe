package mail

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestGetLatency(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := GetLatencies("./testdata/mail/", buf); err != nil {
		t.Errorf("got error %s", err)
	}

	bytes, err := os.ReadFile("./testdata/latency.csv")
	if err != nil {
		t.Errorf("csv read error %s", err)
	}

	got := buf.String()
	expects := string(bytes)
	if got != expects {
		t.Errorf("\nExpected:\n%s\nGot:\n%s", expects, got)
	}
}

func TestGetSentTimeWithParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
		wantErr  bool
	}{
		{
			name:     "single digit day with double space",
			input:    "Date: Wed,  8 Oct 2025 07:11:55 +0000",
			expected: time.Date(2025, 10, 8, 7, 11, 55, 0, time.FixedZone("UTC", 0)),
			wantErr:  false,
		},
		{
			name:     "double digit day",
			input:    "Date: Sun, 25 Aug 2024 16:55:51 +0900",
			expected: time.Date(2024, 8, 25, 16, 55, 51, 0, time.FixedZone("JST", 9*3600)),
			wantErr:  false,
		},
		{
			name:     "single digit day - first day of month",
			input:    "Date: Mon,  1 Jan 2025 00:00:00 -0700",
			expected: time.Date(2025, 1, 1, 0, 0, 0, 0, time.FixedZone("MST", -7*3600)),
			wantErr:  false,
		},
	}

	l := &Latencies{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := l.getSentTimeWithParse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSentTimeWithParse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !got.Equal(tt.expected) {
				t.Errorf("getSentTimeWithParse() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetReceivedTimeWithParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
		wantErr  bool
	}{
		{
			name:     "single digit day with double space",
			input:    "Received: from mx.example.com; Wed,  8 Oct 2025 07:42:05 +0000",
			expected: time.Date(2025, 10, 8, 7, 42, 5, 0, time.FixedZone("UTC", 0)),
			wantErr:  false,
		},
		{
			name:     "double digit day",
			input:    "Received: from mx.example.com; Sun, 25 Aug 2024 17:00:01 +0900",
			expected: time.Date(2024, 8, 25, 17, 0, 1, 0, time.FixedZone("JST", 9*3600)),
			wantErr:  false,
		},
		{
			name:     "malformed header without semicolon",
			input:    "Received: from mx.example.com",
			wantErr:  true,
		},
	}

	l := &Latencies{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := l.getReceivedTimeWithParse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("getReceivedTimeWithParse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.expected) {
				t.Errorf("getReceivedTimeWithParse() = %v, want %v", got, tt.expected)
			}
		})
	}
}
