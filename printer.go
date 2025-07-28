package probe

import (
	"fmt"
	"os"
	"strings"
	"sync"
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

// JobResult stores execution results for a job
type JobResult struct {
	JobName     string
	JobID       string
	Status      string
	StartTime   time.Time
	EndTime     time.Time
	Success     bool
	StepResults []StepResult // Store all step results for this job
	mutex       sync.Mutex
}

// Result manages execution results for multiple jobs
type Result struct {
	Jobs map[string]*JobResult
}

// NewResult creates a new Result instance
func NewResult() *Result {
	return &Result{
		Jobs: make(map[string]*JobResult),
	}
}

// AddStepResult adds a StepResult to the specified job result
func (rs *Result) AddStepResult(jobID string, stepResult StepResult) {
	if jr, exists := rs.Jobs[jobID]; exists {
		jr.mutex.Lock()
		defer jr.mutex.Unlock()
		jr.StepResults = append(jr.StepResults, stepResult)
	}
}

// PrintWriter defines the interface for different print implementations
type PrintWriter interface {
	// Workflow level output
	PrintHeader(name, description string)

	// Error output
	PrintError(format string, args ...interface{})

	// Verbose output
	PrintVerbose(format string, args ...interface{})
	PrintSeparator()

	// Unified logging methods
	LogDebug(format string, args ...interface{})
	LogError(format string, args ...interface{})

	PrintReport(rs *Result)
	StartSpinner()
	StopSpinner()
	AddSpinnerSuffix(txt string)
}

// StatusType represents the status of execution
type StatusType int

const (
	StatusSuccess StatusType = iota
	StatusError
	StatusWarning
	StatusSkipped
)

// StepResult represents the result of a step execution
type StepResult struct {
	Index         int
	Name          string
	Status        StatusType
	RT            string
	WaitTime      string
	TestOutput    string
	EchoOutput    string
	HasTest       bool
	RepeatCounter *StepRepeatCounter // For repeat execution information
}

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

// PrintHeader prints the workflow name and description
func (p *Printer) PrintHeader(name, description string) {
	if name != "" {
		bold := color.New(color.Bold)
		bold.Printf("%s\n", name)
		if description != "" {
			colorDim().Printf("%s\n", description)
		}
		fmt.Println("")
	}
}

// PrintError prints an error message
func (p *Printer) PrintError(format string, args ...interface{}) {
	fmt.Printf("%s: %s\n", colorError().Sprintf("Error"), fmt.Sprintf(format, args...))
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

// LogDebug prints debug messages (only in verbose mode)
func (p *Printer) LogDebug(format string, args ...interface{}) {
	if p.verbose {
		fmt.Printf("[DEBUG] %s\n", fmt.Sprintf(format, args...))
	}
}

// LogError prints error messages to stderr
func (p *Printer) LogError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s\n", colorError().Sprintf("[ERROR] %s", fmt.Sprintf(format, args...)))
}

// SilentPrinter is a no-op implementation of PrintWriter for testing
type SilentPrinter struct{}

// NewSilentPrinter creates a new silent print writer for testing
func NewSilentPrinter() *SilentPrinter {
	return &SilentPrinter{}
}

// PrintHeader does nothing in silent mode
func (s *SilentPrinter) PrintHeader(name, description string) {}

// AddSpinnerSuffix does nothing in silent mode
func (s *SilentPrinter) AddSpinnerSuffix(txt string) {
}

// StartSpinner does nothing in silent mode
func (s *SilentPrinter) StartSpinner() {
}

// StopSpinner does nothing in silent mode
func (s *SilentPrinter) StopSpinner() {
}

// PrintError does nothing in silent mode
func (s *SilentPrinter) PrintError(format string, args ...interface{}) {}

// PrintVerbose does nothing in silent mode
func (s *SilentPrinter) PrintVerbose(format string, args ...interface{}) {}

// PrintSeparator does nothing in silent mode
func (s *SilentPrinter) PrintSeparator() {}

// LogDebug does nothing in silent mode
func (s *SilentPrinter) LogDebug(format string, args ...interface{}) {}

// LogError does nothing in silent mode
func (s *SilentPrinter) LogError(format string, args ...interface{}) {}

func (s *SilentPrinter) PrintReport(rs *Result) {}
