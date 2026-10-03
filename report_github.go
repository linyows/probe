package probe

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// maxStepSummaryBytes is the most GitHub accepts in one step's job summary.
// A summary over it is rejected as a whole, so Probe keeps under it.
const maxStepSummaryBytes = 1024 * 1024

// ErrNoStepSummary is returned for a github-summary report when no path was
// given and GITHUB_STEP_SUMMARY is not set, as on a machine outside GitHub
// Actions. It is not a failure: the same command line can run anywhere.
var ErrNoStepSummary = errors.New("GITHUB_STEP_SUMMARY is not set; skipping the github-summary report")

// writeGitHubSummary appends the Markdown page to the job summary file. It
// appends because earlier steps, and other tools in the same step, write to
// the same file. When the page would push the file past GitHub's limit, it
// leaves out the requests and responses, and if that is still too large it
// cuts the page short and says so.
func (r *Report) writeGitHubSummary(path string) error {
	if path == "" {
		path = os.Getenv("GITHUB_STEP_SUMMARY")
	}
	if path == "" {
		return ErrNoStepSummary
	}

	var existing int64
	if info, err := os.Stat(path); err == nil {
		existing = info.Size()
	}
	page := fitStepSummary(r, maxStepSummaryBytes-int(existing))

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open the job summary %s: %w", path, err)
	}
	// Separate this page from whatever the file already holds.
	if existing > 0 {
		page = "\n" + page
	}
	_, err = f.WriteString(page)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("failed to write the job summary %s: %w", path, err)
	}
	return nil
}

const (
	payloadsOmittedNote = "\n_Requests and responses are left out to keep the job summary within GitHub's size limit. Write a json or markdown report to keep them._\n"
	truncatedNote       = "\n_The job summary was cut short to stay within GitHub's size limit. Write a json or markdown report for the full result._\n"
)

// fitStepSummary renders the page so that it fits in budget bytes.
func fitStepSummary(r *Report, budget int) string {
	// One byte is kept for the newline that separates this page from
	// earlier content.
	budget--

	if page := r.markdown(true); len(page) <= budget {
		return page
	}
	if page := r.markdown(false) + payloadsOmittedNote; len(page) <= budget {
		return page
	}

	page := r.markdown(false)
	limit := budget - len(truncatedNote)
	if limit <= 0 {
		return ""
	}
	if limit < len(page) {
		page = cutMarkdown(page, limit)
	}
	return page + truncatedNote
}

// cutMarkdown shortens page to at most limit bytes. It cuts before a failure
// section when it can, since a section holds a code fence that cut in the
// middle would swallow everything after it, and otherwise at a line break.
func cutMarkdown(page string, limit int) string {
	page = page[:limit]
	if i := strings.LastIndex(page, "\n### "); i >= 0 {
		return page[:i+1]
	}
	if i := strings.LastIndexByte(page, '\n'); i >= 0 {
		return page[:i+1]
	}
	return ""
}
