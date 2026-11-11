package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPrinter(t *testing.T) {
	p := NewPrinter()
	assert.NotNil(t, p)
}

func TestNewPrinterWithWriter(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)
	assert.NotNil(t, p)
}

func TestDefaultPrinter_SessionStart(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.SessionStart()

	output := buf.String()
	assert.Contains(t, output, "Session started")
}

func TestDefaultPrinter_SessionEnd(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.SessionEnd(5*time.Second, true)

	output := buf.String()
	assert.Contains(t, output, "Session complete")
}

func TestDefaultPrinter_StepStart(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.StepStart(1, 2, "build")

	output := buf.String()
	assert.Contains(t, output, "[1/2]")
	assert.Contains(t, output, "build")
}

func TestDefaultPrinter_ToolUse(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.ToolUse("Bash", "List files", "ls -la", "")

	output := buf.String()
	assert.Contains(t, output, "Bash")
	assert.Contains(t, output, "List files")
	assert.Contains(t, output, "ls -la")
}

func TestDefaultPrinter_ToolUse_WithFilePath(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.ToolUse("Read", "", "", "/path/to/file.go")

	output := buf.String()
	assert.Contains(t, output, "Read")
	assert.Contains(t, output, "/path/to/file.go")
}

func TestDefaultPrinter_ToolResult(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.ToolResult("file1.go\nfile2.go", "", 20)

	output := buf.String()
	assert.Contains(t, output, "file1.go")
	assert.Contains(t, output, "file2.go")
}

func TestDefaultPrinter_ToolResult_WithStderr(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.ToolResult("", "error message", 20)

	output := buf.String()
	assert.Contains(t, output, "stderr")
	assert.Contains(t, output, "error message")
}

func TestDefaultPrinter_Text(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.Text("Hello from Claude!")

	output := buf.String()
	assert.Contains(t, output, "Claude:")
	assert.Contains(t, output, "Hello from Claude!")
}

func TestDefaultPrinter_Text_Empty(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.Text("")

	output := buf.String()
	assert.Empty(t, output)
}

func TestDefaultPrinter_CommandHeader(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.CommandHeader("build: test-123", "Long prompt here", 20)

	output := buf.String()
	assert.Contains(t, output, "build: test-123")
}

func TestDefaultPrinter_CommandFooter_Success(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.CommandFooter(5*time.Second, true, 0)

	output := buf.String()
	assert.Contains(t, output, "SUCCESS")
}

func TestDefaultPrinter_CommandFooter_Failure(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.CommandFooter(5*time.Second, false, 1)

	output := buf.String()
	assert.Contains(t, output, "FAILED")
	assert.Contains(t, output, "Exit code: 1")
}

func TestDefaultPrinter_CycleHeader(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.CycleHeader("test-story")

	output := buf.String()
	assert.Contains(t, output, "BMAD Full Cycle")
	assert.Contains(t, output, "test-story")
}

func TestDefaultPrinter_CycleSummary(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	steps := []StepResult{
		{Name: "build", Duration: 30 * time.Second, Success: true},
		{Name: "git-commit", Duration: 10 * time.Second, Success: true},
	}

	p.CycleSummary("test-story", steps, 40*time.Second)

	output := buf.String()
	assert.Contains(t, output, "CYCLE COMPLETE")
	assert.Contains(t, output, "test-story")
	assert.Contains(t, output, "build")
	assert.Contains(t, output, "git-commit")
}

func TestDefaultPrinter_CycleFailed(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.CycleFailed("test-story", "build", 15*time.Second)

	output := buf.String()
	assert.Contains(t, output, "CYCLE FAILED")
	assert.Contains(t, output, "test-story")
	assert.Contains(t, output, "build")
}

func TestDefaultPrinter_QueueHeader(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.QueueHeader(3, []string{"story-1", "story-2", "story-3"})

	output := buf.String()
	assert.Contains(t, output, "BMAD Queue")
	assert.Contains(t, output, "3 stories")
}

func TestDefaultPrinter_QueueStoryStart(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	p.QueueStoryStart(2, 5, "story-key")

	output := buf.String()
	assert.Contains(t, output, "QUEUE")
	assert.Contains(t, output, "[2/5]")
	assert.Contains(t, output, "story-key")
}

func TestDefaultPrinter_QueueSummary_Success(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	results := []StoryResult{
		{Key: "story-1", Success: true, Duration: 10 * time.Second},
		{Key: "story-2", Success: true, Duration: 20 * time.Second},
	}

	p.QueueSummary(results, []string{"story-1", "story-2"}, 30*time.Second)

	output := buf.String()
	assert.Contains(t, output, "QUEUE COMPLETE")
	assert.Contains(t, output, "Completed: 2")
}

func TestDefaultPrinter_QueueSummary_WithFailure(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinterWithWriter(&buf)

	results := []StoryResult{
		{Key: "story-1", Success: true, Duration: 10 * time.Second},
		{Key: "story-2", Success: false, Duration: 5 * time.Second, FailedAt: "build"},
	}

	p.QueueSummary(results, []string{"story-1", "story-2", "story-3"}, 15*time.Second)

	output := buf.String()
	assert.Contains(t, output, "QUEUE STOPPED")
	assert.Contains(t, output, "Failed: 1")
	assert.Contains(t, output, "Remaining: 1")
	assert.Contains(t, output, "(pending)")
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		assert.Equal(t, tt.expected, result)
	}
}

func TestTruncateOutput(t *testing.T) {
	// Create 30 lines
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = "line"
	}
	input := strings.Join(lines, "\n")

	result := truncateOutput(input, 10)

	assert.Contains(t, result, "lines omitted")
}

func TestTruncateOutput_NoTruncation(t *testing.T) {
	input := "line1\nline2\nline3"
	result := truncateOutput(input, 10)

	assert.Equal(t, input, result)
}

func TestTruncateOutput_ZeroMaxLines(t *testing.T) {
	input := "line1\nline2\nline3"
	result := truncateOutput(input, 0)

	assert.Equal(t, input, result)
}
