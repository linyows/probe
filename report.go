package probe

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReportFormat names a report file format.
type ReportFormat string

const (
	ReportJSON     ReportFormat = "json"
	ReportJUnit    ReportFormat = "junit"
	ReportMarkdown ReportFormat = "markdown"
)

// defaultReportPaths is where a format is written when no path is given.
var defaultReportPaths = map[ReportFormat]string{
	ReportJSON:     "probe-report.json",
	ReportJUnit:    "probe-junit.xml",
	ReportMarkdown: "probe-report.md",
}

// ReportTarget is one report file to write after a run.
type ReportTarget struct {
	Format ReportFormat
	Path   string
}

// ParseReportTargets parses a comma separated list of format[=path] entries,
// such as "junit=out/junit.xml,json". A format without a path is written to
// its default file name in the current directory.
func ParseReportTargets(s string) ([]ReportTarget, error) {
	var targets []ReportTarget
	seen := make(map[ReportFormat]bool)

	for _, entry := range strings.Split(s, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		name, path, _ := strings.Cut(entry, "=")
		format := ReportFormat(strings.ToLower(strings.TrimSpace(name)))
		path = strings.TrimSpace(path)

		def, ok := defaultReportPaths[format]
		if !ok {
			return nil, fmt.Errorf("unknown report format: %s (expected json, junit or markdown)", name)
		}
		if seen[format] {
			return nil, fmt.Errorf("report format given more than once: %s", format)
		}
		seen[format] = true

		if path == "" {
			path = def
		}
		targets = append(targets, ReportTarget{Format: format, Path: path})
	}

	return targets, nil
}

// Report statuses for the workflow, jobs and steps.
const (
	ReportPassed   = "passed"
	ReportFailed   = "failed"
	ReportSkipped  = "skipped"
	ReportUntested = "untested" // A step that ran but has no test expression
)

// Report is the result of a workflow run in a form meant for machines: the
// JSON file is this structure as is, and the other formats are rendered from it.
type Report struct {
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Status      string        `json:"status"`
	StartedAt   time.Time     `json:"started_at"`
	FinishedAt  time.Time     `json:"finished_at"`
	DurationMs  int64         `json:"duration_ms"`
	Summary     ReportSummary `json:"summary"`
	Jobs        []JobReport   `json:"jobs"`
}

// ReportSummary counts jobs and steps by status.
type ReportSummary struct {
	Jobs  ReportCount `json:"jobs"`
	Steps ReportCount `json:"steps"`
}

// ReportCount is a tally by status.
type ReportCount struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	Skipped  int `json:"skipped"`
	Untested int `json:"untested,omitempty"`
}

// JobReport is one job of a Report.
type JobReport struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Status     string       `json:"status"`
	StartedAt  time.Time    `json:"started_at"`
	DurationMs int64        `json:"duration_ms"`
	Steps      []StepReport `json:"steps"`
}

// StepReport is one step of a JobReport.
type StepReport struct {
	Index      int            `json:"index"`
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	Test       string         `json:"test,omitempty"`
	DurationMs int64          `json:"duration_ms"`
	Retry      *RetryReport   `json:"retry,omitempty"`
	Repeat     *RepeatReport  `json:"repeat,omitempty"`
	Echo       string         `json:"echo,omitempty"`
	Failure    *FailureReport `json:"failure,omitempty"`
}

// RetryReport records how many attempts a retried step took.
type RetryReport struct {
	Attempts int `json:"attempts"`
	Max      int `json:"max"`
}

// RepeatReport records the outcome of every iteration of a repeated job.
type RepeatReport struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failure int `json:"failure"`
}

// FailureReport says why a step failed.
type FailureReport struct {
	Kind     string         `json:"kind"`
	Message  string         `json:"message"`
	Request  map[string]any `json:"request,omitempty"`
	Response map[string]any `json:"response,omitempty"`
}

// BuildReport assembles a Report from the results of a run. Jobs are listed in
// the given order, which is the order they are declared in the workflow.
func BuildReport(name, description string, rs *Result, order []string, startedAt, finishedAt time.Time) *Report {
	r := &Report{
		Name:        name,
		Description: description,
		Status:      ReportPassed,
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
		DurationMs:  finishedAt.Sub(startedAt).Milliseconds(),
		Jobs:        []JobReport{},
	}
	if rs == nil {
		return r
	}

	for _, id := range order {
		jr, ok := rs.Jobs[id]
		if !ok {
			continue
		}
		job := buildJobReport(jr)
		r.Jobs = append(r.Jobs, job)

		r.Summary.Jobs.add(job.Status)
		for _, st := range job.Steps {
			r.Summary.Steps.add(st.Status)
		}
		if job.Status == ReportFailed {
			r.Status = ReportFailed
		}
	}

	return r
}

func (c *ReportCount) add(status string) {
	c.Total++
	switch status {
	case ReportPassed:
		c.Passed++
	case ReportFailed:
		c.Failed++
	case ReportSkipped:
		c.Skipped++
	case ReportUntested:
		c.Untested++
	}
}

