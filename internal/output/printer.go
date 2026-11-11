package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// StepResult represents the result of a single workflow step execution.
//
// It captures the step name, execution duration, and success/failure status
// for display in cycle summaries.
type StepResult struct {
	// Name is the step identifier (e.g., "build", "git-commit").
	Name string
	// Duration is how long the step took to execute.
	Duration time.Duration
	// Success indicates whether the step completed successfully.
	Success bool
}

// StoryResult represents the result of processing a story in queue or epic operations.
//
// It tracks the outcome of each story in a batch operation, including whether
// it was skipped (already done), completed successfully, or failed.
type StoryResult struct {
	// Key is the story identifier (e.g., "7-1-define-schema").
	Key string
	// Success indicates whether the story completed all lifecycle steps.
	Success bool
	// Duration is how long the story processing took.
	Duration time.Duration
	// FailedAt contains the step name where processing failed, if any.
	FailedAt string
	// Skipped indicates the story was skipped because it was already done.
	Skipped bool
}

// Printer defines the interface for structured terminal output operations.
//
// The interface enables output capture in tests via [NewPrinterWithWriter],
// which accepts a custom io.Writer instead of writing to stdout.
//
// Methods are grouped by operation type: session lifecycle, step progress,
// tool usage display, content output, cycle summaries, and queue summaries.
type Printer interface {
	// SessionStart prints an indicator that a new execution session has begun.
	SessionStart()
	// SessionEnd prints completion status for the session with total duration.
	SessionEnd(duration time.Duration, success bool)

	// StepStart prints a numbered step header (e.g., "[1/2] build").
	StepStart(step, total int, name string)
	// StepEnd prints step completion status with duration.
	StepEnd(duration time.Duration, success bool)

	// ToolUse displays Claude tool invocation details including name,
	// description, command, and file path as applicable.
	ToolUse(name, description, command, filePath string)
	// ToolResult displays tool execution output, optionally truncating
	// stdout to the specified number of lines.
	ToolResult(stdout, stderr string, truncateLines int)

	// Text displays plain text content from Claude.
	Text(message string)
	// Divider prints a visual separator line between sections.
	Divider()

	// CycleHeader prints the header for a full lifecycle cycle operation.
	CycleHeader(storyKey string)
	// CycleSummary prints the completion summary showing all steps and durations.
	CycleSummary(storyKey string, steps []StepResult, totalDuration time.Duration)
	// CycleFailed prints failure information when a cycle fails at a step.
	CycleFailed(storyKey string, failedStep string, duration time.Duration)

	// QueueHeader prints the header for a batch queue operation.
	QueueHeader(count int, stories []string)
	// QueueStoryStart prints the header when starting a story in a queue.
	QueueStoryStart(index, total int, storyKey string)
	// QueueSummary prints the batch results summary showing completed,
	// skipped, failed, and remaining stories.
	QueueSummary(results []StoryResult, allKeys []string, totalDuration time.Duration)

	// CommandHeader prints the header before running a workflow command.
	CommandHeader(label, prompt string, truncateLength int)
	// CommandFooter prints the footer after a command completes with
	// duration, success status, and exit code.
	CommandFooter(duration time.Duration, success bool, exitCode int)
}

// DefaultPrinter implements [Printer] with lipgloss terminal styling.
//
// It is the production implementation used for CLI output. The styles
// are defined in styles.go and provide consistent color and formatting
// across all output operations.
type DefaultPrinter struct {
	out io.Writer
}

// NewPrinter creates a new [DefaultPrinter] that writes to stdout.
//
// This is the standard constructor for production CLI output.
func NewPrinter() *DefaultPrinter {
	return &DefaultPrinter{out: os.Stdout}
}

// NewPrinterWithWriter creates a new [DefaultPrinter] with a custom writer.
//
// This constructor enables output capture in tests by providing a bytes.Buffer
// or other io.Writer implementation instead of stdout.
func NewPrinterWithWriter(w io.Writer) *DefaultPrinter {
	return &DefaultPrinter{out: w}
}

func (p *DefaultPrinter) writeln(format string, args ...interface{}) {
	fmt.Fprintf(p.out, format+"\n", args...)
}

