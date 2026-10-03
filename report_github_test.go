package probe

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseReportTargets_GitHubSummary(t *testing.T) {
	got, err := ParseReportTargets("github-summary,markdown")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Format != ReportGitHubSummary || got[0].Path != "" {
		t.Errorf("github-summary target = %+v, want an empty path resolved at write time", got[0])
	}

	got, err = ParseReportTargets("github-summary=out/summary.md")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Path != "out/summary.md" {
		t.Errorf("path = %q, want the one given", got[0].Path)
	}
}

func TestReport_WriteGitHubSummary_Appends(t *testing.T) {
	r := newTestReport()
	path := filepath.Join(t.TempDir(), "summary.md")
	if err := os.WriteFile(path, []byte("## Earlier step\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := r.Write(ReportTarget{Format: ReportGitHubSummary, Path: path}); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)
	if !strings.HasPrefix(out, "## Earlier step\n\n# Demo\n") {
		t.Errorf("the page should follow earlier content after a blank line, got:\n%.80s", out)
	}
	if n := strings.Count(out, "# Demo\n"); n != 2 {
		t.Errorf("found the page %d times, want 2 (appended, not overwritten)", n)
	}
	if !strings.Contains(out, "<details><summary>Response</summary>") {
		t.Error("a summary within the limit should keep requests and responses")
	}
}

func TestReport_WriteGitHubSummary_FromEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "step_summary")
	t.Setenv("GITHUB_STEP_SUMMARY", path)

	if err := newTestReport().Write(ReportTarget{Format: ReportGitHubSummary}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "# Demo\n") {
		t.Errorf("a new summary should start with the page, got:\n%.80s", data)
	}
}

func TestReport_WriteGitHubSummary_NoEnv(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", "")

	err := newTestReport().Write(ReportTarget{Format: ReportGitHubSummary})
	if !errors.Is(err, ErrNoStepSummary) {
		t.Errorf("error = %v, want ErrNoStepSummary", err)
	}
}

func TestReport_WriteGitHubSummary_Unwritable(t *testing.T) {
	err := newTestReport().Write(ReportTarget{Format: ReportGitHubSummary, Path: t.TempDir()})
	if err == nil || errors.Is(err, ErrNoStepSummary) {
		t.Errorf("error = %v, want a write failure", err)
	}
}

func TestReport_MarkdownWithoutPayloads(t *testing.T) {
	r := newTestReport()
	full := r.markdown(true)
	compact := r.markdown(false)

	if !strings.Contains(full, "<details>") {
		t.Fatal("the full page should carry the request and response")
	}
	if strings.Contains(compact, "<details>") {
		t.Error("the compact page should leave out the request and response")
	}
	if !strings.Contains(compact, "### API / 1. Create\n\nassertion: test evaluated to false\n\n```\nres.code == 201\n```\n") {
		t.Errorf("the compact page should keep the failure and its test, got:\n%s", compact)
	}

	var buf bytes.Buffer
	if err := r.WriteMarkdown(&buf); err != nil {
		t.Fatal(err)
	}
	if buf.String() != full {
		t.Error("WriteMarkdown should write the full page")
	}
}

func TestFitStepSummary(t *testing.T) {
	r := newTestReport()
	full := r.markdown(true)
	compact := r.markdown(false)

	t.Run("fits as is", func(t *testing.T) {
		if got := fitStepSummary(r, len(full)+1); got != full {
			t.Error("a page within the budget should be written unchanged")
		}
	})

	t.Run("drops payloads", func(t *testing.T) {
		got := fitStepSummary(r, len(full))
		if got != compact+payloadsOmittedNote {
			t.Errorf("got:\n%s", got)
		}
	})

	t.Run("truncates at a section", func(t *testing.T) {
		budget := len(compact) - 20
		got := fitStepSummary(r, budget)
		if len(got) > budget-1 {
			t.Errorf("page is %d bytes, over the budget of %d", len(got), budget-1)
		}
		if !strings.HasSuffix(got, truncatedNote) {
			t.Errorf("a cut page should say so, got:\n%s", got)
		}
		body := strings.TrimSuffix(got, truncatedNote)
		if strings.Count(body, "```")%2 != 0 {
			t.Errorf("the cut left a code fence open:\n%s", body)
		}
		if !strings.HasSuffix(body, "\n") {
			t.Error("the cut should end on a line break")
		}
	})

	t.Run("no room", func(t *testing.T) {
		if got := fitStepSummary(r, 10); got != "" {
			t.Errorf("with no room the page should be empty, got %q", got)
		}
	})
}

func TestCutMarkdown(t *testing.T) {
	page := "# T\n\nline\n\n### A\n\n```\nx\n```\n\n### B\n\n```\ny\n```\n"
	if got := cutMarkdown(page, len(page)-3); got != "# T\n\nline\n\n### A\n\n```\nx\n```\n\n" {
		t.Errorf("should cut before the last section, got %q", got)
	}
	if got := cutMarkdown("# T\n\nline one\nline two", 15); got != "# T\n\nline one\n" {
		t.Errorf("without a section should cut at a line break, got %q", got)
	}
	if got := cutMarkdown("no newline at all", 5); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

// TestWorkflow_GitHubSummaryOutsideActions checks that asking for a job
// summary where there is none warns but neither fails the run nor stops the
// other reports.
func TestWorkflow_GitHubSummaryOutsideActions(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", "")

	w := &Workflow{
		Name:    "summary",
		Jobs:    []Job{{Name: "job", Steps: []*Step{}}},
		printer: newBufferPrinter(),
	}
	md := filepath.Join(t.TempDir(), "report.md")

	err := w.Start(Config{Reports: []ReportTarget{
		{Format: ReportGitHubSummary},
		{Format: ReportMarkdown, Path: md},
	}})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if w.exitStatus != ExitOK {
		t.Errorf("exit status = %d, want %d", w.exitStatus, ExitOK)
	}
	if _, err := os.Stat(md); err != nil {
		t.Errorf("the markdown report should still be written: %v", err)
	}
	stderr := w.printer.errWriter.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "[WARN] GITHUB_STEP_SUMMARY is not set") {
		t.Errorf("expected a warning, got:\n%s", stderr)
	}
}
