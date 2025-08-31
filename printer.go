package probe

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

// Color Functions
// colorSuccess returns a *color.Color for success (RGB 0,175,0)
func colorSuccess() *color.Color {
	return color.RGB(0, 175, 0)
}

// colorError returns a *color.Color for errors (red)
func colorError() *color.Color {
	return color.New(color.FgRed)
}

// colorInfo returns a *color.Color for info messages (blue)
func colorInfo() *color.Color {
	return color.New(color.FgBlue)
}

// colorDim returns a *color.Color for subdued text
func colorDim() *color.Color {
	return color.New(color.FgHiBlack)
}

// colorWarning returns a *color.Color for warnings (yellow)
func colorWarning() *color.Color {
	return color.New(color.FgYellow)
}

// colorSkipped returns a *color.Color for skipped items (gray)
func colorSkipped() *color.Color {
	return color.New(color.FgHiBlack)
}

func colorNoTest() *color.Color {
	return color.RGB(0, 102, 204)
}

// String truncation utilities

const (
	// MaxLogStringLength is the maximum length for log output to prevent log bloat
	MaxLogStringLength = 200
	// MaxStringLength is the maximum length for general string processing
	MaxStringLength = 1000000
)

// GetTruncationMessage returns a colored truncation message
func GetTruncationMessage() string {
	return "... [" + colorWarning().Sprintf("⚠︎ probe truncated") + "]"
}

// TruncateString truncates a string if it exceeds the maximum length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + GetTruncationMessage()
}

// TruncateMapStringString truncates long values in map[string]string for logging
func TruncateMapStringString(params map[string]string, maxLen int) map[string]string {
	truncated := make(map[string]string)
	for key, value := range params {
		truncated[key] = TruncateString(value, maxLen)
	}
	return truncated
}

// TruncateMapStringAny truncates long values in map[string]any for logging
func TruncateMapStringAny(params map[string]any, maxLen int) map[string]any {
	truncated := make(map[string]any)
	for key, value := range params {
		switch v := value.(type) {
		case string:
			truncated[key] = TruncateString(v, maxLen)
		default:
			// For non-string values, convert to string first, then truncate
			str := fmt.Sprintf("%v", v)
			truncated[key] = TruncateString(str, maxLen)
		}
	}
	return truncated
}

// Icon constants
const (
	IconSuccess  = "✓ "
	IconError    = "✗ "
	IconTriangle = "△ "
	IconCircle   = "⏺"
	IconWait     = "🕐︎"
	IconSkip     = "⏭ "
)

// LogLevel defines different logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// Printer implements PrintWriter for console print
type Printer struct {
	verbose   bool
	Buffer    map[string]*strings.Builder
	BufferIDs []string // Order preservation
	spinner   *spinner.Spinner
	outWriter io.Writer
	errWriter io.Writer
}