// SessionStart prints session start indicator.
func (p *DefaultPrinter) SessionStart() {
	p.writeln("%s Session started\n", iconInProgress)
}

// SessionEnd prints session end with status.
func (p *DefaultPrinter) SessionEnd(duration time.Duration, success bool) {
	p.writeln("%s Session complete", iconInProgress)
}

// StepStart prints step start header.
func (p *DefaultPrinter) StepStart(step, total int, name string) {
	header := fmt.Sprintf("[%d/%d] %s", step, total, name)
	p.writeln(stepHeaderStyle.Render(header))
}

// StepEnd prints step completion status.
func (p *DefaultPrinter) StepEnd(duration time.Duration, success bool) {
	// Step end is usually handled by CommandFooter
}

// ToolUse prints tool invocation details.
func (p *DefaultPrinter) ToolUse(name, description, command, filePath string) {
	p.writeln("%s Tool: %s", iconTool, toolNameStyle.Render(name))

	if description != "" {
		p.writeln("%s  %s", iconToolLine, description)
	}
	if command != "" {
		p.writeln("%s  $ %s", iconToolLine, command)
	}
	if filePath != "" {
		p.writeln("%s  File: %s", iconToolLine, filePath)
	}

	p.writeln(iconToolEnd)
}

// ToolResult prints tool execution results.
func (p *DefaultPrinter) ToolResult(stdout, stderr string, truncateLines int) {
	if stdout != "" {
		output := truncateOutput(stdout, truncateLines)
		// Indent each line
		indented := "   " + strings.ReplaceAll(output, "\n", "\n   ")
		p.writeln("%s\n", indented)
	}
	if stderr != "" {
		p.writeln("   %s\n", mutedStyle.Render("[stderr] "+stderr))
	}
}

// Text prints a text message from Claude.
func (p *DefaultPrinter) Text(message string) {
	if message != "" {
		p.writeln("Claude: %s\n", message)
	}
}

// Divider prints a visual divider.
func (p *DefaultPrinter) Divider() {
	p.writeln(dividerStyle.Render(strings.Repeat("═", 65)))
}

// CycleHeader prints the header for a full cycle run.
func (p *DefaultPrinter) CycleHeader(storyKey string) {
	p.writeln("")
	content := fmt.Sprintf("BMAD Full Cycle: %s\nSteps: build → git-commit", storyKey)
	p.writeln(headerStyle.Render(content))
	p.writeln("")
}

// CycleSummary prints the summary after a successful cycle.
func (p *DefaultPrinter) CycleSummary(storyKey string, steps []StepResult, totalDuration time.Duration) {
	var sb strings.Builder

	sb.WriteString(successStyle.Render(iconSuccess+" CYCLE COMPLETE") + "\n")
	sb.WriteString(fmt.Sprintf("Story: %s\n", storyKey))
	sb.WriteString(strings.Repeat("─", 50) + "\n")

	for i, step := range steps {
		sb.WriteString(fmt.Sprintf("[%d] %-15s %s\n", i+1, step.Name, step.Duration.Round(time.Millisecond)))
	}

	sb.WriteString(strings.Repeat("─", 50) + "\n")
	sb.WriteString(fmt.Sprintf("Total: %s", totalDuration.Round(time.Millisecond)))

	p.writeln(summaryStyle.Render(sb.String()))
}

// CycleFailed prints failure information when a cycle fails.
func (p *DefaultPrinter) CycleFailed(storyKey string, failedStep string, duration time.Duration) {
	var sb strings.Builder

	sb.WriteString(errorStyle.Render(iconError+" CYCLE FAILED") + "\n")
	sb.WriteString(fmt.Sprintf("Story: %s\n", storyKey))
	sb.WriteString(fmt.Sprintf("Failed at: %s\n", failedStep))
	sb.WriteString(fmt.Sprintf("Duration: %s", duration.Round(time.Millisecond)))

	p.writeln(summaryStyle.Render(sb.String()))
}

