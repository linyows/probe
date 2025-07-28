package probe

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

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

// colorWarning returns a *color.Color for warnings (blue)
func colorWarning() *color.Color {
	return color.New(color.FgBlue)
}

// colorDim returns a *color.Color for subdued text
func colorDim() *color.Color {
	return color.New(color.FgHiBlack)
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

// JobBuffer stores buffered output for a job
type JobBuffer struct {
	JobName     string
	JobID       string
	Buffer      strings.Builder
	Status      string
	StartTime   time.Time
	EndTime     time.Time
	Success     bool
	StepResults []StepResult // Store all step results for this job
	mutex       sync.Mutex
}

// WorkflowBuffer manages output for multiple jobs
type WorkflowBuffer struct {
	Jobs map[string]*JobBuffer
}

// NewWorkflowBuffer creates a new WorkflowBuffer instance
func NewWorkflowBuffer() *WorkflowBuffer {
	return &WorkflowBuffer{
		Jobs: make(map[string]*JobBuffer),
	}
}

// AddStepResult adds a StepResult to the specified job buffer
func (wb *WorkflowBuffer) AddStepResult(jobID string, stepResult StepResult) {
	if jb, exists := wb.Jobs[jobID]; exists {
		jb.mutex.Lock()
		defer jb.mutex.Unlock()
		jb.StepResults = append(jb.StepResults, stepResult)
	}
}

// GetStepResults returns all step results for the specified job
func (wb *WorkflowBuffer) GetStepResults(jobID string) []StepResult {
	if jb, exists := wb.Jobs[jobID]; exists {
		jb.mutex.Lock()
		defer jb.mutex.Unlock()
		// Return a copy to avoid race conditions
		results := make([]StepResult, len(jb.StepResults))
		copy(results, jb.StepResults)
		return results
	}
	return nil
}

// PrintWriter defines the interface for different print implementations
type PrintWriter interface {
	// Workflow level output
	PrintHeader(name, description string)
	PrintJobName(name string)

	// Step level output
	PrintStepRepeatStart(jobID string, stepIdx int, stepName string, repeatCount int)
	PrintStepRepeatResult(jobID string, stepIdx int, counter StepRepeatCounter, hasTest bool)

	// Job level output
	PrintJobStatus(jobID string, jobName string, status StatusType, duration float64)
	PrintJobResults(jobID string, output string)

	// Workflow summary
	PrintFooter(totalTime float64, successCount, totalJobs int)

	// Error output
	PrintError(format string, args ...interface{})

	// Verbose output
	PrintVerbose(format string, args ...interface{})
	PrintSeparator()

	// Unified logging methods
	LogDebug(format string, args ...interface{})
	LogInfo(format string, args ...interface{})
	LogWarn(format string, args ...interface{})
	LogError(format string, args ...interface{})

	PrintReport(wb *WorkflowBuffer)
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
	mutex     sync.RWMutex
}

// NewPrinter creates a new console print writer
func NewPrinter(verbose bool, bufferIDs []string) *Printer {
	buffer := make(map[string]*strings.Builder)

	// Pre-initialize buffers for all provided job IDs
	for _, id := range bufferIDs {
		buffer[id] = &strings.Builder{}
	}

	return &Printer{
		verbose:   verbose,
		Buffer:    buffer,
		BufferIDs: bufferIDs, // Store order information
	}
}

// PrintReport prints a complete workflow report using WorkflowBuffer data
func (p *Printer) PrintReport(wb *WorkflowBuffer) {
	if wb == nil {
		return
	}

	totalTime := time.Duration(0)
	successCount := 0

	// Print step results and job summaries for each job in BufferIDs order
	for _, jobID := range p.BufferIDs {
		if jb, exists := wb.Jobs[jobID]; exists {
			jb.mutex.Lock()

			// Calculate job status and duration
			duration := jb.EndTime.Sub(jb.StartTime)
			totalTime += duration

			status := StatusSuccess
			if jb.Status == "Skipped" {
				status = StatusWarning
			} else if !jb.Success {
				status = StatusError
			} else {
				successCount++
			}

			// Print job status
			p.PrintJobStatus(jb.JobID, jb.JobName, status, duration.Seconds())

			// Generate and print job results from StepResults instead of using os.Pipe buffer
			stepOutput := p.generateJobResultsFromStepResults(jb.StepResults)
			p.PrintJobResults(jb.JobID, stepOutput)

			jb.mutex.Unlock()
		}
	}

	// Print workflow footer
	p.PrintFooter(totalTime.Seconds(), successCount, len(wb.Jobs))
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
			// Generate repeat step start output
			totalCount := stepResult.RepeatCounter.SuccessCount + stepResult.RepeatCounter.FailureCount
			stepName := stepResult.Name
			if strings.HasSuffix(stepName, " (SKIPPED)") {
				stepName = strings.TrimSuffix(stepName, " (SKIPPED)")
			}

			num := colorDim().Sprintf("%2d.", stepResult.Index)
			output.WriteString(fmt.Sprintf("%s %s (repeating %d times)\n", num, stepName, totalCount))

			// Generate repeat step result output
			hasTest := stepResult.HasTest
			if hasTest {
				successRate := float64(stepResult.RepeatCounter.SuccessCount) / float64(totalCount) * 100
				statusIcon := colorSuccess().Sprintf(IconSuccess)
				if stepResult.RepeatCounter.FailureCount > 0 {
					if stepResult.RepeatCounter.SuccessCount == 0 {
						statusIcon = colorError().Sprintf(IconError)
					} else {
						statusIcon = colorWarning().Sprintf(IconWarning)
					}
				}

				output.WriteString(fmt.Sprintf("    %s %d/%d success (%.1f%%)\n",
					statusIcon,
					stepResult.RepeatCounter.SuccessCount,
					totalCount,
					successRate))
			} else {
				output.WriteString(fmt.Sprintf("    %s %d/%d completed (no test)\n",
					colorWarning().Sprintf(IconWarning),
					totalCount,
					totalCount))
			}
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
				output.WriteString(fmt.Sprintf("%s %s %s%s%s\n", num, colorSuccess().Sprintf(IconSuccess), waitPrefix, stepResult.Name, ps))
			case StatusError:
				if stepResult.TestOutput != "" {
					output.WriteString(fmt.Sprintf("%s %s %s%s%s\n%s\n", num, colorError().Sprintf(IconError), waitPrefix, stepResult.Name, ps, stepResult.TestOutput))
				} else {
					output.WriteString(fmt.Sprintf("%s %s %s%s%s\n", num, colorError().Sprintf(IconError), waitPrefix, stepResult.Name, ps))
				}
			case StatusWarning:
				output.WriteString(fmt.Sprintf("%s %s %s%s%s\n", num, colorWarning().Sprintf(IconWarning), waitPrefix, stepResult.Name, ps))
			case StatusSkipped:
				output.WriteString(fmt.Sprintf("%s %s %s%s%s\n", num, colorWarning().Sprintf(IconSkip), waitPrefix, colorDim().Sprintf("%s", stepResult.Name), ps))
			}

			if stepResult.EchoOutput != "" {
				output.WriteString(stepResult.EchoOutput)
			}
		}
	}

	return output.String()
}

