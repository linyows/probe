package probe

import (
	"fmt"
	"io"
	"strings"
)

// WriteMarkdown writes the report as a Markdown page: a summary line, a table
// of jobs, and a section for every failed step with its test, the reason it
// failed, and the request and response. The page is meant to be read as is,
// for example as a GitHub job summary or by a coding agent.
func (r *Report) WriteMarkdown(w io.Writer) error {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n\n", r.Name)
	if r.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", strings.TrimSpace(r.Description))
	}

	fmt.Fprintf(&b, "**%s** in %.2fs. Jobs: %s. Steps: %s.\n\n",
		markdownStatus(r.Status), msToSec(r.DurationMs),
		countPhrase(r.Summary.Jobs), countPhrase(r.Summary.Steps))

	b.WriteString("| Job | Status | Steps | Duration |\n")
	b.WriteString("|---|---|---|---|\n")
	for _, job := range r.Jobs {
		var passed int
		for _, st := range job.Steps {
			if st.Status == ReportPassed {
				passed++
			}
		}
		fmt.Fprintf(&b, "| %s | %s | %d/%d | %.2fs |\n",
			markdownCell(job.Name), job.Status, passed, len(job.Steps), msToSec(job.DurationMs))
	}

	failures := false
	for _, job := range r.Jobs {
		for _, st := range job.Steps {
			if st.Status != ReportFailed {
				continue
			}
			if !failures {
				b.WriteString("\n## Failures\n\n")
				failures = true
			}
			writeMarkdownFailure(&b, job, st)
		}
	}

	// Every block ends in a blank line; the page itself ends in one newline.
	_, err := io.WriteString(w, strings.TrimRight(b.String(), "\n")+"\n")
	return err
}

func writeMarkdownFailure(b *strings.Builder, job JobReport, st StepReport) {
	fmt.Fprintf(b, "### %s / %d. %s\n\n", job.Name, st.Index, st.Name)

	if st.Failure != nil {
		fmt.Fprintf(b, "%s: %s\n\n", st.Failure.Kind, st.Failure.Message)
	}
	if st.Repeat != nil {
		fmt.Fprintf(b, "%d of %d iterations succeeded.\n\n", st.Repeat.Success, st.Repeat.Total)
	}
	if st.Test != "" {
		writeFence(b, "", st.Test)
	}
	if st.Failure == nil {
		return
	}
	for _, part := range []struct {
		label string
		data  map[string]any
	}{
		{"Request", st.Failure.Request},
		{"Response", st.Failure.Response},
	} {
		if part.data == nil {
			continue
		}
		fmt.Fprintf(b, "<details><summary>%s</summary>\n\n", part.label)
		writeFence(b, "json", indentJSON(part.data))
		b.WriteString("</details>\n\n")
	}
}

// writeFence writes content in a fenced code block whose fence is longer than
// any backtick run inside it, so the content cannot close the block early.
func writeFence(b *strings.Builder, lang, content string) {
	fence := "```"
	for strings.Contains(content, fence) {
		fence += "`"
	}
	fmt.Fprintf(b, "%s%s\n%s\n%s\n\n", fence, lang, strings.TrimRight(content, "\n"), fence)
}

func markdownStatus(status string) string {
	if status == ReportFailed {
		return "Failed"
	}
	return "Passed"
}

// countPhrase renders a tally such as "3 total, 2 passed, 1 failed", leaving
// out the statuses nothing has.
func countPhrase(c ReportCount) string {
	parts := []string{fmt.Sprintf("%d total", c.Total)}
	for _, p := range []struct {
		n     int
		label string
	}{
		{c.Passed, "passed"},
		{c.Failed, "failed"},
		{c.Skipped, "skipped"},
		{c.Untested, "untested"},
	} {
		if p.n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", p.n, p.label))
		}
	}
	return strings.Join(parts, ", ")
}

// markdownCell keeps a value on one table row and out of the column syntax.
func markdownCell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "|", `\|`)
}
