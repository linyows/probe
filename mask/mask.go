// Package mask hides secret values in everything Probe prints or writes.
package mask

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// redactedValue replaces the value of a header that carries credentials.
const redactedValue = "<redacted>"

// sensitiveKeys are header and parameter names whose values are credentials
// whatever they contain. A token obtained at run time, such as one returned by
// a login step, is never declared as a secret, but it reaches the target
// through one of these headers; a password written in a step's `with` need
// not be declared either. Their values are hidden wherever they are shown.
var sensitiveKeys = map[string]bool{
	"authorization":       true,
	"proxy-authorization": true,
	"cookie":              true,
	"set-cookie":          true,
	"password":            true,
	"key_passphrase":      true,
}

// dsnKey names the parameter that holds a database URL. Only the password in
// it is a credential: the rest says which database was used, so it stays
// visible.
const dsnKey = "dsn"

// Masker hides secret values in everything Probe prints or writes. It starts
// with the secrets the workflow declares and learns the values of credential
// headers as actions are about to send them, so that a token obtained at run
// time is hidden too. A nil Masker hides only credential headers in the maps
// passed to Map, and learns nothing.
type Masker struct {
	mu       sync.RWMutex
	labels   map[string]string // secret value -> what it is replaced with
	replacer *strings.Replacer
}

// New builds a Masker for the environment variables named in secrets.
// Each value is replaced by <secret:NAME>. A name that is not set, or set to
// an empty string, has nothing to hide and is skipped.
func New(secrets []string, env map[string]string) *Masker {
	m := &Masker{labels: make(map[string]string)}
	for _, name := range secrets {
		m.addLocked(env[name], "<secret:"+name+">")
	}
	m.rebuildLocked()
	return m
}

// addLocked registers a value under the label it is replaced with, along with
// the escaped forms it takes when Go, JSON or a log line quotes it, since a
// secret containing a quote or a backslash would otherwise slip through in
// an error message or a dump. Callers must hold mu or own m exclusively.
func (m *Masker) addLocked(value, label string) bool {
	if value == "" {
		return false
	}
	added := false
	for _, v := range escapedForms(value) {
		if _, exists := m.labels[v]; exists {
			continue
		}
		m.labels[v] = label
		added = true
	}
	return added
}

// rebuildLocked recompiles the replacer from the registered values. A value
// that contains another is replaced first, or the shorter one would cut it
// apart and leave the rest of it visible.
func (m *Masker) rebuildLocked() {
	if len(m.labels) == 0 {
		m.replacer = nil
		return
	}
	values := make([]string, 0, len(m.labels))
	for v := range m.labels {
		values = append(values, v)
	}
	sort.Slice(values, func(i, j int) bool {
		if len(values[i]) != len(values[j]) {
			return len(values[i]) > len(values[j])
		}
		return values[i] < values[j]
	})
	oldnew := make([]string, 0, len(values)*2)
	for _, v := range values {
		oldnew = append(oldnew, v, m.labels[v])
	}
	m.replacer = strings.NewReplacer(oldnew...)
}

// escapedForms returns value as written and in the escaped forms it takes
// when Go quotes it, JSON encodes it, or a log line escapes its double quotes.
// The escapes are also applied on top of each other, because an error that Go
// quoted is often quoted again by the logger that prints it.
func escapedForms(value string) []string {
	escapers := []func(string) string{
		func(s string) string {
			q := strconv.Quote(s)
			return q[1 : len(q)-1]
		},
		func(s string) string {
			var b strings.Builder
			enc := json.NewEncoder(&b)
			enc.SetEscapeHTML(false)
			_ = enc.Encode(s)
			e := strings.TrimSuffix(b.String(), "\n")
			return e[1 : len(e)-1]
		},
		func(s string) string {
			js, _ := json.Marshal(s)
			return string(js[1 : len(js)-1])
		},
		func(s string) string {
			return strings.ReplaceAll(s, `"`, `\"`)
		},
	}

	seen := map[string]bool{value: true}
	forms := []string{value}
	layer := []string{value}
	for depth := 0; depth < 2; depth++ {
		var next []string
		for _, f := range layer {
			for _, esc := range escapers {
				e := esc(f)
				if !seen[e] {
					seen[e] = true
					forms = append(forms, e)
					next = append(next, e)
				}
			}
		}
		layer = next
	}
	return forms
}

// Learn registers the values of credential headers and parameters found
// anywhere in data, and the password in a database URL, so that they are
// hidden from then on. It is called with an action's parameters
// before the action runs, which is what keeps them out of the action's own
// log records.
func (m *Masker) Learn(data map[string]any) {
	if m == nil || data == nil {
		return
	}
	var found []string
	collectSensitive(data, &found)
	if len(found) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	added := false
	for _, v := range found {
		if m.addLocked(v, redactedValue) {
			added = true
		}
	}
	if added {
		m.rebuildLocked()
	}
}

