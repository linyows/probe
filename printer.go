package probe

import (
	"fmt"
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

// Icon constants
const (
	IconSuccess = "✔︎ "
	IconError   = "✘ "
	IconWarning = "▲ "
	IconCircle  = "⏺ "
	IconWait    = "🕐︎"
	IconSkip    = "⏭ "
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
}

// NewPrinter creates a new console print writer
func NewPrinter(verbose bool, bufferIDs []string) *Printer {
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
	}
}

func (p *Printer) StartSpinner() {
	p.spinner.Start()
}

func (p *Printer) StopSpinner() {
	p.spinner.Stop()
}

func (p *Printer) AddSpinnerSuffix(txt string) {
	p.spinner.Suffix = fmt.Sprintf(" %s...", txt)
}

// printStepRepeatStart prints the start of a repeated step execution
func (p *Printer) printStepRepeatStart(stepIdx int, stepName string, repeatCount int, output *strings.Builder) {
	num := colorDim().Sprintf("%2d.", stepIdx)
	fmt.Fprintf(output, "%s %s (repeating %d times)\n", num, stepName, repeatCount)
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
				statusIcon = colorInfo().Sprintf(IconWarning)
			}
		}

		fmt.Fprintf(output, "    %s %d/%d success (%.1f%%)\n",
			statusIcon,
			counter.SuccessCount,
			totalCount,
			successRate)
	} else {
		totalCount := counter.SuccessCount + counter.FailureCount
		fmt.Fprintf(output, "    %s %d/%d completed (no test)\n",
			colorInfo().Sprintf(IconWarning),
			totalCount,
			totalCount)
	}
}

// generateJobStatus generates the result of a job execution as string
func (p *Printer) generateJobStatus(jobID string, jobName string, status StatusType, duration float64, output *strings.Builder) {
	statusColor := colorSuccess()
	statusIcon := IconCircle

	switch status {
	case StatusError:
		statusColor = colorError()
	case StatusWarning:
		statusColor = colorInfo()
	}

	statusStr := ""
	switch status {
	case StatusSuccess:
		statusStr = "Completed"
	case StatusError:
		statusStr = "Failed"
	case StatusWarning:
		statusStr = "Skipped"
	}

	dt := colorDim().Sprintf("(%s in %.2fs)",
		statusStr,
		duration)
	outputLine := fmt.Sprintf("%s%s %s\n",
		statusColor.Sprint(statusIcon),
		jobName,
		dt)

	output.WriteString(outputLine)
}

// generateJobResults generates buffered job results as string
func (p *Printer) generateJobResults(jobID string, input string, output *strings.Builder) {
	input = strings.TrimSpace(input)
	if input != "" {
		lines := strings.Split(input, "\n")
		for i, line := range lines {
			if strings.TrimSpace(line) != "" {
				if i == 0 {
					fmt.Fprintf(output, "  ⎿ %s\n", line)
				} else {
					fmt.Fprintf(output, "    %s\n", line)
				}
			}
		}
	}

	output.WriteString("\n")
}

// generateFooter generates the workflow execution summary as string
func (p *Printer) generateFooter(totalTime float64, successCount, totalJobs int, output *strings.Builder) {
	if successCount == totalJobs {
		fmt.Fprintf(output, "Total workflow time: %.2fs %s\n",
			totalTime,
			colorSuccess().Sprintf(IconSuccess+"All jobs succeeded"))
	} else {
		failedCount := totalJobs - successCount
		fmt.Fprintf(output, "Total workflow time: %.2fs %s\n",
			totalTime,
			colorError().Sprintf(IconError+"%d job(s) failed", failedCount))
	}
}

// generateReport generates a complete workflow report string using Result data
func (p *Printer) generateReport(rs *Result) string {
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
			if jr.Status == "Skipped" {
				status = StatusWarning
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

// PrintReport prints a complete workflow report using Result data
func (p *Printer) PrintReport(rs *Result) {
	reportOutput := p.generateReport(rs)
	if reportOutput != "" {
		fmt.Print(reportOutput)
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
				fmt.Fprintf(&output, "%s %s %s%s%s\n", num, colorSuccess().Sprintf(IconSuccess), waitPrefix, stepResult.Name, ps)
			case StatusError:
				if stepResult.TestOutput != "" {
					fmt.Fprintf(&output, "%s %s %s%s%s\n%s\n", num, colorError().Sprintf(IconError), waitPrefix, stepResult.Name, ps, stepResult.TestOutput)
				} else {
					fmt.Fprintf(&output, "%s %s %s%s%s\n", num, colorError().Sprintf(IconError), waitPrefix, stepResult.Name, ps)
				}
			case StatusWarning:
				fmt.Fprintf(&output, "%s %s %s%s%s\n", num, colorInfo().Sprintf(IconWarning), waitPrefix, stepResult.Name, ps)
			case StatusSkipped:
				fmt.Fprintf(&output, "%s %s %s%s%s\n", num, colorInfo().Sprintf(IconSkip), waitPrefix, colorDim().Sprintf("%s", stepResult.Name), ps)
			}

			if stepResult.EchoOutput != "" {
				output.WriteString(stepResult.EchoOutput)
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
		fmt.Print(header)
	}
}

// generateError generates an error message string
func (p *Printer) generateError(format string, args ...interface{}) string {
	return fmt.Sprintf("%s: %s\n", colorError().Sprintf("Error"), fmt.Sprintf(format, args...))
}

// PrintError prints an error message
func (p *Printer) PrintError(format string, args ...interface{}) {
	fmt.Print(p.generateError(format, args...))
}

// PrintVerbose prints verbose output (only if verbose mode is enabled)
func (p *Printer) PrintVerbose(format string, args ...interface{}) {
	if p.verbose {
		fmt.Printf(format, args...)
	}
}

// PrintSeparator prints a separator line for verbose output
func (p *Printer) PrintSeparator() {
	if p.verbose {
		fmt.Println("- - -")
	}
}

// generateLogDebug generates debug message string
func (p *Printer) generateLogDebug(format string, args ...interface{}) string {
	return fmt.Sprintf("[DEBUG] %s\n", fmt.Sprintf(format, args...))
}

// LogDebug prints debug messages (only in verbose mode)
func (p *Printer) LogDebug(format string, args ...interface{}) {
	if p.verbose {
		fmt.Print(p.generateLogDebug(format, args...))
	}
}

// generateLogError generates error log message string
func (p *Printer) generateLogError(format string, args ...interface{}) string {
	return fmt.Sprintf("%s\n", colorError().Sprintf("[ERROR] %s", fmt.Sprintf(format, args...)))
}

// LogError prints error messages to stderr
func (p *Printer) LogError(format string, args ...interface{}) {
	fmt.Fprint(os.Stderr, p.generateLogError(format, args...))
}

