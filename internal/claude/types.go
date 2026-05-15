// Package claude provides types and functionality for interacting with the Claude CLI.
//
// This package handles spawning Claude as a subprocess, parsing its streaming JSON
// output, and converting raw events into a convenient structured format.
//
// Key types:
//   - [Executor]: Interface for running Claude CLI commands
//   - [Parser]: Interface for parsing streaming JSON output
//   - [Event]: Parsed event with convenience methods for common checks
//
// For testing, use [MockExecutor] which implements [Executor] without spawning
// real processes.
package claude

import "time"

// StreamEvent represents a raw JSON event from Claude's streaming output.
//
// This is the low-level structure that maps directly to Claude's stream-json format.
// Most users should work with [Event] instead, which provides parsed fields and
// convenience methods. StreamEvent is primarily used internally by [Parser] and
// is available via [Event.Raw] for cases where access to the original JSON
// structure is needed.
type StreamEvent struct {
	Type          string          `json:"type"`
	Subtype       string          `json:"subtype,omitempty"`
	Message       *MessageContent `json:"message,omitempty"`
	ToolUseResult *ToolResult     `json:"tool_use_result,omitempty"`
	RateLimitInfo *RateLimitInfo  `json:"rate_limit_info,omitempty"`
}

// RateLimitInfo contains rate limit status from a rate_limit_event.
//
// This appears in Claude's stream-json output on every run. When the user
// has hit their quota, Status will be non-"allowed" and ResetsAt contains
// the Unix timestamp when the quota resets.
type RateLimitInfo struct {
	// Status is "allowed" when under quota, or another value when rate limited.
	Status string `json:"status"`

	// ResetsAt is the Unix timestamp when the rate limit resets.
	ResetsAt int64 `json:"resetsAt"`
}

// MessageContent represents the content of a message in Claude's streaming output.
//
// A message may contain multiple [ContentBlock] items, typically text output
// and/or tool invocations. This structure appears in assistant-type events
// within [StreamEvent.Message].
type MessageContent struct {
	Content []ContentBlock `json:"content,omitempty"`
}

// ContentBlock represents a single block of content within a [MessageContent].
//
// The Type field indicates the kind of content:
//   - "text": Contains text output in the Text field
//   - "tool_use": Contains a tool invocation with Name and Input fields
//
// For text blocks, only Type and Text are populated. For tool_use blocks,
// Type, Name, and Input are populated.
type ContentBlock struct {
	Type  string     `json:"type"`
	Text  string     `json:"text,omitempty"`
	Name  string     `json:"name,omitempty"`
	Input *ToolInput `json:"input,omitempty"`
}