func buildJobReport(jr *JobResult) JobReport {
	jr.mutex.Lock()
	defer jr.mutex.Unlock()

	status := ReportPassed
	switch {
	case jr.Status == "skipped":
		status = ReportSkipped
	case !jr.Success:
		status = ReportFailed
	}

	job := JobReport{
		ID:         jr.JobID,
		Name:       jr.JobName,
		Status:     status,
		StartedAt:  jr.StartTime,
		DurationMs: jr.EndTime.Sub(jr.StartTime).Milliseconds(),
		Steps:      make([]StepReport, 0, len(jr.StepResults)),
	}
	for _, sr := range jr.StepResults {
		job.Steps = append(job.Steps, buildStepReport(sr))
	}

	return job
}

func buildStepReport(sr StepResult) StepReport {
	step := StepReport{
		Index:      sr.Index,
		Name:       sr.Name,
		Test:       sr.Test,
		DurationMs: sr.Elapsed.Milliseconds(),
	}

	switch sr.Status {
	case StatusSuccess:
		step.Status = ReportPassed
	case StatusError:
		step.Status = ReportFailed
	case StatusSkipped:
		step.Status = ReportSkipped
		// The terminal report marks a skipped step in its name; here the
		// status already says so.
		step.Name = strings.TrimSuffix(sr.Name, " (SKIPPED)")
	default:
		step.Status = ReportUntested
	}

	if sr.RetryAttempt > 0 {
		step.Retry = &RetryReport{Attempts: sr.RetryAttempt, Max: sr.RetryMax}
	}

	if c := sr.RepeatCounter; c != nil {
		step.Name = c.Name
		step.Repeat = &RepeatReport{
			Total:   c.SuccessCount + c.FailureCount,
			Success: c.SuccessCount,
			Failure: c.FailureCount,
		}
		// The terminal report marks a mix of passing and failing iterations as
		// a warning. Any failing iteration fails the job, so it fails the step.
		switch {
		case c.FailureCount > 0:
			step.Status = ReportFailed
		case sr.HasTest:
			step.Status = ReportPassed
		default:
			step.Status = ReportUntested
		}
	} else {
		step.Echo = unindentEcho(sr.EchoOutput)
	}

	if f := sr.Failure; f != nil && step.Status == ReportFailed {
		step.Failure = &FailureReport{
			Kind:     f.Kind,
			Message:  f.Message,
			Request:  jsonSafeMap(f.Request),
			Response: jsonSafeMap(f.Response),
		}
	}

	return step
}

// unindentEcho strips the indentation the terminal report puts on echo lines.
func unindentEcho(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimPrefix(line, "       ")
	}
	return strings.Join(lines, "\n")
}

// jsonSafeMap returns m unchanged when it encodes as JSON, and otherwise a
// copy in which every value that does not encode is replaced by its %v form.
// Action results are plain data, so the fallback only guards against a
// report that cannot be written at all.
func jsonSafeMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	if _, err := json.Marshal(m); err == nil {
		return m
	}

	safe := make(map[string]any, len(m))
	for k, v := range m {
		if _, err := json.Marshal(v); err != nil {
			if nested, ok := v.(map[string]any); ok {
				safe[k] = jsonSafeMap(nested)
				continue
			}
			safe[k] = fmt.Sprintf("%v", v)
			continue
		}
		safe[k] = v
	}
	return safe
}

// WriteJSON writes the report as indented JSON.
func (r *Report) WriteJSON(w io.Writer) error {
	// Responses are full of HTML and comparisons; the report is not embedded
	// in a page, so keep <, > and & readable.
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// Write renders the report in the target's format and writes it to the
// target's path, creating parent directories as needed.
func (r *Report) Write(t ReportTarget) error {
	if dir := filepath.Dir(t.Path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create report directory: %w", err)
		}
	}

	f, err := os.Create(t.Path)
	if err != nil {
		return fmt.Errorf("failed to create %s report: %w", t.Format, err)
	}

	switch t.Format {
	case ReportJSON:
		err = r.WriteJSON(f)
	case ReportJUnit:
		err = r.WriteJUnit(f)
	case ReportMarkdown:
		err = r.WriteMarkdown(f)
	default:
		err = fmt.Errorf("unknown report format: %s", t.Format)
	}

	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("failed to write %s report to %s: %w", t.Format, t.Path, err)
	}
	return nil
}

// failureDetail renders a failure as plain text for formats that embed it in
// a text block: the test, the message, then the request and response.
func failureDetail(st StepReport) string {
	var b strings.Builder
	if st.Test != "" {
		fmt.Fprintf(&b, "test: %s\n", st.Test)
	}
	if st.Repeat != nil {
		fmt.Fprintf(&b, "repeat: %d/%d succeeded\n", st.Repeat.Success, st.Repeat.Total)
	}
	if st.Failure == nil {
		return b.String()
	}
	fmt.Fprintf(&b, "%s: %s\n", st.Failure.Kind, st.Failure.Message)
	if st.Failure.Request != nil {
		fmt.Fprintf(&b, "\nrequest:\n%s\n", indentJSON(st.Failure.Request))
	}
	if st.Failure.Response != nil {
		fmt.Fprintf(&b, "\nresponse:\n%s\n", indentJSON(st.Failure.Response))
	}
	return b.String()
}

func indentJSON(v any) string {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Sprintf("%v", v)
	}
	return strings.TrimRight(b.String(), "\n")
}

func msToSec(ms int64) float64 {
	return float64(ms) / 1000
}