// NewPrinter creates a new console print writer
func NewPrinter(verbose bool, bufferIDs []string) *Printer {
	if os.Getenv("FORCE_COLOR") == "1" || os.Getenv("PROBE_TTY") == "1" {
		color.NoColor = false
	}

	buffer := make(map[string]*strings.Builder)

	// Pre-initialize buffers for all provided job IDs
	for _, id := range bufferIDs {
		buffer[id] = &strings.Builder{}
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	return &Printer{
		verbose:   verbose,
		Buffer:    buffer,
		BufferIDs: bufferIDs, // Store order information
		spinner:   s,
		outWriter: os.Stdout,
		errWriter: os.Stderr,
	}
}

func newBufferPrinter() *Printer {
	pr := NewPrinter(false, []string{})
	pr.outWriter = new(bytes.Buffer)
	pr.errWriter = new(bytes.Buffer)
	return pr
}

func (p *Printer) StartSpinner() {
	if !p.verbose {
		p.spinner.Start()
	}
}

func (p *Printer) StopSpinner() {
	if !p.verbose {
		p.spinner.Stop()
	}
}

func (p *Printer) AddSpinnerSuffix(txt string) {
	if !p.verbose {
		p.spinner.Suffix = fmt.Sprintf(" %s...", txt)
	}
}

func (p *Printer) Fprint(w io.Writer, a ...any) {
	_, err := fmt.Fprint(w, a...)
	if err != nil {
		fmt.Printf("Fprint: %v\n", err)
	}
}

func (p *Printer) Fprintf(w io.Writer, f string, a ...any) {
	_, err := fmt.Fprintf(w, f, a...)
	if err != nil {
		fmt.Printf("Fprintf: %v\n", err)
	}
}

func (p *Printer) Fprintln(w io.Writer, a ...any) {
	_, err := fmt.Fprintln(w, a...)
	if err != nil {
		fmt.Printf("Fprintln: %v\n", err)
	}
}

// printStepRepeatStart prints the start of a repeated step execution
func (p *Printer) printStepRepeatStart(stepIdx int, stepName string, repeatCount int, output *strings.Builder) {
	num := colorDim().Sprintf("%2d.", stepIdx)
	p.Fprintf(output, "%s %s (repeating %d times)\n", num, stepName, repeatCount)
}

// printStepRepeatResult prints the final result of a repeated step execution
func (p *Printer) printStepRepeatResult(counter *StepRepeatCounter, hasTest bool, output *strings.Builder) {
	if hasTest {
		totalCount := counter.SuccessCount + counter.FailureCount
		successRate := float64(counter.SuccessCount) / float64(totalCount) * 100
		statusIcon := colorSuccess().Sprintf(IconSuccess)
		if counter.FailureCount > 0 {
			if counter.SuccessCount == 0 {
				statusIcon = colorError().Sprintf(IconError)
			} else {
				statusIcon = colorNoTest().Sprintf(IconTriangle)
			}
		}

		p.Fprintf(output, "    %s %d/%d success (%.1f%%)\n",
			statusIcon,
			counter.SuccessCount,
			totalCount,
			successRate)
	} else {
		totalCount := counter.SuccessCount + counter.FailureCount
		p.Fprintf(output, "    %s %d/%d completed (no test)\n",
			colorNoTest().Sprintf(IconTriangle),
			totalCount,
			totalCount)
	}
}

// generateJobResults generates buffered job results as string
func (p *Printer) generateJobResults(jobID string, input string, output *strings.Builder) {
	input = strings.TrimSpace(input)
	if input != "" {
		lines := strings.Split(input, "\n")
		for i, line := range lines {
			if strings.TrimSpace(line) != "" {
				if i == 0 {
					p.Fprintf(output, "  ⎿ %s\n", line)
				} else {
					p.Fprintf(output, "    %s\n", line)
				}
			}
		}
	}

	output.WriteString("\n")
}

// generateFooter generates the workflow execution summary as string
func (p *Printer) generateFooter(totalTime float64, successCount, totalJobs int, output *strings.Builder) {
	if successCount == totalJobs {
		p.Fprintf(output, "Total workflow time: %.2fs %s\n",
			totalTime,
			colorSuccess().Sprintf(IconSuccess+"All jobs succeeded"))
	} else {
		failedCount := totalJobs - successCount
		p.Fprintf(output, "Total workflow time: %.2fs %s\n",
			totalTime,
			colorError().Sprintf(IconError+"%d job(s) failed", failedCount))
	}
}

// generateJobStatus generates job status line with appropriate icon and color
func (p *Printer) generateJobStatus(jobID, jobName string, status StatusType, duration float64, output *strings.Builder) {
	var icon string
	var statusText string
	var colorFunc func() *color.Color

	switch status {
	case StatusSuccess:
		icon = IconCircle
		statusText = fmt.Sprintf("Completed in %.2fs", duration)
		colorFunc = colorSuccess
	case StatusError:
		icon = IconCircle
		statusText = fmt.Sprintf("Failed in %.2fs", duration)
		colorFunc = colorError
	case StatusSkipped:
		icon = IconCircle
		statusText = "(SKIPPED)"
		colorFunc = colorSkipped
	default:
		icon = IconCircle
		statusText = fmt.Sprintf("Unknown status in %.2fs", duration)
		colorFunc = colorWarning
	}

	// For skipped jobs, make the entire line gray
	if status == StatusSkipped {
		p.Fprintf(output, "%s %s %s\n",
			colorFunc().Sprint(icon),
			colorFunc().Sprint(jobName),
			colorFunc().Sprint(statusText))
	} else {
		p.Fprintf(output, "%s %s (%s)\n",
			colorFunc().Sprint(icon),
			jobName,
			statusText)
	}
}

// GenerateReport generates a complete workflow report string using Result data
func (p *Printer) GenerateReport(rs *Result) string {
	if rs == nil {
		return ""
	}

	var output strings.Builder
	totalTime := time.Duration(0)
	successCount := 0

	// Generate step results and job summaries for each job in BufferIDs order
	for _, jobID := range p.BufferIDs {
		if jr, exists := rs.Jobs[jobID]; exists {
			jr.mutex.Lock()

			// Calculate job status and duration
			duration := jr.EndTime.Sub(jr.StartTime)
			totalTime += duration

			status := StatusSuccess
			if jr.Status == "skipped" {
				status = StatusSkipped
				successCount++ // Skipped jobs are considered successful
			} else if !jr.Success {
				status = StatusError
			} else {
				successCount++
			}

			// Generate job status output
			p.generateJobStatus(jr.JobID, jr.JobName, status, duration.Seconds(), &output)

			// Generate job results from StepResults
			stepOutput := p.generateJobResultsFromStepResults(jr.StepResults)
			p.generateJobResults(jr.JobID, stepOutput, &output)

			jr.mutex.Unlock()
		}
	}

	// Generate workflow footer
	p.generateFooter(totalTime.Seconds(), successCount, len(rs.Jobs), &output)

	return output.String()
}

func (p *Printer) GenerateReportOnlySteps(rs *Result) string {
	if rs == nil {
		return ""
	}

	var output strings.Builder

	for _, jobID := range p.BufferIDs {
		if jr, exists := rs.Jobs[jobID]; exists {
			jr.mutex.Lock()
			// Generate job results from StepResults
			stepOutput := p.generateJobResultsFromStepResults(jr.StepResults)
			p.generateJobResults(jr.JobID, stepOutput, &output)
			jr.mutex.Unlock()
		}
	}

	return output.String()
}

// PrintReport prints a complete workflow report using Result data
func (p *Printer) PrintReport(rs *Result) {
	reportOutput := p.GenerateReport(rs)
	if reportOutput != "" {
		p.Fprint(p.outWriter, reportOutput)
	}
}

// generateJobResultsFromStepResults creates job output string from StepResults
func (p *Printer) generateJobResultsFromStepResults(stepResults []StepResult) string {
	if len(stepResults) == 0 {
		return ""
	}

	var output strings.Builder

	for _, stepResult := range stepResults {
		// Handle repeat steps
		if stepResult.RepeatCounter != nil {
			p.printStepRepeatStart(stepResult.Index, stepResult.RepeatCounter.Name, stepResult.RepeatCounter.SuccessCount+stepResult.RepeatCounter.FailureCount, &output)
			p.printStepRepeatResult(stepResult.RepeatCounter, stepResult.HasTest, &output)
		} else {
			// Generate regular step output
			num := colorDim().Sprintf("%2d.", stepResult.Index)

			// Add wait time indicator if present
			waitPrefix := ""
			if stepResult.WaitTime != "" {
				waitPrefix = colorDim().Sprintf("%s%s → ", IconWait, stepResult.WaitTime)
			}

			// Add response time suffix if present
			ps := ""
			if stepResult.RT != "" {
				ps = colorDim().Sprintf(" (%s)", stepResult.RT)
			}

			switch stepResult.Status {
			case StatusSuccess:
				p.Fprintf(&output, "%s %s %s%s%s\n", num, colorSuccess().Sprintf(IconSuccess), waitPrefix, stepResult.Name, ps)
			case StatusError:
				if stepResult.TestOutput != "" {
					p.Fprintf(&output, "%s %s %s%s%s\n%s\n", num, colorError().Sprintf(IconError), waitPrefix, stepResult.Name, ps, stepResult.TestOutput)
				} else {
					p.Fprintf(&output, "%s %s %s%s%s\n", num, colorError().Sprintf(IconError), waitPrefix, stepResult.Name, ps)
				}
			case StatusWarning:
				p.Fprintf(&output, "%s %s %s%s%s\n", num, colorNoTest().Sprintf(IconTriangle), waitPrefix, stepResult.Name, ps)
			case StatusSkipped:
				p.Fprintf(&output, "%s %s %s%s%s\n", num, colorInfo().Sprintf(IconSkip), waitPrefix, colorDim().Sprintf("%s", stepResult.Name), ps)
			}

			if stepResult.Report != "" {
				a5space := "     "
				re := strings.ReplaceAll(stepResult.Report, "\n", "\n"+a5space)
				re = strings.TrimRight(re, " \n\t\r")
				output.WriteString(a5space + re + "\n")
			}

			if stepResult.EchoOutput != "" {
				// NOTE:
				// If you apply an ANSI reset code to a string that contains line breaks, the Trim methods will no longer work as expected.
				// Therefore, remove the trailing line breaks first, then apply the color settings and finally add the line break.
				echo := strings.TrimRight(stepResult.EchoOutput, " \n\t\r")
				output.WriteString(colorInfo().Sprint(echo) + "\n")
			}
		}
	}

	return output.String()
}

// generateHeader generates the workflow header string
func (p *Printer) generateHeader(name, description string) string {
	if name == "" {
		return ""
	}

	var output strings.Builder
	bold := color.New(color.Bold)
	output.WriteString(bold.Sprintf("%s\n", name))
	if description != "" {
		output.WriteString(colorDim().Sprintf("%s\n", description))
	}
	output.WriteString("\n")
	return output.String()
}

// PrintHeader prints the workflow name and description
func (p *Printer) PrintHeader(name, description string) {
	header := p.generateHeader(name, description)
	if header != "" {
		p.Fprint(p.outWriter, header)
	}
}

// generateError generates an error message string
func (p *Printer) generateError(format string, args ...interface{}) string {
	return fmt.Sprintf("%s: %s\n", colorError().Sprintf("Error"), fmt.Sprintf(format, args...))
}

// PrintError prints an error message
func (p *Printer) PrintError(format string, args ...interface{}) {
	p.Fprint(p.errWriter, p.generateError(format, args...))
}

// PrintVerbose prints verbose output (only if verbose mode is enabled)
func (p *Printer) PrintVerbose(format string, args ...interface{}) {
	if p.verbose {
		p.Fprintf(p.errWriter, format, args...)
	}
}

// PrintSeparator prints a separator line for verbose output
func (p *Printer) PrintSeparator() {
	if p.verbose {
		p.Fprintln(p.errWriter, "- - -")
	}
}

// generateLogDebug generates debug message string
func (p *Printer) generateLogDebug(format string, args ...interface{}) string {
	return fmt.Sprintf("[DEBUG] %s\n", fmt.Sprintf(format, args...))
}

// LogDebug prints debug messages (only in verbose mode)
func (p *Printer) LogDebug(format string, args ...interface{}) {
	if p.verbose {
		p.Fprint(p.errWriter, p.generateLogDebug(format, args...))
	}
}

// generateLogError generates error log message string
func (p *Printer) generateLogError(format string, args ...interface{}) string {
	return fmt.Sprintf("%s\n", colorError().Sprintf("[ERROR] %s", fmt.Sprintf(format, args...)))
}

// LogError prints error messages to stderr
func (p *Printer) LogError(format string, args ...interface{}) {
	p.Fprint(os.Stderr, p.generateLogError(format, args...))
}

// Step Output Formatting Functions
// These functions handle output formatting with proper separation of concerns

// generateEchoOutput formats echo output with proper indentation
func (p *Printer) generateEchoOutput(content string, err error) string {
	if err != nil {
		return fmt.Sprintf("Echo\nerror: %#v\n", err)
	}

	// Add indent to all lines, including after user-specified newlines
	indent := "       "
	lines := strings.Split(strings.TrimSpace(content), "\n")
	indentedLines := make([]string, len(lines))

	for i, line := range lines {
		indentedLines[i] = indent + line
	}

	return strings.Join(indentedLines, "\n") + "\n"
}

// generateTestFailure formats test failure output with request/response info
func (p *Printer) generateTestFailure(testExpr string, result interface{}, req, res map[string]any) string {
	output := fmt.Sprintf("       %s %#v\n", colorInfo().Sprintf("request:"), req)
	output += fmt.Sprintf("       %s %#v\n", colorInfo().Sprintf("response:"), res)
	return output
}

// generateTestError formats test evaluation error output
func (p *Printer) generateTestError(testExpr string, err error) string {
	if p.verbose {
		p.LogError("Test Error: %s", err)
		p.LogError("Input: %s", testExpr)
	}
	return fmt.Sprintf("Test\nerror: %#v\n", err)
}

// generateTestTypeMismatch formats test type mismatch error output
func (p *Printer) generateTestTypeMismatch(testExpr string, result interface{}) string {
	txt := fmt.Sprintf("Test: `%s` = %v\n", testExpr, result)
	if p.verbose {
		p.LogDebug("%s", txt)
	}
	return txt
}

// PrintTestResult prints test result in verbose mode
func (p *Printer) PrintTestResult(success bool, testExpr string, context interface{}) {
	var resultStr string
	if success {
		resultStr = colorSuccess().Sprintf("Success")
	} else {
		resultStr = colorError().Sprintf("Failure")
	}
	p.LogDebug("Test: %s (input: %s, env: %s)", resultStr, testExpr, colorDim().Sprintf("%#v", context))
}

// PrintEchoContent prints echo content with proper indentation in verbose mode
func (p *Printer) PrintEchoContent(content string) {
	// Add indent to all lines, including after user-specified newlines
	indent := "       "
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		p.LogDebug("%s%s", indent, line)
	}
}

