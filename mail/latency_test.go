package mail

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// useLocation makes loc the local time zone for the rest of the test.
func useLocation(t *testing.T, name string) {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	orig := time.Local
	time.Local = loc
	t.Cleanup(func() { time.Local = orig })
}

func TestGetLatency(t *testing.T) {
	// The expected CSV is written in Japan time, which is what the times are
	// shown in when that is the local time zone.
	useLocation(t, "Asia/Tokyo")
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
			name:     "trailing comment",
			input:    "Received: from mx.example.com; Wed, 8 Oct 2025 07:12:00 +0000 (UTC)",
			expected: time.Date(2025, 10, 8, 7, 12, 0, 0, time.UTC),
		},
		{
			name:     "semicolon inside a trailing comment",
			input:    "Received: from client by mx; Wed, 8 Oct 2025 07:12:00 +0000 (delivery metadata; cached route)",
			expected: time.Date(2025, 10, 8, 7, 12, 0, 0, time.UTC),
		},
		{
			name:     "semicolon inside an earlier comment",
			input:    "Received: from client (helo; spoofed) by mx; Wed, 8 Oct 2025 07:12:00 +0000",
			expected: time.Date(2025, 10, 8, 7, 12, 0, 0, time.UTC),
		},
		{
			name:     "nested and escaped parentheses",
			input:    `Received: from client by mx; Wed, 8 Oct 2025 07:12:00 +0000 (a (b; c) \); d)`,
			expected: time.Date(2025, 10, 8, 7, 12, 0, 0, time.UTC),
		},
		{
			name:    "malformed header without semicolon",
			input:   "Received: from mx.example.com",
			wantErr: true,
		},
		{
			name:    "semicolon only inside a comment",
			input:   "Received: from mx.example.com (a; b)",
			wantErr: true,
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

// writeMail writes a message into dir under name.
func writeMail(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLatenciesSingleLineReceived(t *testing.T) {
	// A Received header on one line is as complete as a folded one.
	dir := t.TempDir()
	writeMail(t, dir, "1", "Return-Path: <a@example.com>\r\n"+
		"Received: from relay.example.com by mx.example.com; Wed, 8 Oct 2025 07:12:05 +0000\r\n"+
		"Received: from client by relay.example.com; Wed, 8 Oct 2025 07:12:00 +0000\r\n"+
		"Date: Wed, 8 Oct 2025 07:11:55 +0000\r\n"+
		"Subject: test\r\n\r\nbody\r\n")

	l := Latencies{MailDir: dir}
	if err := l.Make(); err != nil {
		t.Fatalf("Make() error: %v", err)
	}
	if len(l.Data) != 1 {
		t.Fatalf("Data = %v, want one message", l.Data)
	}
	if got := l.Data[0].EndToEnd; got != 10*time.Second {
		t.Errorf("EndToEnd = %v, want 10s", got)
	}
	if got := l.Data[0].Relay; got != 5*time.Second {
		t.Errorf("Relay = %v, want 5s", got)
	}
}

func TestLatenciesSkipsOtherFilesAndBody(t *testing.T) {
	// A file that is not a message needs no Date, and a body line that looks
	// like a header is not read as one.
	dir := t.TempDir()
	writeMail(t, dir, "notes.txt", "these are notes\n")
	writeMail(t, dir, "1", "Return-Path: <a@example.com>\n"+
		"Received: from client by mx.example.com;\n"+
		"\tWed, 8 Oct 2025 07:12:00 +0000\n"+
		"Date: Wed, 8 Oct 2025 07:11:55 +0000\n"+
		"\n"+
		"Date: not a date\n"+
		"Received: not a header\n")

	l := Latencies{MailDir: dir}
	if err := l.Make(); err != nil {
		t.Fatalf("Make() error: %v", err)
	}
	if len(l.Data) != 1 || l.Data[0].EndToEnd != 5*time.Second {
		t.Errorf("Data = %+v, want one message with a 5s latency", l.Data)
	}
}

func TestLatenciesLocalTime(t *testing.T) {
	// Times are written in the local time zone.
	useLocation(t, "America/New_York")
	dir := t.TempDir()
	writeMail(t, dir, "1", "Return-Path: <a@example.com>\n"+
		"Received: from client by mx.example.com; Wed, 8 Oct 2025 07:12:00 +0000\n"+
		"Date: Wed, 8 Oct 2025 07:11:55 +0000\n\n")

	buf := new(bytes.Buffer)
	if err := GetLatencies(dir, buf); err != nil {
		t.Fatalf("GetLatencies() error: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("2025-10-08 03:11:55")) {
		t.Errorf("CSV = %s, want the sent time in New York time", buf)
	}
}

func TestLatenciesFoldedReceivedWithComment(t *testing.T) {
	// The comment after the date has a ";" of its own and sits on a line of
	// its own; the date before it still counts.
	dir := t.TempDir()
	writeMail(t, dir, "1", "Return-Path: <a@example.com>\n"+
		"Received: from client\n"+
		"\tby mx; Wed, 8 Oct 2025 07:12:00 +0000\n"+
		"\t(delivery metadata; cached route)\n"+
		"Date: Wed, 8 Oct 2025 07:11:55 +0000\n\n")

	l := Latencies{MailDir: dir}
	if err := l.Make(); err != nil {
		t.Fatalf("Make() error: %v", err)
	}
	if len(l.Data) != 1 || l.Data[0].EndToEnd != 5*time.Second {
		t.Errorf("Data = %+v, want one message with a 5s latency", l.Data)
	}
}