// QueueHeader prints the header for a queue run.
func (p *DefaultPrinter) QueueHeader(count int, stories []string) {
	p.writeln("")
	storiesList := truncateString(strings.Join(stories, ", "), 50)
	content := fmt.Sprintf("BMAD Queue: %d stories\nStories: %s", count, storiesList)
	p.writeln(headerStyle.Render(content))
	p.writeln("")
}

// QueueStoryStart prints the header for starting a story in a queue.
func (p *DefaultPrinter) QueueStoryStart(index, total int, storyKey string) {
	header := fmt.Sprintf("QUEUE [%d/%d]: %s", index, total, storyKey)
	p.writeln(queueHeaderStyle.Render(header))
}

// QueueSummary prints the summary after a queue completes or fails.
func (p *DefaultPrinter) QueueSummary(results []StoryResult, allKeys []string, totalDuration time.Duration) {
	completed := 0
	failed := 0
	skipped := 0
	for _, r := range results {
		if r.Skipped {
			skipped++
		} else if r.Success {
			completed++
		} else {
			failed++
		}
	}
	remaining := len(allKeys) - len(results)

	var sb strings.Builder

	if failed == 0 && remaining == 0 {
		sb.WriteString(successStyle.Render(iconSuccess+" QUEUE COMPLETE") + "\n")
	} else {
		sb.WriteString(errorStyle.Render(iconError+" QUEUE STOPPED") + "\n")
	}

	sb.WriteString(strings.Repeat("─", 50) + "\n")
	sb.WriteString(fmt.Sprintf("Completed: %d | Skipped: %d | Failed: %d | Remaining: %d\n", completed, skipped, failed, remaining))
	sb.WriteString(strings.Repeat("─", 50) + "\n")

	for _, r := range results {
		var status string
		var suffix string
		if r.Skipped {
			status = mutedStyle.Render("↷")
			suffix = "(done)"
		} else if r.Success {
			status = successStyle.Render(iconSuccess)
			suffix = ""
		} else {
			status = errorStyle.Render(iconError)
			suffix = ""
		}
		if suffix != "" {
			sb.WriteString(fmt.Sprintf("%s %-30s %s\n", status, r.Key, suffix))
		} else {
			sb.WriteString(fmt.Sprintf("%s %-30s %s\n", status, r.Key, r.Duration.Round(time.Second)))
		}
	}

	if remaining > 0 {
		for i := len(results); i < len(allKeys); i++ {
			sb.WriteString(fmt.Sprintf("%s %-30s (pending)\n", mutedStyle.Render(iconPending), allKeys[i]))
		}
	}

	sb.WriteString(strings.Repeat("─", 50) + "\n")
	sb.WriteString(fmt.Sprintf("Total: %s", totalDuration.Round(time.Second)))

	p.writeln(summaryStyle.Render(sb.String()))
}

// CommandHeader prints the header before running a command.
func (p *DefaultPrinter) CommandHeader(label, prompt string, truncateLength int) {
	p.Divider()
	p.writeln("  Command: %s", labelStyle.Render(label))
	p.writeln("  Prompt:  %s", truncateString(prompt, truncateLength))
	p.Divider()
	p.writeln("")
}

// CommandFooter prints the footer after a command completes.
func (p *DefaultPrinter) CommandFooter(duration time.Duration, success bool, exitCode int) {
	p.writeln("")
	p.Divider()
	if success {
		p.writeln("  %s | Duration: %s", successStyle.Render(iconSuccess+" SUCCESS"), duration.Round(time.Millisecond))
	} else {
		p.writeln("  %s | Duration: %s | Exit code: %d", errorStyle.Render(iconError+" FAILED"), duration.Round(time.Millisecond), exitCode)
	}
	p.Divider()
}

// truncateString truncates a string to maxLen, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// truncateOutput truncates output to maxLines, showing first and last portions.
func truncateOutput(output string, maxLines int) string {
	if maxLines <= 0 {
		return output
	}

	lines := strings.Split(output, "\n")
	if len(lines) <= maxLines {
		return output
	}

	half := maxLines / 2
	omitted := len(lines) - maxLines

	first := strings.Join(lines[:half], "\n")
	last := strings.Join(lines[len(lines)-half:], "\n")

	return fmt.Sprintf("%s\n  ... (%d lines omitted) ...\n%s", first, omitted, last)
}
