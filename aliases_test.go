package probe

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/linyows/probe/actionref"
)

const aliasSHA = "0123456789abcdef0123456789abcdef01234567"

var builtinNames = []string{"embedded", "http", "shell"}

func loadWorkflow(t *testing.T, content string, more ...string) (*Probe, error) {
	t.Helper()
	dir := t.TempDir()
	paths := []string{writeWorkflow(t, dir, "workflow.yml", content)}
	for i, c := range more {
		paths = append(paths, writeWorkflow(t, dir, fmt.Sprintf("more%d.yml", i), c))
	}
	p := &Probe{FilePath: strings.Join(paths, ","), Config: Config{Log: os.Stdout, Actions: builtinNames}}
	return p, p.Load()
}

func TestLoad_ActionNames(t *testing.T) {
	redis := "github.com/mozership/probe-redis@" + aliasSHA
	p, err := loadWorkflow(t, `name: names
actions:
  redis: `+redis+`
  greet: ./greet
jobs:
- name: J
  defaults:
    redis:
      url: redis://localhost
      nested:
        a: 1
    http:
      url: http://localhost
  steps:
  - uses: redis
    with:
      commands: [PING]
      nested:
        b: 2
  - uses: greet
  - uses: http
  - uses: `+redis+`
- name: K
  steps:
  - uses: redis
`)
	if err != nil {
		t.Fatal(err)
	}
	job := p.workflow.Jobs[0]
	for i, want := range []string{redis, "./greet", "http", redis} {
		if got := job.Steps[i].Uses; got != want {
			t.Errorf("steps[%d].uses = %q, want %q", i, got, want)
		}
	}
	// The defaults written by the name reach the steps that use the action,
	// by the name or in full.
	for _, i := range []int{0, 3} {
		if got := job.Steps[i].With["url"]; got != "redis://localhost" {
			t.Errorf("steps[%d].with.url = %v, want the default", i, got)
		}
	}
	nested, _ := job.Steps[0].With["nested"].(map[string]any)
	if len(nested) != 2 {
		t.Errorf("steps[0].with.nested = %v, want the keys of the step and of the defaults", nested)
	}
	if got := job.Steps[2].With["url"]; got != "http://localhost" {
		t.Errorf("steps[2].with.url = %v, want the default of http", got)
	}
	if _, ok := job.Steps[1].With["url"]; ok {
		t.Errorf("steps[1].with = %v, want none of the defaults of another action", job.Steps[1].With)
	}
	defaults, _ := job.Defaults.(map[string]any)
	if _, ok := defaults[redis]; !ok || len(defaults) != 2 {
		t.Errorf("defaults = %v, want them keyed by the action", defaults)
	}
	// A job without defaults has its steps resolved too.
	if got := p.workflow.Jobs[1].Steps[0].Uses; got != redis {
		t.Errorf("jobs[1].steps[0].uses = %q, want %q", got, redis)
	}
}

func TestLoad_ActionNamesRefused(t *testing.T) {
	tests := []struct {
		name    string
		actions string
		job     string
		want    string
	}{
		{"the name of an action of Probe", "  http: ./greet\n", "", `the name "http" is that of an action of Probe`},
		{"embedded", "  embedded: ./greet\n", "", `the name "embedded" is that of an action of Probe`},
		{"a name with a slash", "  \"my/redis\": ./greet\n", "", `the name "my/redis" must be letters`},
		{"a name with a dot", "  my.redis: ./greet\n", "", `the name "my.redis" must be letters`},
		{"a name that holds a template", "  \"{{vars.x}}\": ./greet\n", "", `must be letters`},
		{"an empty name", "  \"\": ./greet\n", "", `must be letters`},
		{"another name", "  a: ./greet\n  b: a\n", "", `b must name an external action`},
		{"an action of Probe", "  web: http\n", "", `web must name an external action`},
		{"nothing", "  redis: \"\"\n", "", `redis must name an external action`},
		{"a tag", "  redis: github.com/mozership/probe-redis@v1.0.0\n", "", `must be pinned to a full 40-character commit SHA`},
		{"no commit", "  redis: github.com/mozership/probe-redis\n", "", `must be pinned to a commit`},
		{"another host", "  redis: gitlab.com/mozership/probe-redis@" + aliasSHA + "\n", "", `unsupported action`},
		{"a template in the commit", "  redis: github.com/mozership/probe-redis@{{vars.sha}}\n", "", `redis must name an external action as it is, not by a template`},
		{"a template in a local path", "  greet: \"./{{vars.action}}\"\n", "", `greet must name an external action as it is, not by a template`},
		{"a template for the whole of it", "  greet: \"{{vars.action}}/greet\"\n", "", `greet must name an external action as it is, not by a template`},
		{
			"defaults written twice",
			"  greet: ./greet\n",
			"  defaults:\n    greet: {name: a}\n    ./greet: {name: b}\n",
			`greet and ./greet are the same action`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadWorkflow(t, "name: refused\nactions:\n"+tt.actions+"jobs:\n- name: J\n"+tt.job+"  steps:\n  - uses: http\n")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Load() error = %v, want one saying %q", err, tt.want)
			}
		})
	}
}