// printRepeatStepFromResult prints repeat step information from StepResult
func (p *Printer) printRepeatStepFromResult(jobID string, stepResult StepResult) {
	if stepResult.RepeatCounter == nil {
		return
	}

	counter := *stepResult.RepeatCounter
	hasTest := stepResult.HasTest

	// For repeat steps, we need to extract the base name (remove SKIPPED suffix if present)
	stepName := stepResult.Name
	if strings.HasSuffix(stepName, " (SKIPPED)") {
		stepName = strings.TrimSuffix(stepName, " (SKIPPED)")
	}

	// Print repeat start - use total count from counter
	totalCount := counter.SuccessCount + counter.FailureCount
	p.PrintStepRepeatStart(jobID, stepResult.Index, stepName, totalCount)

	// Print repeat result
	p.PrintStepRepeatResult(jobID, stepResult.Index, counter, hasTest)
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

// PrintJobName prints the job name
func (p *Printer) PrintJobName(name string) {
	fmt.Printf("%s\n", name)
}

// PrintStepRepeatStart prints the start of a repeated step execution
func (p *Printer) PrintStepRepeatStart(jobID string, stepIdx int, stepName string, repeatCount int) {
	num := colorDim().Sprintf("%2d.", stepIdx)
	output := fmt.Sprintf("%s %s (repeating %d times)\n", num, stepName, repeatCount)

	fmt.Print(output)
}

// PrintStepRepeatResult prints the final result of a repeated step execution
func (p *Printer) PrintStepRepeatResult(jobID string, stepIdx int, counter StepRepeatCounter, hasTest bool) {
	var output string

	if hasTest {
		totalCount := counter.SuccessCount + counter.FailureCount
		successRate := float64(counter.SuccessCount) / float64(totalCount) * 100
		statusIcon := colorSuccess().Sprintf(IconSuccess)
		if counter.FailureCount > 0 {
			if counter.SuccessCount == 0 {
				statusIcon = colorError().Sprintf(IconError)
			} else {
				statusIcon = colorWarning().Sprintf(IconWarning)
			}
		}

		output = fmt.Sprintf("    %s %d/%d success (%.1f%%)\n",
			statusIcon,
			counter.SuccessCount,
			totalCount,
			successRate)
	} else {
		totalCount := counter.SuccessCount + counter.FailureCount
		output = fmt.Sprintf("    %s %d/%d completed (no test)\n",
			colorWarning().Sprintf(IconWarning),
			totalCount,
			totalCount)
	}

	fmt.Print(output)
}

// PrintJobStatus prints the result of a job execution
func (p *Printer) PrintJobStatus(jobID string, jobName string, status StatusType, duration float64) {
	statusColor := colorSuccess()
	statusIcon := IconCircle

	switch status {
	case StatusError:
		statusColor = colorError()
	case StatusWarning:
		statusColor = colorWarning()
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
	output := fmt.Sprintf("%s%s %s\n",
		statusColor.Sprint(statusIcon),
		jobName,
		dt)

	fmt.Print(output)
}

// PrintJobResults prints buffered job results
func (p *Printer) PrintJobResults(jobID string, output string) {
	txt := ""

	output = strings.TrimSpace(output)
	if output != "" {
		lines := strings.Split(output, "\n")
		for i, line := range lines {
			if strings.TrimSpace(line) != "" {
				if i == 0 {
					txt += fmt.Sprintf("  ⎿ %s\n", line)
				} else {
					txt += fmt.Sprintf("    %s\n", line)
				}
			}
		}
	}

	txt += "\n"
	fmt.Print(txt)
}

// PrintFooter prints the workflow execution summary
func (p *Printer) PrintFooter(totalTime float64, successCount, totalJobs int) {
	if successCount == totalJobs {
		fmt.Printf("Total workflow time: %.2fs %s\n",
			totalTime,
			colorSuccess().Sprintf(IconSuccess+"All jobs succeeded"))
	} else {
		failedCount := totalJobs - successCount
		fmt.Printf("Total workflow time: %.2fs %s\n",
			totalTime,
			colorError().Sprintf(IconError+"%d job(s) failed", failedCount))
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

// LogInfo prints informational messages
func (p *Printer) LogInfo(format string, args ...interface{}) {
	fmt.Printf("[INFO] %s\n", fmt.Sprintf(format, args...))
}

// LogWarn prints warning messages to stderr
func (p *Printer) LogWarn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s\n", colorWarning().Sprintf("[WARN] %s", fmt.Sprintf(format, args...)))
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

// PrintJobName does nothing in silent mode
func (s *SilentPrinter) PrintJobName(name string) {}

// PrintStepRepeatStart does nothing in silent mode
func (s *SilentPrinter) PrintStepRepeatStart(jobID string, stepIdx int, stepName string, repeatCount int) {
}

// PrintStepRepeatResult does nothing in silent mode
func (s *SilentPrinter) PrintStepRepeatResult(jobID string, stepIdx int, counter StepRepeatCounter, hasTest bool) {
}

// PrintJobStatus does nothing in silent mode
func (s *SilentPrinter) PrintJobStatus(jobID string, jobName string, status StatusType, duration float64) {
}

// PrintJobResults does nothing in silent mode
func (s *SilentPrinter) PrintJobResults(jobID string, output string) {}

// PrintFooter does nothing in silent mode
func (s *SilentPrinter) PrintFooter(totalTime float64, successCount, totalJobs int) {}

// PrintError does nothing in silent mode
func (s *SilentPrinter) PrintError(format string, args ...interface{}) {}

// PrintVerbose does nothing in silent mode
func (s *SilentPrinter) PrintVerbose(format string, args ...interface{}) {}

// PrintSeparator does nothing in silent mode
func (s *SilentPrinter) PrintSeparator() {}

// LogDebug does nothing in silent mode
func (s *SilentPrinter) LogDebug(format string, args ...interface{}) {}

// LogInfo does nothing in silent mode
func (s *SilentPrinter) LogInfo(format string, args ...interface{}) {}

// LogWarn does nothing in silent mode
func (s *SilentPrinter) LogWarn(format string, args ...interface{}) {}

// LogError does nothing in silent mode
func (s *SilentPrinter) LogError(format string, args ...interface{}) {}

func (s *SilentPrinter) PrintReport(wb *WorkflowBuffer) {}
