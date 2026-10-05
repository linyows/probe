package mask

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	env := map[string]string{
		"TOKEN":   "tok-123",
		"LONG":    "tok-123-and-more",
		"EMPTY":   "",
		"UNUSED":  "not-declared",
		"PASS":    "hunter2",
		"PASSDUP": "hunter2",
	}
	m := New([]string{"TOKEN", "LONG", "EMPTY", "MISSING", "PASS", "PASSDUP"}, env)

	tests := []struct {
		in, want string
	}{
		{"token=tok-123", "token=<secret:TOKEN>"},
		{"tok-123-and-more", "<secret:LONG>"},
		{"pw hunter2", "pw <secret:PASS>"},
		{"not-declared stays", "not-declared stays"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := m.String(tt.in); got != tt.want {
			t.Errorf("String(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMasker_NilAndEmpty(t *testing.T) {
	var nilMasker *Masker
	if got := nilMasker.String("x"); got != "x" {
		t.Errorf("nil String = %q", got)
	}
	nilMasker.Learn(map[string]any{"authorization": "a"})

	var buf bytes.Buffer
	if w := nilMasker.Writer(&buf); w != &buf {
		t.Error("a nil masker should return the writer unchanged")
	}

	empty := New(nil, nil)
	if got := empty.String("anything"); got != "anything" {
		t.Errorf("empty String = %q", got)
	}
}

func TestEscapedForms(t *testing.T) {
	value := `a"b\c`
	forms := escapedForms(value)

	goQuoted := strconv.Quote(value)
	js, _ := json.Marshal(value)
	for _, want := range []string{
		value,
		goQuoted[1 : len(goQuoted)-1], // a\"b\\c
		string(js[1 : len(js)-1]),     // a\"b\\c
		`a\"b\c`,                      // a log line escaping only quotes
		`a\\"b\\c`,                    // Go quoted, then quotes escaped by the logger
		`a\\\"b\\\\c`,                 // quoted twice
	} {
		found := false
		for _, f := range forms {
			if f == want {
				found = true
			}
		}
		if !found {
			t.Errorf("escapedForms(%q) is missing %q; got %q", value, want, forms)
		}
	}

	if got := escapedForms("plain"); len(got) != 1 {
		t.Errorf("a value nothing escapes should have one form, got %q", got)
	}
}

func TestMasker_StringHidesEscapedForms(t *testing.T) {
	secret := `s3cr"et\value`
	m := New([]string{"S"}, map[string]string{"S": secret})

	inputs := []string{
		fmt.Sprintf("%q", secret),
		fmt.Sprintf("%#v", map[string]any{"v": secret}),
		fmt.Sprintf("error=%q", fmt.Sprintf("Get %q", secret)),
	}
	for _, in := range inputs {
		out := m.String(in)
		if strings.Contains(out, "s3cr") {
			t.Errorf("String(%s) = %s, still shows the secret", in, out)
		}
	}
}

func TestMasker_Map(t *testing.T) {
	m := New([]string{"S"}, map[string]string{"S": "sekret"})
	original := map[string]any{
		"url": "http://x/?k=sekret",
		"headers": map[string]any{
			"Authorization": "Bearer abc",
			"Accept":        "text/plain",
		},
		"plain":  map[string]string{"cookie": "sid=1", "x": "sekret"},
		"list":   []any{"sekret", map[string]any{"set-cookie": "a=b"}},
		"names":  []string{"sekret"},
		"number": 3,
	}
	snapshot := fmt.Sprintf("%#v", original)

	got := m.Map(original)

	if got["url"] != "http://x/?k=<secret:S>" {
		t.Errorf("url = %v", got["url"])
	}
	headers := got["headers"].(map[string]any)
	if headers["Authorization"] != redactedValue || headers["Accept"] != "text/plain" {
		t.Errorf("headers = %v", headers)
	}
	plain := got["plain"].(map[string]any)
	if plain["cookie"] != redactedValue || plain["x"] != "<secret:S>" {
		t.Errorf("map[string]string = %v", plain)
	}
	list := got["list"].([]any)
	if list[0] != "<secret:S>" || list[1].(map[string]any)["set-cookie"] != redactedValue {
		t.Errorf("list = %v", list)
	}
	if names := got["names"].([]any); names[0] != "<secret:S>" {
		t.Errorf("names = %v", names)
	}
	if got["number"] != 3 {
		t.Errorf("number = %v", got["number"])
	}

	if fmt.Sprintf("%#v", original) != snapshot {
		t.Error("Map must not change the original")
	}

	var nilMasker *Masker
	if r := nilMasker.Map(map[string]any{"cookie": "c", "k": "v"}); r["cookie"] != redactedValue || r["k"] != "v" {
		t.Errorf("a nil masker should still redact credential headers, got %v", r)
	}
	if m.Map(nil) != nil {
		t.Error("nil should stay nil")
	}
}

func TestMasker_Learn(t *testing.T) {
	m := New(nil, nil)
	m.Learn(map[string]any{
		"url": "http://x",
		"headers": map[string]any{
			"AUTHORIZATION": "Bearer runtime-token",
		},
		"nested": []any{
			map[string]any{"set-cookie": []any{"sid=abc123", "theme=dark"}},
		},
		"plain": map[string]string{"Cookie": "jar=789"},
	})

	for in, want := range map[string]string{
		"sent Bearer runtime-token": "sent <redacted>",
		"sid=abc123; theme=dark":    "<redacted>; <redacted>",
		"jar=789":                   "<redacted>",
		"http://x":                  "http://x",
	} {
		if got := m.String(in); got != want {
			t.Errorf("String(%q) = %q, want %q", in, got, want)
		}
	}

	// A declared secret keeps its label even when it is also learned.
	d := New([]string{"T"}, map[string]string{"T": "Bearer tok"})
	d.Learn(map[string]any{"authorization": "Bearer tok"})
	if got := d.String("Bearer tok"); got != "<secret:T>" {
		t.Errorf("declared label should win, got %q", got)
	}
}

func TestMasker_ConcurrentLearnAndString(t *testing.T) {
	m := New(nil, nil)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			m.Learn(map[string]any{"authorization": fmt.Sprintf("token-%d", i)})
		}(i)
		go func(i int) {
			defer wg.Done()
			_ = m.String(fmt.Sprintf("token-%d", i))
		}(i)
	}
	wg.Wait()

	for i := 0; i < 20; i++ {
		if got := m.String(fmt.Sprintf("token-%d.", i)); got != "<redacted>." {
			t.Errorf("token-%d not learned: %q", i, got)
		}
	}
}

func TestMasker_Writer(t *testing.T) {
	m := New([]string{"S"}, map[string]string{"S": "sekret"})
	var buf bytes.Buffer
	w := m.Writer(&buf)

	in := []byte("log sekret line\n")
	n, err := w.Write(in)
	if err != nil || n != len(in) {
		t.Errorf("Write = %d, %v; want %d, nil", n, err, len(in))
	}
	if buf.String() != "log <secret:S> line\n" {
		t.Errorf("wrote %q", buf.String())
	}

	// Values learned after the writer was made are hidden too.
	m.Learn(map[string]any{"cookie": "late-cookie"})
	_, _ = w.Write([]byte("late-cookie\n"))
	if !strings.HasSuffix(buf.String(), "<redacted>\n") {
		t.Errorf("learned value not hidden: %q", buf.String())
	}
}

// A password passed to an action is hidden even when the workflow does not
// declare it as a secret, wherever the action or Probe shows it.
func TestMasker_LearnCredentialParams(t *testing.T) {
	m := New(nil, nil)
	m.Learn(map[string]any{
		"host":           "imap.example.com",
		"Password":       "imap-pass",
		"key_passphrase": "key-pass",
		"user":           "alice",
	})

	got := m.String(`dial alice:imap-pass with "key-pass" at imap.example.com`)
	want := `dial alice:<redacted> with "<redacted>" at imap.example.com`
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}

	r := New(nil, nil).Map(map[string]any{"password": "p", "user": "alice"})
	if r["password"] != redactedValue || r["user"] != "alice" {
		t.Errorf("Map() = %v, want the password redacted and the user kept", r)
	}
}

