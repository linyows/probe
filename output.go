package probe

import (
	"fmt"
	"os"
	"strings"

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
)

// LogLevel defines different logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// OutputWriter defines the interface for different output implementations
type OutputWriter interface {
	// Workflow level output
	PrintWorkflowHeader(name, description string)
	PrintJobName(name string)

	// Step level output
	PrintStepResult(step StepResult)
	PrintStepRepeatStart(stepIdx int, stepName string, repeatCount int)
	PrintStepRepeatResult(stepIdx int, counter StepRepeatCounter, hasTest bool)

	// Job level output
	PrintJobResult(jobName string, status StatusType, duration float64)
	PrintJobOutput(output string)

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
)

// StepResult represents the result of a step execution
type StepResult struct {
	Index      int
	Name       string
	Status     StatusType
	RT         string
	TestOutput string
	EchoOutput string
	HasTest    bool
}

// Output implements OutputWriter for console output
type Output struct {
	verbose bool
}

// NewOutput creates a new console output writer
func NewOutput(verbose bool) *Output {
	return &Output{
		verbose: verbose,
	}
}

// PrintWorkflowHeader prints the workflow name and description
func (o *Output) PrintWorkflowHeader(name, description string) {
	if name != "" {
		bold := color.New(color.Bold)
		bold.Printf("%s\n", name)
		if description != "" {
			colorDim().Printf("%s\n", description)
		}
	}
}

// PrintJobName prints the job name
func (o *Output) PrintJobName(name string) {
	fmt.Printf("%s\n", name)
}

// PrintStepResult prints the result of a single step execution
func (o *Output) PrintStepResult(step StepResult) {
	num := colorDim().Sprintf("%2d.", step.Index)
	ps := ""
	if step.RT != "" {
		ps = colorDim().Sprintf(" (%s)", step.RT)
	}

	output := fmt.Sprintf("%s %%s %s%s", num, step.Name, ps)

	switch step.Status {
	case StatusSuccess:
		output = fmt.Sprintf(output+"\n", colorSuccess().Sprintf(IconSuccess))
	case StatusError:
		output = fmt.Sprintf(output+"\n"+step.TestOutput+"\n", colorError().Sprintf(IconError))
	case StatusWarning:
		output = fmt.Sprintf(output+"\n", colorWarning().Sprintf(IconWarning))
	}

	fmt.Print(output)

	if step.EchoOutput != "" {
		fmt.Print(step.EchoOutput)
	}
}

// PrintStepRepeatStart prints the start of a repeated step execution
func (o *Output) PrintStepRepeatStart(stepIdx int, stepName string, repeatCount int) {
	num := colorDim().Sprintf("%2d.", stepIdx)
	fmt.Printf("%s %s (repeating %d times)\n", num, stepName, repeatCount)
}

// PrintStepRepeatResult prints the final result of a repeated step execution
func (o *Output) PrintStepRepeatResult(stepIdx int, counter StepRepeatCounter, hasTest bool) {
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
func (o *Output) PrintJobResult(jobName string, status StatusType, duration float64) {
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

	fmt.Printf("%s%s (%s in %.2fs)\n",
		statusColor.Sprint(statusIcon),
		jobName,
		statusStr,
		duration)
}

// PrintJobOutput prints buffered job output
func (o *Output) PrintJobOutput(output string) {
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
func (o *Output) PrintWorkflowSummary(totalTime float64, successCount, totalJobs int) {
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
func (o *Output) PrintError(format string, args ...interface{}) {
	fmt.Printf("%s: %s\n", colorError().Sprintf("Error"), fmt.Sprintf(format, args...))
}

// PrintVerbose prints verbose output (only if verbose mode is enabled)
func (o *Output) PrintVerbose(format string, args ...interface{}) {
	if o.verbose {
		fmt.Printf(format, args...)
	}
}

// PrintSeparator prints a separator line for verbose output
func (o *Output) PrintSeparator() {
	if o.verbose {
		fmt.Println("- - -")
	}
}

// LogDebug prints debug messages (only in verbose mode)
func (o *Output) LogDebug(format string, args ...interface{}) {
	if o.verbose {
		fmt.Printf("[DEBUG] %s\n", fmt.Sprintf(format, args...))
	}
}

// LogInfo prints informational messages
func (o *Output) LogInfo(format string, args ...interface{}) {
	fmt.Printf("[INFO] %s\n", fmt.Sprintf(format, args...))
}

// LogWarn prints warning messages to stderr
func (o *Output) LogWarn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s\n", colorWarning().Sprintf("[WARN] %s", fmt.Sprintf(format, args...)))
}

// LogError prints error messages to stderr
func (o *Output) LogError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s\n", colorError().Sprintf("[ERROR] %s", fmt.Sprintf(format, args...)))
}