// ToolInput represents the input parameters for a tool invocation.
//
// Different tools use different fields:
//   - Command: Used by bash/shell tools for the command to execute
//   - Description: Human-readable description of what the tool is doing
//   - FilePath: Used by file operations (read, write, edit) for the target path
//   - Content: Used by write operations for the content to write
//
// All fields are optional; which fields are populated depends on the specific tool.
type ToolInput struct {
	Command     string `json:"command,omitempty"`
	Description string `json:"description,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
	Content     string `json:"content,omitempty"`
}

// ToolResult represents the result of a tool execution.
//
// This structure appears in user-type events within [StreamEvent.ToolUseResult]
// and contains the output from tool execution:
//   - Stdout: Standard output from the tool (e.g., command output)
//   - Stderr: Standard error output from the tool
//   - Interrupted: True if the tool execution was interrupted (e.g., timeout or cancellation)
//
// Either Stdout or Stderr (or both) may be populated depending on the tool's output.
type ToolResult struct {
	Stdout      string `json:"stdout,omitempty"`
	Stderr      string `json:"stderr,omitempty"`
	Interrupted bool   `json:"interrupted,omitempty"`
}

// EventType represents the type of event received from Claude's streaming output.
//
// Events flow through the stream in a typical order: system (init), then alternating
// assistant and user events, and finally a result event when the session completes.
type EventType string

const (
	// EventTypeSystem indicates a system event, typically session initialization.
	// Check [Event.SessionStarted] to detect the init subtype.
	EventTypeSystem EventType = "system"

	// EventTypeAssistant indicates output from Claude, either text or tool invocations.
	// Use [Event.IsText] and [Event.IsToolUse] to distinguish between content types.
	EventTypeAssistant EventType = "assistant"

	// EventTypeUser indicates tool execution results returned to Claude.
	// Use [Event.IsToolResult] to check if this event contains tool output.
	EventTypeUser EventType = "user"

	// EventTypeResult indicates the session has completed.
	// Check [Event.SessionComplete] which will be true for result events.
	EventTypeResult EventType = "result"

	// EventTypeRateLimit indicates a rate limit status event.
	// Check [Event.IsRateLimited] to determine if the user has hit their quota.
	EventTypeRateLimit EventType = "rate_limit_event"
)

// SubtypeInit is the subtype value for system initialization events.
// When [Event.Type] is [EventTypeSystem] and [Event.Subtype] equals SubtypeInit,
// the Claude session has started.
const SubtypeInit = "init"

// Event is a parsed event from Claude's streaming output.
//
// This is the primary type that users interact with when processing Claude's output.
// It wraps the raw [StreamEvent] and extracts commonly needed fields into convenient
// top-level properties. Use the convenience methods [Event.IsText], [Event.IsToolUse],
// and [Event.IsToolResult] to quickly identify event types.
//
// Event is created by [NewEventFromStream] and emitted by [Parser.Parse].
type Event struct {
	// Raw provides access to the original [StreamEvent] for cases where
	// the parsed fields are insufficient.
	Raw *StreamEvent

	// Type is the parsed event type (system, assistant, user, or result).
	Type EventType

	// Subtype provides additional classification for certain event types.
	// For system events, this may be "init" (see [SubtypeInit]).
	Subtype string

	// Text contains the text content when Type is [EventTypeAssistant]
	// and the content block is of type "text". Empty otherwise.
	Text string

	// ToolName is the name of the tool being invoked when Type is
	// [EventTypeAssistant] and the content block is of type "tool_use".
	ToolName string

	// ToolDescription is a human-readable description of what the tool
	// is doing. Populated for tool_use events.
	ToolDescription string

	// ToolCommand is the command string for bash/shell tool invocations.
	ToolCommand string

	// ToolFilePath is the file path for file operation tools.
	ToolFilePath string

	// ToolStdout contains the standard output from a tool execution.
	// Populated when Type is [EventTypeUser] and the event contains tool results.
	ToolStdout string

	// ToolStderr contains the standard error output from a tool execution.
	ToolStderr string

	// ToolInterrupted indicates whether tool execution was interrupted.
	ToolInterrupted bool

	// SessionStarted is true for system init events, indicating the
	// Claude session has begun.
	SessionStarted bool

	// SessionComplete is true for result events, indicating the
	// Claude session has finished.
	SessionComplete bool

	// RateLimitStatus is the rate limit status (e.g., "allowed").
	// Populated when Type is [EventTypeRateLimit].
	RateLimitStatus string

	// RateLimitResetAt is when the rate limit resets.
	// Populated when Type is [EventTypeRateLimit] and ResetsAt > 0.
	RateLimitResetAt time.Time
}

// NewEventFromStream creates an [Event] from a raw [StreamEvent].
//
// This function parses the StreamEvent and extracts relevant fields into the
// Event's convenience properties based on the event type. It handles all event
// types (system, assistant, user, result) and populates the appropriate fields.
func NewEventFromStream(raw *StreamEvent) Event {
	e := Event{
		Raw:     raw,
		Type:    EventType(raw.Type),
		Subtype: raw.Subtype,
	}

	switch e.Type {
	case EventTypeSystem:
		if raw.Subtype == SubtypeInit {
			e.SessionStarted = true
		}

	case EventTypeAssistant:
		if raw.Message != nil {
			for _, block := range raw.Message.Content {
				switch block.Type {
				case "text":
					e.Text = block.Text
				case "tool_use":
					e.ToolName = block.Name
					if block.Input != nil {
						e.ToolDescription = block.Input.Description
						e.ToolCommand = block.Input.Command
						e.ToolFilePath = block.Input.FilePath
					}
				}
			}
		}

	case EventTypeUser:
		if raw.ToolUseResult != nil {
			e.ToolStdout = raw.ToolUseResult.Stdout
			e.ToolStderr = raw.ToolUseResult.Stderr
			e.ToolInterrupted = raw.ToolUseResult.Interrupted
		}

	case EventTypeRateLimit:
		if raw.RateLimitInfo != nil {
			e.RateLimitStatus = raw.RateLimitInfo.Status
			if raw.RateLimitInfo.ResetsAt > 0 {
				e.RateLimitResetAt = time.Unix(raw.RateLimitInfo.ResetsAt, 0)
			}
		}

	case EventTypeResult:
		e.SessionComplete = true
	}

	return e
}

// IsText returns true if this event contains text content from Claude.
//
// Use this method to filter for events where Claude is outputting text
// (as opposed to invoking tools). Returns true when Type is [EventTypeAssistant]
// and the Text field is non-empty.
func (e Event) IsText() bool {
	return e.Type == EventTypeAssistant && e.Text != ""
}

// IsToolUse returns true if this event represents a tool invocation by Claude.
//
// Use this method to detect when Claude is calling a tool. When true, the
// ToolName, ToolDescription, ToolCommand, and/or ToolFilePath fields will
// be populated depending on the specific tool being invoked.
func (e Event) IsToolUse() bool {
	return e.Type == EventTypeAssistant && e.ToolName != ""
}

// IsRateLimited returns true if this event indicates the user has hit their rate limit.
//
// Returns true when the event is a rate_limit_event with a status other than "allowed".
func (e Event) IsRateLimited() bool {
	return e.Type == EventTypeRateLimit && e.RateLimitStatus != "" && e.RateLimitStatus != "allowed"
}

// IsToolResult returns true if this event contains output from a tool execution.
//
// Use this method to detect tool execution results. When true, ToolStdout
// and/or ToolStderr will contain the tool's output. Check ToolInterrupted
// to determine if the tool was interrupted before completion.
func (e Event) IsToolResult() bool {
	return e.Type == EventTypeUser && (e.ToolStdout != "" || e.ToolStderr != "")
}