func TestMasker_LearnDSNPassword(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		in   string
		want string
	}{
		{
			name: "postgres",
			dsn:  "postgres://app:pg-secret@db:5432/main?sslmode=disable",
			in:   "connect postgres://app:pg-secret@db:5432/main?sslmode=disable",
			want: "connect postgres://app:<redacted>@db:5432/main?sslmode=disable",
		},
		{
			name: "mysql with a tcp address",
			dsn:  "mysql://app:my-secret@tcp(db:3306)/main",
			in:   "app:my-secret@tcp(db:3306)/main",
			want: "app:<redacted>@tcp(db:3306)/main",
		},
		{
			name: "an @ in the password",
			dsn:  "mysql://app:p@ss@db/main",
			in:   "app:p@ss@db",
			want: "app:<redacted>@db",
		},
		{
			name: "percent-encoded",
			dsn:  "postgres://app:a%2Fb%40c@db/main",
			in:   "raw a%2Fb%40c decoded a/b@c",
			want: "raw <redacted> decoded <redacted>",
		},
		{
			name: "no path, an @ in the query",
			dsn:  "postgres://app:sample-pass@db?application_name=ops@example.com",
			in:   "app:sample-pass@db?application_name=ops@example.com",
			want: "app:<redacted>@db?application_name=ops@example.com",
		},
		{
			name: "a fragment",
			dsn:  "postgres://app:frag-pass@db#a@b",
			in:   "frag-pass db#a@b",
			want: "<redacted> db#a@b",
		},
		{
			name: "no password",
			dsn:  "postgres://app@db/main",
			in:   "postgres://app@db/main",
			want: "postgres://app@db/main",
		},
		{
			name: "sqlite",
			dsn:  "file:./data.db",
			in:   "file:./data.db",
			want: "file:./data.db",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(nil, nil)
			m.Learn(map[string]any{"dsn": tt.dsn, "query": "select 1"})
			if got := m.String(tt.in); got != tt.want {
				t.Errorf("String(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMasker_LearnDSNInStringMap(t *testing.T) {
	m := New(nil, nil)
	m.Learn(map[string]any{
		"conn": map[string]string{"dsn": "postgres://app:typed-pass@db/main"},
	})
	if got := m.String("typed-pass"); got != redactedValue {
		t.Errorf("String() = %q, want the DSN password hidden", got)
	}
}

// A password written without quotes is a number in the workflow, but the
// action receives the same digits, so they are hidden too.
func TestMasker_LearnNumericCredential(t *testing.T) {
	tests := []struct {
		name  string
		value any
		in    string
	}{
		{name: "int", value: 482915, in: "pass 482915"},
		{name: "uint64", value: uint64(482915), in: "pass 482915"},
		{name: "float64", value: float64(482915), in: "pass 482915"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(nil, nil)
			m.Learn(map[string]any{"password": tt.value})
			if got := m.String(tt.in); got != "pass "+redactedValue {
				t.Errorf("String(%q) = %q, want the number hidden", tt.in, got)
			}
		})
	}

	m := New(nil, nil)
	m.Learn(map[string]any{"password": true})
	if got := m.String("enabled: true"); got != "enabled: true" {
		t.Errorf("String() = %q, want a boolean password not to hide every true", got)
	}
}

func TestHoldsCredential(t *testing.T) {
	for key, want := range map[string]bool{
		"password":       true,
		"Password":       true,
		"key_passphrase": true,
		"authorization":  true,
		"Cookie":         true,
		"dsn":            true,
		"url":            false,
		"headers":        false,
	} {
		if got := HoldsCredential(key); got != want {
			t.Errorf("HoldsCredential(%q) = %v, want %v", key, got, want)
		}
	}
}
