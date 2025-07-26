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

// JobPrinter stores buffered output for a job
type JobPrinter struct {
	JobName   string
	JobID     string
	Buffer    strings.Builder
	Status    string
	StartTime time.Time
	EndTime   time.Time
	Success   bool
	mutex     sync.Mutex
}

// WorkflowPrinter manages output for multiple jobs
type WorkflowPrinter struct {
	Jobs        map[string]*JobPrinter
	mutex       sync.RWMutex //nolint:unused // Reserved for future concurrent access
	outputMutex sync.Mutex   // Protects stdout redirection
}

// NewWorkflowPrinter creates a new WorkflowPrinter instance
func NewWorkflowPrinter() *WorkflowPrinter {
	return &WorkflowPrinter{
		Jobs: make(map[string]*JobPrinter),
	}
}

// PrintWriter defines the interface for different print implementations
type PrintWriter interface {
	// Workflow level output
	PrintHeader(name, description string)
	PrintJobName(name string)

	// Step level output
	PrintStepResult(step StepResult)
	PrintStepRepeatStart(stepIdx int, stepName string, repeatCount int)
	PrintStepRepeatResult(stepIdx int, counter StepRepeatCounter, hasTest bool)

	// Job level output
	PrintJobResult(jobName string, status StatusType, duration float64)
	PrintJobResults(output string)

	// Workflow summary
	PrintWorkflowSummary(totalTime float64, successCount, totalJobs int)

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
	Index      int
	Name       string
	Status     StatusType
	RT         string
	WaitTime   string
	TestOutput string
	EchoOutput string
	HasTest    bool
}

// Printer implements PrintWriter for console print
type Printer struct {
	verbose bool
}

// NewPrinter creates a new console print writer
func NewPrinter(verbose bool) *Printer {
	return &Printer{
		verbose: verbose,
	}
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

// PrintStepResult prints the result of a single step execution
func (p *Printer) PrintStepResult(step StepResult) {
	num := colorDim().Sprintf("%2d.", step.Index)

	// Add wait time indicator if present
	waitPrefix := ""
	if step.WaitTime != "" {
		waitPrefix = colorDim().Sprintf("%s%s → ", IconWait, step.WaitTime)
	}

	// Add response time suffix if present
	ps := ""
	if step.RT != "" {
		ps = colorDim().Sprintf(" (%s)", step.RT)
	}

	var output string

	switch step.Status {
	case StatusSuccess:
		output = fmt.Sprintf("%s %s %s%s%s\n", num, colorSuccess().Sprintf(IconSuccess), waitPrefix, step.Name, ps)
	case StatusError:
		output = fmt.Sprintf("%s %s %s%s%s\n"+step.TestOutput+"\n", num, colorError().Sprintf(IconError), waitPrefix, step.Name, ps)
	case StatusWarning:
		output = fmt.Sprintf("%s %s %s%s%s\n", num, colorWarning().Sprintf(IconWarning), waitPrefix, step.Name, ps)
	case StatusSkipped:
		output = fmt.Sprintf("%s %s %s%s%s\n", num, colorWarning().Sprintf(IconSkip), waitPrefix, colorDim().Sprintf("%s", step.Name), ps)
	}

	fmt.Print(output)

	if step.EchoOutput != "" {
		fmt.Print(step.EchoOutput)
	}
}

// PrintStepRepeatStart prints the start of a repeated step execution
func (p *Printer) PrintStepRepeatStart(stepIdx int, stepName string, repeatCount int) {
	num := colorDim().Sprintf("%2d.", stepIdx)
	fmt.Printf("%s %s (repeating %d times)\n", num, stepName, repeatCount)
}

// PrintStepRepeatResult prints the final result of a repeated step execution
func (p *Printer) PrintStepRepeatResult(stepIdx int, counter StepRepeatCounter, hasTest bool) {
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

		fmt.Printf("    %s %d/%d success (%.1f%%)\n",
			statusIcon,
			counter.SuccessCount,
			totalCount,
			successRate)
	} else {
		totalCount := counter.SuccessCount + counter.FailureCount
		fmt.Printf("    %s %d/%d completed (no test)\n",
			colorWarning().Sprintf(IconWarning),
			totalCount,
			totalCount)
	}
}

// PrintJobResult prints the result of a job execution
func (p *Printer) PrintJobResult(jobName string, status StatusType, duration float64) {
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
	fmt.Printf("%s%s %s\n",
		statusColor.Sprint(statusIcon),
		jobName,
		dt)
}

// PrintJobResults prints buffered job results
func (p *Printer) PrintJobResults(output string) {
	output = strings.TrimSpace(output)
	if output != "" {
		lines := strings.Split(output, "\n")
		for i, line := range lines {
			if strings.TrimSpace(line) != "" {
				if i == 0 {
					fmt.Printf("  ⎿ %s\n", line)
				} else {
					fmt.Printf("    %s\n", line)
				}
			}
		}
	}
	fmt.Println()
}

// PrintWorkflowSummary prints the workflow execution summary
func (p *Printer) PrintWorkflowSummary(totalTime float64, successCount, totalJobs int) {
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

// PrintStepResult does nothing in silent mode
func (s *SilentPrinter) PrintStepResult(step StepResult) {}

// PrintStepRepeatStart does nothing in silent mode
func (s *SilentPrinter) PrintStepRepeatStart(stepIdx int, stepName string, repeatCount int) {}

// PrintStepRepeatResult does nothing in silent mode
func (s *SilentPrinter) PrintStepRepeatResult(stepIdx int, counter StepRepeatCounter, hasTest bool) {}

// PrintJobResult does nothing in silent mode
func (s *SilentPrinter) PrintJobResult(jobName string, status StatusType, duration float64) {}

// PrintJobResults does nothing in silent mode
func (s *SilentPrinter) PrintJobResults(output string) {}

// PrintWorkflowSummary does nothing in silent mode
func (s *SilentPrinter) PrintWorkflowSummary(totalTime float64, successCount, totalJobs int) {}

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
