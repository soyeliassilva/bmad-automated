// Package codex provides an executor and parser for OpenAI's Codex CLI.
//
// This package implements the [claude.Executor] and [claude.Parser] interfaces,
// allowing the workflow runner to use Codex CLI as an alternative to Claude CLI
// for specific workflows (e.g., code review).
//
// Codex outputs JSONL via `codex exec --json`, which this package translates
// into [claude.Event] objects so the rest of the pipeline (workflow runner,
// output printer) works without changes.
package codex

// CodexEvent represents a raw JSONL line from `codex exec --json`.
type CodexEvent struct {
	Type     string      `json:"type"`
	ThreadID string      `json:"thread_id,omitempty"`
	Item     *CodexItem  `json:"item,omitempty"`
	Usage    *CodexUsage `json:"usage,omitempty"`
}

// CodexItem represents an item within a Codex event.
//
// The Type field indicates the kind of item:
//   - "command_execution": A shell command execution with Command, AggregatedOutput, ExitCode
//   - "agent_message": A text response with Text
type CodexItem struct {
	ID               string `json:"id"`
	Type             string `json:"type"`
	Command          string `json:"command,omitempty"`
	AggregatedOutput string `json:"aggregated_output,omitempty"`
	ExitCode         *int   `json:"exit_code,omitempty"`
	Status           string `json:"status,omitempty"`
	Text             string `json:"text,omitempty"`
}

// CodexUsage contains token usage information from a Codex turn.
type CodexUsage struct {
	InputTokens       int `json:"input_tokens"`
	CachedInputTokens int `json:"cached_input_tokens"`
	OutputTokens      int `json:"output_tokens"`
	ReasoningTokens   int `json:"reasoning_output_tokens"`
}