// PrintRequestResponse prints request and response data with proper formatting
func (p *Printer) PrintRequestResponse(stepIdx int, stepName string, req, res map[string]any, rt string) {
	p.LogDebug("%s", colorWarning().Sprintf("--- Step %d: %s", stepIdx, stepName))
	p.LogDebug("Request:")
	p.PrintMapData(req)

	p.LogDebug("Response:")
	p.PrintMapData(res)

	p.LogDebug("RT: %s", colorInfo().Sprintf("%s", rt))
}

// PrintMapData prints map data with proper formatting for nested structures
func (p *Printer) PrintMapData(data map[string]any) {
	p.printMapDataRecursive(data, 1)
}

// printMapDataRecursive recursively prints map data with YAML-like indentation
func (p *Printer) printMapDataRecursive(data map[string]any, indentLevel int) {
	indent := strings.Repeat("  ", indentLevel)

	for k, v := range data {
		switch val := v.(type) {
		case map[string]any:
			if len(val) == 0 {
				p.LogDebug("%s%s: {}", indent, k)
			} else {
				p.LogDebug("%s%s:", indent, k)
				p.printMapDataRecursive(val, indentLevel+1)
			}
		case []any:
			if len(val) == 0 {
				p.LogDebug("%s%s: []", indent, k)
			} else {
				p.LogDebug("%s%s:", indent, k)
				p.printSliceRecursive(val, indentLevel+1)
			}
		case map[string]string:
			if len(val) == 0 {
				p.LogDebug("%s%s: {}", indent, k)
			} else {
				p.LogDebug("%s%s:", indent, k)
				for kk, vv := range val {
					p.LogDebug("%s  %s: %v", indent, kk, vv)
				}
			}
		case []string:
			if len(val) == 0 {
				p.LogDebug("%s%s: []", indent, k)
			} else {
				p.LogDebug("%s%s:", indent, k)
				for _, item := range val {
					p.LogDebug("%s- %v", indent+"  ", item)
				}
			}
		default:
			p.LogDebug("%s%s: %v", indent, k, v)
		}
	}
}

// printSliceRecursive recursively prints slice data with YAML-like indentation
func (p *Printer) printSliceRecursive(data []any, indentLevel int) {
	indent := strings.Repeat("  ", indentLevel)

	for _, v := range data {
		switch val := v.(type) {
		case map[string]any:
			if len(val) == 0 {
				p.LogDebug("%s- {}", indent)
			} else {
				p.LogDebug("%s-", indent)
				p.printMapDataRecursive(val, indentLevel+1)
			}
		case []any:
			if len(val) == 0 {
				p.LogDebug("%s- []", indent)
			} else {
				p.LogDebug("%s-", indent)
				p.printSliceRecursive(val, indentLevel+1)
			}
		case map[string]string:
			if len(val) == 0 {
				p.LogDebug("%s- {}", indent)
			} else {
				p.LogDebug("%s-", indent)
				for kk, vv := range val {
					p.LogDebug("%s  %s: %v", indent, kk, vv)
				}
			}
		case []string:
			if len(val) == 0 {
				p.LogDebug("%s- []", indent)
			} else {
				p.LogDebug("%s-", indent)
				for _, item := range val {
					p.LogDebug("%s  - %v", indent, item)
				}
			}
		default:
			p.LogDebug("%s- %v", indent, v)
		}
	}
}