func collectSensitive(v any, found *[]string) {
	switch val := v.(type) {
	case map[string]any:
		for k, e := range val {
			if sensitiveKeys[strings.ToLower(k)] {
				collectStrings(e, found)
				continue
			}
			if dsn, ok := e.(string); ok && strings.ToLower(k) == dsnKey {
				*found = append(*found, dsnPassword(dsn)...)
				continue
			}
			collectSensitive(e, found)
		}
	case map[string]string:
		for k, s := range val {
			switch {
			case sensitiveKeys[strings.ToLower(k)]:
				*found = append(*found, s)
			case strings.ToLower(k) == dsnKey:
				*found = append(*found, dsnPassword(s)...)
			}
		}
	case []any:
		for _, e := range val {
			collectSensitive(e, found)
		}
	}
}

// dsnPassword returns the password in a URL-style DSN, both as written and
// percent-decoded, or nothing when it has none. It does not rely on url.Parse,
// which rejects the tcp(host:port) address a MySQL DSN may carry.
func dsnPassword(dsn string) []string {
	_, rest, ok := strings.Cut(dsn, "://")
	if !ok {
		return nil
	}
	// The authority ends where the path, query or fragment begins, and the
	// user info at the last @ before that, since an unescaped password may
	// itself contain an @.
	authority := rest
	if i := strings.IndexAny(rest, "/?#"); i >= 0 {
		authority = rest[:i]
	}
	at := strings.LastIndex(authority, "@")
	if at < 0 {
		return nil
	}
	_, password, ok := strings.Cut(authority[:at], ":")
	if !ok || password == "" {
		return nil
	}
	found := []string{password}
	if decoded, err := url.PathUnescape(password); err == nil && decoded != password {
		found = append(found, decoded)
	}
	return found
}

// collectStrings gathers the values a credential header or parameter holds;
// Set-Cookie, for one, can carry several.
func collectStrings(v any, found *[]string) {
	switch val := v.(type) {
	case string:
		*found = append(*found, val)
	// A password written without quotes, such as 123456, is a number in the
	// workflow, and reaches the action as the same digits. A boolean is not
	// learned: hiding every "true" in the output would hide nothing useful.
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		*found = append(*found, fmt.Sprint(val))
	case []any:
		for _, e := range val {
			collectStrings(e, found)
		}
	case []string:
		*found = append(*found, val...)
	}
}

// String hides every secret value in s.
func (m *Masker) String(s string) string {
	if m == nil {
		return s
	}
	m.mu.RLock()
	r := m.replacer
	m.mu.RUnlock()
	if r == nil {
		return s
	}
	return r.Replace(s)
}

// Map returns a copy of a request or response with secret values hidden in
// every string, and the values of credential headers replaced outright. The
// original is left untouched, since tests and outputs still need the real
// values. A nil map stays nil.
func (m *Masker) Map(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}
	out := make(map[string]any, len(data))
	for k, v := range data {
		if sensitiveKeys[strings.ToLower(k)] {
			out[k] = redactedValue
			continue
		}
		out[k] = m.value(v)
	}
	return out
}

func (m *Masker) value(v any) any {
	switch val := v.(type) {
	case string:
		return m.String(val)
	case map[string]any:
		return m.Map(val)
	case map[string]string:
		out := make(map[string]any, len(val))
		for k, s := range val {
			if sensitiveKeys[strings.ToLower(k)] {
				out[k] = redactedValue
				continue
			}
			out[k] = m.String(s)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, e := range val {
			out[i] = m.value(e)
		}
		return out
	case []string:
		out := make([]any, len(val))
		for i, e := range val {
			out[i] = m.String(e)
		}
		return out
	default:
		return v
	}
}

// Writer wraps w so that secret values are hidden in what is written to it.
// Every caller writes whole messages or lines, so a value is never split
// across two writes. The wrapper consults the masker on each write, so values
// learned later are hidden too.
func (m *Masker) Writer(w io.Writer) io.Writer {
	if m == nil {
		return w
	}
	return &maskingWriter{w: w, m: m}
}

type maskingWriter struct {
	w io.Writer
	m *Masker
}

func (mw *maskingWriter) Write(p []byte) (int, error) {
	if _, err := io.WriteString(mw.w, mw.m.String(string(p))); err != nil {
		return 0, err
	}
	// Report the input length: the caller wrote all of p, whatever it
	// became on the way out.
	return len(p), nil
}
