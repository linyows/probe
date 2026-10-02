package probe

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseReportTargets(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      []ReportTarget
		expectErr string
	}{
		{
			name:  "empty",
			input: "",
			want:  nil,
		},
		{
			name:  "format without a path uses the default file",
			input: "json",
			want:  []ReportTarget{{Format: ReportJSON, Path: "probe-report.json"}},
		},
		{
			name:  "every format with and without paths",
			input: "junit=out/junit.xml, JSON ,markdown=summary.md",
			want: []ReportTarget{
				{Format: ReportJUnit, Path: "out/junit.xml"},
				{Format: ReportJSON, Path: "probe-report.json"},
				{Format: ReportMarkdown, Path: "summary.md"},
			},
		},
		{
			name:  "empty entries are ignored",
			input: ",junit,,",
			want:  []ReportTarget{{Format: ReportJUnit, Path: "probe-junit.xml"}},
		},
		{
			name:  "empty path falls back to the default",
			input: "markdown=",
			want:  []ReportTarget{{Format: ReportMarkdown, Path: "probe-report.md"}},
		},
		{
			name:      "unknown format",
			input:     "json,html=out.html",
			expectErr: "unknown report format: html",
		},
		{
			name:      "same format twice",
			input:     "json=a.json,json=b.json",
			expectErr: "report format given more than once: json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseReportTargets(tt.input)
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tt.expectErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

// newReportResult builds a Result covering every status a report has to map.
func newReportResult() (*Result, []string) {
	start := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	rs := NewResult()

	rs.Jobs["api"] = &JobResult{
		JobID:     "api",
		JobName:   "API",
		Status:    "Failed",
		StartTime: start,
		EndTime:   start.Add(1500 * time.Millisecond),
		Success:   false,
		StepResults: []StepResult{
			{Index: 0, Name: "Health", Status: StatusSuccess, Test: "res.code == 200", Elapsed: 120 * time.Millisecond, EchoOutput: "       line 1\n       line 2\n"},
			{Index: 1, Name: "Create", Status: StatusError, Test: "res.code == 201", Elapsed: 80 * time.Millisecond,
				RetryAttempt: 3, RetryMax: 3,
				Failure: &StepFailure{
					Kind:     FailureAssertion,
					Message:  "test evaluated to false",
					Request:  map[string]any{"url": "http://example.test/items"},
					Response: map[string]any{"code": 500, "body": "boom ``` ]]>"},
				}},
			{Index: 2, Name: "Connect", Status: StatusError, Failure: &StepFailure{Kind: FailureAction, Message: "connection refused"}},
			{Index: 3, Name: "Cleanup (SKIPPED)", Status: StatusSkipped},
			{Index: 4, Name: "Note", Status: StatusWarning},
		},
	}
	rs.Jobs["after"] = &JobResult{
		JobID:     "after",
		JobName:   "After | API",
		Status:    "skipped",
		StartTime: start,
		EndTime:   start,
		Success:   true,
	}
	rs.Jobs["poll"] = &JobResult{
		JobID:     "poll",
		JobName:   "Poll",
		Status:    "Failed",
		StartTime: start,
		EndTime:   start.Add(3 * time.Second),
		Success:   false,
		StepResults: []StepResult{
			{Index: 0, Name: "Tick", Status: StatusWarning, HasTest: true, Test: "res.code == 0",
				RepeatCounter: &StepRepeatCounter{Name: "Tick", SuccessCount: 2, FailureCount: 1, RepeatTotal: 3}},
		},
	}
	rs.Jobs["setup"] = &JobResult{
		JobID:     "setup",
		JobName:   "Setup",
		Status:    "Failed",
		StartTime: start,
		EndTime:   start.Add(10 * time.Millisecond),
		Success:   false,
	}

	return rs, []string{"api", "after", "poll", "setup"}
}

func newTestReport() *Report {
	rs, order := newReportResult()
	start := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	return BuildReport("Demo", "A demo run", rs, order, start, start.Add(3*time.Second))
}

func TestBuildReport(t *testing.T) {
	r := newTestReport()

	if r.Status != ReportFailed {
		t.Errorf("Status = %q, want %q", r.Status, ReportFailed)
	}
	if r.DurationMs != 3000 {
		t.Errorf("DurationMs = %d, want 3000", r.DurationMs)
	}

	wantJobs := ReportCount{Total: 4, Failed: 3, Skipped: 1}
	if r.Summary.Jobs != wantJobs {
		t.Errorf("Summary.Jobs = %+v, want %+v", r.Summary.Jobs, wantJobs)
	}
	wantSteps := ReportCount{Total: 6, Passed: 1, Failed: 3, Skipped: 1, Untested: 1}
	if r.Summary.Steps != wantSteps {
		t.Errorf("Summary.Steps = %+v, want %+v", r.Summary.Steps, wantSteps)
	}

	var ids []string
	for _, j := range r.Jobs {
		ids = append(ids, j.ID)
	}
	if want := []string{"api", "after", "poll", "setup"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("job order = %v, want %v", ids, want)
	}

	steps := r.Jobs[0].Steps
	if steps[0].Echo != "line 1\nline 2" {
		t.Errorf("echo = %q, want the terminal indentation removed", steps[0].Echo)
	}
	if steps[0].DurationMs != 120 {
		t.Errorf("DurationMs = %d, want 120", steps[0].DurationMs)
	}
	if steps[1].Retry == nil || *steps[1].Retry != (RetryReport{Attempts: 3, Max: 3}) {
		t.Errorf("Retry = %+v, want 3/3", steps[1].Retry)
	}
	if steps[1].Failure == nil || steps[1].Failure.Response["code"] != 500 {
		t.Errorf("Failure = %+v, want the response carried over", steps[1].Failure)
	}
	if steps[3].Name != "Cleanup" || steps[3].Status != ReportSkipped {
		t.Errorf("skipped step = %q/%q, want the SKIPPED marker removed from the name", steps[3].Name, steps[3].Status)
	}
	if steps[4].Status != ReportUntested {
		t.Errorf("step without a test = %q, want %q", steps[4].Status, ReportUntested)
	}

	repeated := r.Jobs[2].Steps[0]
	if repeated.Status != ReportFailed {
		t.Errorf("repeat with a failing iteration = %q, want %q", repeated.Status, ReportFailed)
	}
	if repeated.Repeat == nil || *repeated.Repeat != (RepeatReport{Total: 3, Success: 2, Failure: 1}) {
		t.Errorf("Repeat = %+v, want 2 of 3", repeated.Repeat)
	}
}

func TestBuildReport_StatusMapping(t *testing.T) {
	tests := []struct {
		name string
		sr   StepResult
		want string
	}{
		{"success", StepResult{Status: StatusSuccess}, ReportPassed},
		{"error", StepResult{Status: StatusError}, ReportFailed},
		{"skipped", StepResult{Status: StatusSkipped}, ReportSkipped},
		{"no test", StepResult{Status: StatusWarning}, ReportUntested},
		{"repeat all passed", StepResult{Status: StatusSuccess, HasTest: true, RepeatCounter: &StepRepeatCounter{SuccessCount: 3}}, ReportPassed},
		{"repeat without a test", StepResult{Status: StatusSuccess, RepeatCounter: &StepRepeatCounter{SuccessCount: 3}}, ReportUntested},
		{"repeat all failed", StepResult{Status: StatusError, HasTest: true, RepeatCounter: &StepRepeatCounter{FailureCount: 3}}, ReportFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildStepReport(tt.sr).Status; got != tt.want {
				t.Errorf("status = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildReport_FailureOnlyOnFailedSteps(t *testing.T) {
	sr := StepResult{Status: StatusSuccess, Failure: &StepFailure{Kind: FailureAssertion}}
	if got := buildStepReport(sr).Failure; got != nil {
		t.Errorf("Failure = %+v, want nil for a passing step", got)
	}
}

func TestBuildReport_NilResult(t *testing.T) {
	r := BuildReport("Empty", "", nil, []string{"a"}, time.Time{}, time.Time{})
	if r.Status != ReportPassed || len(r.Jobs) != 0 {
		t.Errorf("got status %q with %d jobs, want an empty passing report", r.Status, len(r.Jobs))
	}

	var buf bytes.Buffer
	if err := r.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"jobs": []`) {
		t.Errorf("jobs should encode as an empty array, got:\n%s", buf.String())
	}
}

func TestJSONSafeMap(t *testing.T) {
	m := map[string]any{
		"ok":     "text",
		"bad":    make(chan int),
		"nested": map[string]any{"fn": func() {}, "n": 1},
	}

	safe := jsonSafeMap(m)
	if _, err := json.Marshal(safe); err != nil {
		t.Fatalf("result should encode, got %v", err)
	}
	if safe["ok"] != "text" {
		t.Errorf("encodable values should be kept, got %v", safe["ok"])
	}
	if nested, _ := safe["nested"].(map[string]any); nested["n"] != 1 {
		t.Errorf("encodable nested values should be kept, got %v", safe["nested"])
	}

	plain := map[string]any{"a": 1}
	if got := jsonSafeMap(plain); reflect.ValueOf(got).Pointer() != reflect.ValueOf(plain).Pointer() {
		t.Error("an encodable map should be returned as is")
	}
	if jsonSafeMap(nil) != nil {
		t.Error("nil should stay nil")
	}
}

func TestReport_WriteJSON(t *testing.T) {
	r := newTestReport()

	var buf bytes.Buffer
	if err := r.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}

	var decoded Report
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if decoded.Summary != r.Summary {
		t.Errorf("summary = %+v, want %+v", decoded.Summary, r.Summary)
	}

	out := buf.String()
	for _, want := range []string{`"status": "failed"`, `"kind": "assertion"`, `"duration_ms": 3000`} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON should contain %s", want)
		}
	}
	if strings.Contains(out, `"failure": null`) || strings.Contains(out, `"retry": null`) {
		t.Error("absent optional fields should be omitted, not null")
	}
}

func TestReport_WriteJUnit(t *testing.T) {
	r := newTestReport()

	var buf bytes.Buffer
	if err := r.WriteJUnit(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.HasPrefix(out, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("JUnit output should start with the XML header")
	}

	var doc junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("output is not valid XML: %v\n%s", err, out)
	}

	// api: 5 steps; after: 1 synthetic skipped case; poll: 1 step; setup: 1 synthetic error case
	if doc.Tests != 8 || doc.Failures != 2 || doc.Errors != 2 || doc.Skipped != 2 {
		t.Errorf("totals = tests %d failures %d errors %d skipped %d, want 8/2/2/2",
			doc.Tests, doc.Failures, doc.Errors, doc.Skipped)
	}
	if len(doc.Suites) != 4 {
		t.Fatalf("suites = %d, want 4", len(doc.Suites))
	}

	api := doc.Suites[0]
	if api.Name != "API" || api.Timestamp != "2026-10-02T12:00:00Z" || api.Time != "1.500" {
		t.Errorf("suite = %s %s %s, want API 2026-10-02T12:00:00Z 1.500", api.Name, api.Timestamp, api.Time)
	}
	create := api.Cases[1]
	if create.Failure == nil || create.Failure.Type != FailureAssertion {
		t.Fatalf("a false test should be a <failure>, got %+v", create)
	}
	if !strings.Contains(create.Failure.Body, "boom ``` ]]>") {
		t.Errorf("failure body should survive CDATA intact, got %q", create.Failure.Body)
	}
	if !strings.Contains(create.Failure.Body, "test: res.code == 201") {
		t.Errorf("failure body should name the test, got %q", create.Failure.Body)
	}
	if connect := api.Cases[2]; connect.Error == nil || connect.Error.Type != FailureAction {
		t.Errorf("an action error should be an <error>, got %+v", connect)
	}
	if api.Cases[3].Skipped == nil {
		t.Error("a skipped step should carry <skipped>")
	}
	if out := api.Cases[0].SystemOut; out == nil || out.Text != "line 1\nline 2" {
		t.Errorf("echo should land in system-out, got %+v", out)
	}

	if after := doc.Suites[1]; len(after.Cases) != 1 || after.Cases[0].Skipped == nil {
		t.Errorf("a job skipped before any step should get one skipped case, got %+v", after.Cases)
	}
	if poll := doc.Suites[2].Cases[0]; poll.Failure == nil || poll.Failure.Message != "1 of 3 iterations failed" {
		t.Errorf("a failing repeat should report its iterations, got %+v", poll.Failure)
	}
	if setup := doc.Suites[3]; len(setup.Cases) != 1 || setup.Cases[0].Error == nil {
		t.Errorf("a job that failed without a failed step should get one error case, got %+v", setup.Cases)
	}
}

func TestReport_WriteMarkdown(t *testing.T) {
	r := newTestReport()

	var buf bytes.Buffer
	if err := r.WriteMarkdown(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	for _, want := range []string{
		"# Demo\n\nA demo run\n\n",
		"**Failed** in 3.00s. Jobs: 4 total, 3 failed, 1 skipped. Steps: 6 total, 1 passed, 3 failed, 1 skipped, 1 untested.\n",
		"| API | failed | 1/5 | 1.50s |\n",
		"| After \\| API | skipped | 0/0 | 0.00s |\n",
		"## Failures\n\n### API / 1. Create\n\nassertion: test evaluated to false\n\n```\nres.code == 201\n```\n",
		"<details><summary>Response</summary>\n\n````json\n",
		"### API / 2. Connect\n\naction: connection refused\n",
		"### Poll / 0. Tick\n\n2 of 3 iterations succeeded.\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Markdown should contain %q, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Request</summary>") && !strings.Contains(out, `"url": "http://example.test/items"`) {
		t.Error("request block should hold the request")
	}
	if strings.Contains(out, "### API / 2. Connect\n\naction: connection refused\n\n<details>") {
		t.Error("a failure without a request or response should not render empty blocks")
	}
	if !strings.HasSuffix(out, "\n") || strings.HasSuffix(out, "\n\n") {
		t.Error("the page should end in exactly one newline")
	}
}

func TestReport_WriteMarkdown_NoFailures(t *testing.T) {
	r := &Report{Name: "Green", Status: ReportPassed, Summary: ReportSummary{Jobs: ReportCount{Total: 1, Passed: 1}}}

	var buf bytes.Buffer
	if err := r.WriteMarkdown(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if strings.Contains(out, "## Failures") {
		t.Errorf("a passing run should have no failures section, got:\n%s", out)
	}
	if !strings.Contains(out, "**Passed** in 0.00s. Jobs: 1 total, 1 passed. Steps: 0 total.") {
		t.Errorf("summary line missing, got:\n%s", out)
	}
}

func TestWriteFence(t *testing.T) {
	var b strings.Builder
	writeFence(&b, "json", "a ```` b")
	if want := "`````json\na ```` b\n`````\n\n"; b.String() != want {
		t.Errorf("got %q, want %q", b.String(), want)
	}
}

func TestReport_Write(t *testing.T) {
	r := newTestReport()
	dir := t.TempDir()

	for _, format := range []ReportFormat{ReportJSON, ReportJUnit, ReportMarkdown} {
		path := filepath.Join(dir, "nested", "dir", string(format))
		if err := r.Write(ReportTarget{Format: format, Path: path}); err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		if len(data) == 0 {
			t.Errorf("%s: file is empty", format)
		}
	}

	// A directory where the file should go cannot be created over.
	blocked := filepath.Join(dir, "blocked")
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	err := r.Write(ReportTarget{Format: ReportJSON, Path: blocked})
	if err == nil || !strings.Contains(err.Error(), "failed to create json report") {
		t.Errorf("error = %v, want a create failure", err)
	}
}

// TestWorkflow_StartWritesReports runs a workflow end to end and checks the
// report files reflect what each step did.
func TestWorkflow_StartWritesReports(t *testing.T) {
	runner := NewMockActionRunner()
	runner.SetResult("ok", map[string]any{
		"req": map[string]any{"path": "/ok"},
		"res": map[string]any{"code": 200},
	})
	runner.SetResult("secret", map[string]any{
		"req": map[string]any{"token": "hunter2"},
		"res": map[string]any{"code": 500, "dump": false},
	})
	runner.SetError("down", errors.New("connection refused"))

	workflow := &Workflow{
		Name: "Report Run",
		Jobs: []Job{
			{
				ID:   "checks",
				Name: "Checks",
				Steps: []*Step{
					{Name: "Passes", Uses: "ok", Test: "res.code == 200", actionRunner: runner},
					{Name: "Fails", Uses: "ok", Test: "res.code == 201", actionRunner: runner},
					{Name: "Hidden", Uses: "secret", Test: "res.code == 200", actionRunner: runner},
					{Name: "Errors", Uses: "down", actionRunner: runner},
				},
			},
			{
				ID:    "later",
				Name:  "Later",
				Needs: []string{"checks"},
				Steps: []*Step{{Name: "Never", Uses: "ok", actionRunner: runner}},
			},
		},
		printer: newBufferPrinter(),
	}

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "report.json")
	junitPath := filepath.Join(dir, "junit.xml")
	mdPath := filepath.Join(dir, "report.md")

	err := workflow.Start(Config{Reports: []ReportTarget{
		{Format: ReportJSON, Path: jsonPath},
		{Format: ReportJUnit, Path: junitPath},
		{Format: ReportMarkdown, Path: mdPath},
	}})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	var r Report
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if r.Name != "Report Run" || r.Status != ReportFailed {
		t.Errorf("report = %q/%q, want Report Run/failed", r.Name, r.Status)
	}
	if len(r.Jobs) != 2 || r.Jobs[0].ID != "checks" || r.Jobs[1].Status != ReportSkipped {
		t.Fatalf("jobs = %+v, want checks then a skipped later", r.Jobs)
	}

	steps := r.Jobs[0].Steps
	if len(steps) != 4 {
		t.Fatalf("steps = %d, want 4", len(steps))
	}
	if steps[0].Status != ReportPassed || steps[0].Test != "res.code == 200" {
		t.Errorf("step 0 = %+v, want a passed step with its test", steps[0])
	}

	fails := steps[1].Failure
	if fails == nil || fails.Kind != FailureAssertion || fails.Request["path"] != "/ok" {
		t.Errorf("step 1 failure = %+v, want an assertion with the request", fails)
	}

	hidden := steps[2].Failure
	if hidden == nil || hidden.Request != nil || hidden.Response != nil {
		t.Errorf("step 2 failure = %+v, want no request or response when dump is false", hidden)
	}
	if strings.Contains(string(data), "hunter2") {
		t.Error("a response marked dump: false must not reach the report")
	}

	errored := steps[3].Failure
	if errored == nil || errored.Kind != FailureAction || !strings.Contains(errored.Message, "connection refused") {
		t.Errorf("step 3 failure = %+v, want the action error", errored)
	}

	for _, p := range []string{junitPath, mdPath} {
		if info, err := os.Stat(p); err != nil || info.Size() == 0 {
			t.Errorf("%s was not written: %v", p, err)
		}
	}
}

func TestWorkflow_StartReportsWriteError(t *testing.T) {
	workflow := &Workflow{
		Name:    "Unwritable",
		Jobs:    []Job{{Name: "job", Steps: []*Step{}}},
		printer: newBufferPrinter(),
	}

	dir := t.TempDir()
	good := filepath.Join(dir, "report.md")

	err := workflow.Start(Config{Reports: []ReportTarget{
		{Format: ReportJSON, Path: dir}, // a directory, so the file cannot be created
		{Format: ReportMarkdown, Path: good},
	}})
	if err == nil || !strings.Contains(err.Error(), "json report") {
		t.Errorf("error = %v, want the json report failure", err)
	}
	if _, statErr := os.Stat(good); statErr != nil {
		t.Errorf("a failing report should not stop the others: %v", statErr)
	}
}

func TestStep_DoTestRecordsFailure(t *testing.T) {
	tests := []struct {
		name     string
		test     string
		wantKind string
		wantMsg  string
	}{
		{"passes", "res.code == 200", "", ""},
		{"false", "res.code == 201", FailureAssertion, "test evaluated to false"},
		{"not a boolean", "res.code", FailureTestType, "test evaluated to 200 (int), not a boolean"},
		{"cannot evaluate", "res.code ==", FailureTestError, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &Step{
				Test: tt.test,
				Expr: &Expr{},
				ctx: StepContext{
					Req: map[string]any{"path": "/"},
					Res: map[string]any{"code": 200},
				},
				// A failure left over from an earlier evaluation must not leak.
				failure: &StepFailure{Kind: "stale"},
			}

			st.DoTest(newBufferPrinter())

			if tt.wantKind == "" {
				if st.failure != nil {
					t.Errorf("failure = %+v, want nil", st.failure)
				}
				return
			}
			if st.failure == nil || st.failure.Kind != tt.wantKind {
				t.Fatalf("failure = %+v, want kind %q", st.failure, tt.wantKind)
			}
			if tt.wantMsg != "" && st.failure.Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", st.failure.Message, tt.wantMsg)
			}
			if tt.wantMsg == "" && st.failure.Message == "" {
				t.Error("message should explain the failure")
			}
		})
	}
}
