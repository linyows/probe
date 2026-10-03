package probe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// maxStepSummaryBytes is the most GitHub accepts in one step's job summary.
// A summary over it is rejected as a whole, so Probe keeps under it.
const maxStepSummaryBytes = 1024 * 1024

// ErrNoStepSummary is returned for a github-summary report when no path was
// given and GITHUB_STEP_SUMMARY is not set, as on a machine outside GitHub
// Actions. It is not a failure: the same command line can run anywhere.
var ErrNoStepSummary = errors.New("GITHUB_STEP_SUMMARY is not set; skipping the github-summary report")

// ErrStepSummaryFull is returned when earlier writes have left no room in the
// job summary for even a shortened page. Writing anyway would push it past
// GitHub's limit and lose what the earlier steps wrote, so nothing is written.
var ErrStepSummaryFull = errors.New("the job summary has no room left under GitHub's 1 MiB limit; skipping the github-summary report")

// writeGitHubSummary appends the Markdown page to the job summary file. It
// appends because earlier steps, and other tools in the same step, write to
// the same file. When the page would push the file past GitHub's limit, it
// leaves out the requests and responses, and if that is still too large it
// cuts the page short and says so.
func (r *Report) writeGitHubSummary(path string) error {
	if path != "" {
		// An explicit path can sit in a checked-out project, where a symlink
		// would make the append land in a file elsewhere. The summary is
		// appended to, so it cannot be replaced by a rename; refuse instead.
		// The path GitHub Actions provides is trusted.
		if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to append the job summary to %s: it is a symlink", path)
		}
	} else {
		path = os.Getenv("GITHUB_STEP_SUMMARY")
	}
	if path == "" {
		return ErrNoStepSummary
	}

	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create the job summary directory: %w", err)
		}
	}

	var existing int64
	if info, err := os.Stat(path); err == nil {
		existing = info.Size()
	}
	// A blank line separates this page from whatever the file already holds,
	// and it counts against the limit too.
	sep := ""
	if existing > 0 {
		sep = "\n"
	}
	page := fitStepSummary(r, maxStepSummaryBytes-int(existing)-len(sep))
	if page == "" {
		return ErrStepSummaryFull
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open the job summary %s: %w", path, err)
	}
	_, err = f.WriteString(sep + page)
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

// fitStepSummary renders the page so that it is at most budget bytes, or
// returns an empty string when not even a shortened page fits.
func fitStepSummary(r *Report, budget int) string {
	if page := r.markdown(true); len(page) <= budget {
		return page
	}
	page, cuts := r.markdownWithCuts(false)
	if len(page)+len(payloadsOmittedNote) <= budget {
		return page + payloadsOmittedNote
	}

	// Keep everything before the last cut point that leaves room for the
	// note. Cut points fall between blocks, so no code fence is left open.
	limit := budget - len(truncatedNote)
	end := -1
	for _, c := range cuts {
		if c <= limit && c <= len(page) {
			end = c
		}
	}
	if end <= 0 {
		return ""
	}
	return page[:end] + truncatedNote
}