// Without the names of the actions of Probe, as a caller of the package may
// leave them, a name is not checked against them.
func TestLoad_ActionNamesWithoutBuiltin(t *testing.T) {
	dir := t.TempDir()
	path := writeWorkflow(t, dir, "workflow.yml", "name: n\nactions:\n  http: ./greet\njobs:\n- name: J\n  steps:\n  - uses: http\n")
	p := &Probe{FilePath: path, Config: Config{Log: os.Stdout}}
	if err := p.Load(); err != nil {
		t.Fatal(err)
	}
	if got := p.workflow.Jobs[0].Steps[0].Uses; got != "./greet" {
		t.Errorf("uses = %q, want ./greet", got)
	}
}

// The actions of the last file given replace those of the files before it,
// as any top-level key does.
func TestLoad_ActionNamesOfSeveralFiles(t *testing.T) {
	p, err := loadWorkflow(t,
		"actions:\n  greet: ./old\n  gone: ./gone\n",
		"name: n\nactions:\n  greet: ./new\njobs:\n- name: J\n  steps:\n  - uses: greet\n  - uses: gone\n")
	if err != nil {
		t.Fatal(err)
	}
	steps := p.workflow.Jobs[0].Steps
	if steps[0].Uses != "./new" || steps[1].Uses != "gone" {
		t.Errorf("uses = %q and %q, want ./new and gone", steps[0].Uses, steps[1].Uses)
	}
}

func TestCheck_ActionNames(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "declared"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeWorkflow(t, filepath.Join(dir, "declared"), "action.yml", "runs:\n  using: binary\n  path: x\nparams: [url, calls]\n")
	path := writeWorkflow(t, dir, "workflow.yml", `name: names
actions:
  cache: ./declared
  http: ./declared
  store: github.com/mozership/probe-s3@v1
jobs:
- name: J
  defaults:
    cache:
      url: redis://localhost
      timout: 5s
  steps:
  - uses: cache
    with:
      calls: []
      cals: []
    test: res.code == 0
  - uses: cash
    test: res.code == 0
  - uses: store
    test: res.code == 0
- name: K
  defaults:
    cache: {url: a}
    ./declared: {url: b}
  steps:
  - uses: cache
    test: res.code == 0
`)
	opts := CheckOptions{Actions: []string{"http"}, Params: map[string][]string{"http": {"url"}}, Manifest: actionref.ReadManifest}
	findings, err := Check(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	checkFindings(t, findings, []want{
		{SeverityError, 4, `actions: the name "http" is that of an action of Probe`},
		{SeverityError, 5, `actions: store: action "github.com/mozership/probe-s3@v1" must be pinned to a full 40-character commit SHA`},
		{SeverityError, 11, `defaults.cache: unknown key "timout" for the ./declared action`},
		{SeverityError, 16, `with: unknown key "cals" for the ./declared action; did you mean "calls"?`},
		{SeverityError, 18, `unknown action "cash"`},
		{SeverityError, 24, `defaults: cache and ./declared are the same action`},
	})
}

// TestEndToEndActionNames runs a workflow that names its external action
// under actions, and is refused, under a guard, by the action in full.
func TestEndToEndActionNames(t *testing.T) {
	dir := t.TempDir()
	probeBin := filepath.Join(dir, "probe")
	actionDir := filepath.Join(dir, "greet")
	bin := filepath.Join(actionDir, "greet")
	for _, b := range [][]string{{probeBin, "./cmd/probe"}, {bin, "./testdata/external-action"}} {
		if out, err := exec.Command("go", "build", "-o", b[0], b[1]).CombinedOutput(); err != nil {
			t.Fatalf("go build %s: %v\n%s", b[1], err, out)
		}
	}
	writeWorkflow(t, actionDir, "action.yml", "runs:\n  using: binary\n  path: greet\n")
	named := writeWorkflow(t, dir, "workflow.yml", `name: named
actions:
  greet: ./greet
jobs:
- name: greet
  defaults:
    greet:
      name: probe
  steps:
  - name: greet probe
    uses: greet
    test: res.greeting == "hello probe"
`)
	clash := writeWorkflow(t, dir, "clash.yml", `name: clash
actions:
  shell: ./greet
jobs:
- name: greet
  steps:
  - uses: shell
    with:
      name: probe
`)

	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantOut  string
	}{
		{"a named action runs", []string{named}, ExitOK, ""},
		{"the guard names the action in full", []string{"--read-only", named}, ExitConfigError, "--allow-action ./greet"},
		{"and lets it run by that", []string{"--read-only", "--allow-action", "./greet", named}, ExitOK, ""},
		{"but not by the name", []string{"--read-only", "--allow-action", "greet", named}, ExitConfigError, "--allow-action ./greet"},
		{"the name of an action of Probe", []string{clash}, ExitConfigError, `the name "shell" is that of an action of Probe`},
		{"check passes a named action", []string{"check", named}, ExitOK, ""},
		{"check tells the name of an action of Probe", []string{"check", clash}, ExitConfigError, `the name "shell" is that of an action of Probe`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := exec.Command(probeBin, tt.args...).CombinedOutput()
			code := 0
			if err != nil {
				exitErr, ok := err.(*exec.ExitError)
				if !ok {
					t.Fatalf("unexpected error: %v", err)
				}
				code = exitErr.ExitCode()
			}
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d\n%s", code, tt.wantCode, out)
			}
			if !strings.Contains(string(out), tt.wantOut) {
				t.Errorf("output does not contain %q\n%s", tt.wantOut, out)
			}
		})
	}
}
