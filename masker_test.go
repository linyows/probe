package probe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestNewMasker(t *testing.T) {
	env := map[string]string{
		"TOKEN":   "tok-123",
		"LONG":    "tok-123-and-more",
		"EMPTY":   "",
		"UNUSED":  "not-declared",
		"PASS":    "hunter2",
		"PASSDUP": "hunter2",
	}
	m := NewMasker([]string{"TOKEN", "LONG", "EMPTY", "MISSING", "PASS", "PASSDUP"}, env)

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

	empty := NewMasker(nil, nil)
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
	m := NewMasker([]string{"S"}, map[string]string{"S": secret})

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
	m := NewMasker([]string{"S"}, map[string]string{"S": "sekret"})
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
	m := NewMasker(nil, nil)
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
	d := NewMasker([]string{"T"}, map[string]string{"T": "Bearer tok"})
	d.Learn(map[string]any{"authorization": "Bearer tok"})
	if got := d.String("Bearer tok"); got != "<secret:T>" {
		t.Errorf("declared label should win, got %q", got)
	}
}

func TestMasker_ConcurrentLearnAndString(t *testing.T) {
	m := NewMasker(nil, nil)
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
	m := NewMasker([]string{"S"}, map[string]string{"S": "sekret"})
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

func TestPrinter_MasksOutput(t *testing.T) {
	p := newBufferPrinter()
	p.SetMasker(NewMasker([]string{"S"}, map[string]string{"S": `pa"ss`}))

	p.Fprint(p.outWriter, "a pa\"ss b")
	p.Fprintf(p.outWriter, " %s", `pa"ss`)
	p.Fprintln(p.outWriter, " pa\"ss")
	p.PrintError("failed with %q", `pa"ss`)

	failure := p.generateTestFailure("t", false,
		map[string]any{"headers": map[string]any{"authorization": "Bearer x"}, "body": `pa"ss`},
		map[string]any{"body": `pa"ss`})
	p.Fprint(p.outWriter, failure)

	p.verbose = true
	p.PrintTestResult(false, "t", StepContext{
		Req: map[string]any{"cookie": "c=1"},
		Res: map[string]any{"body": `pa"ss`},
	})

	out := p.outWriter.(*bytes.Buffer).String() + p.errWriter.(*bytes.Buffer).String()
	if strings.Contains(out, "pa") {
		t.Errorf("output still shows the secret:\n%s", out)
	}
	if strings.Contains(out, "Bearer x") || strings.Contains(out, "c=1") {
		t.Errorf("output still shows a credential header:\n%s", out)
	}
	if !strings.Contains(out, "<secret:S>") || !strings.Contains(out, redactedValue) {
		t.Errorf("expected masked markers in:\n%s", out)
	}
}

func TestReport_Mask(t *testing.T) {
	m := NewMasker([]string{"S"}, map[string]string{"S": "sekret"})
	r := &Report{
		Name:        "run sekret",
		Description: "desc sekret",
		Jobs: []JobReport{{
			Name: "job sekret",
			Steps: []StepReport{{
				Name: "step sekret",
				Test: `res.body == "sekret"`,
				Echo: "echo sekret",
				Failure: &FailureReport{
					Kind:     FailureAssertion,
					Message:  "msg sekret",
					Request:  map[string]any{"authorization": "Bearer z", "q": "sekret"},
					Response: map[string]any{"body": "sekret"},
				},
			}},
		}},
	}

	r.Mask(m)

	var buf bytes.Buffer
	if err := r.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "sekret") || strings.Contains(buf.String(), "Bearer z") {
		t.Errorf("report still shows a secret:\n%s", buf.String())
	}
	if got := strings.Count(buf.String(), "<secret:S>"); got != 9 {
		t.Errorf("masked %d values, want 9:\n%s", got, buf.String())
	}
}

// TestWorkflow_MasksSecrets runs a workflow end to end and checks that a
// declared secret and a credential header stay out of the terminal output and
// the report, while the action still receives the real values.
func TestWorkflow_MasksSecrets(t *testing.T) {
	runner := &recordingRunner{result: map[string]any{
		"req": map[string]any{
			"url":     "http://api.test/?key=sekret-key",
			"headers": map[string]any{"authorization": "Bearer runtime-tok"},
		},
		"res": map[string]any{"code": 500, "body": "echo sekret-key"},
	}}

	w := &Workflow{
		Name:    "masking",
		Secrets: []string{"API_KEY"},
		env:     map[string]string{"API_KEY": "sekret-key"},
		Vars:    map[string]any{"key": "{{API_KEY}}"},
		Jobs: []Job{{
			Name: "job",
			Steps: []*Step{{
				Name: "call {{vars.key}}",
				Uses: "http",
				With: map[string]any{
					"url":     "http://api.test/?key={{vars.key}}",
					"headers": map[string]any{"authorization": "Bearer runtime-tok"},
				},
				Test:         "res.code == 200",
				Echo:         "body was {{res.body}}",
				actionRunner: runner,
			}},
		}},
		printer: newBufferPrinter(),
	}
	w.printer.verbose = true

	path := filepath.Join(t.TempDir(), "report.json")
	if err := w.Start(Config{Verbose: true, Reports: []ReportTarget{{Format: ReportJSON, Path: path}}}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if got := runner.with["url"]; got != "http://api.test/?key=sekret-key" {
		t.Errorf("the action should receive the real value, got %v", got)
	}
	if runner.opts.Masker == nil {
		t.Error("the action should be given the masker for its log records")
	}

	report, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := w.printer.outWriter.(*bytes.Buffer).String() +
		w.printer.errWriter.(*bytes.Buffer).String() +
		string(report)

	for _, leak := range []string{"sekret-key", "runtime-tok"} {
		if strings.Contains(out, leak) {
			t.Errorf("%q leaked:\n%s", leak, out)
		}
	}
	if !strings.Contains(out, "<secret:API_KEY>") {
		t.Errorf("expected the secret's marker in the output:\n%s", out)
	}
}

// recordingRunner returns a fixed result and keeps what it was called with.
type recordingRunner struct {
	mu     sync.Mutex
	result map[string]any
	with   map[string]any
	opts   RunOptions
}

func (r *recordingRunner) RunActions(name string, with map[string]any, opts RunOptions) (map[string]any, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.with = with
	r.opts = opts
	return r.result, nil
}

func TestProbe_LoadSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "w.yml")
	yml := "name: s\nsecrets:\n- API_TOKEN\n- DB_PASSWORD\njobs:\n- name: j\n  steps:\n  - name: s\n    uses: hello\n"
	if err := os.WriteFile(path, []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}

	p := New(path, false)
	if err := p.Load(); err != nil {
		t.Fatal(err)
	}
	if want := []string{"API_TOKEN", "DB_PASSWORD"}; !reflect.DeepEqual(p.workflow.Secrets, want) {
		t.Errorf("Secrets = %v, want %v", p.workflow.Secrets, want)
	}
}
