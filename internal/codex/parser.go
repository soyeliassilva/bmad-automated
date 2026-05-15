package codex

import (
	"bufio"
	"encoding/json"
	"io"

	"bmad-automate/internal/claude"
)

// Parser implements [claude.Parser] for Codex's JSONL output format.
//
// It reads JSONL lines from `codex exec --json` and converts them to
// [claude.Event] objects so the workflow runner and output printer can
// handle Codex output without modification.
type Parser struct {
	BufferSize int
}

// NewParser creates a new Codex [Parser] with default settings.
func NewParser() *Parser {
	return &Parser{
		BufferSize: 10 * 1024 * 1024, // 10MB
	}
}

// Parse reads Codex JSONL from the reader and emits [claude.Event] objects.
//
// Event mapping:
//   - thread.started     → SessionStarted: true
//   - item.started       + command_execution → ToolUse event
//   - item.completed     + command_execution → ToolResult event
//   - item.completed     + agent_message     → Text event
//   - turn.completed     → SessionComplete: true
//
// Unknown event types are silently skipped for forward compatibility.
func (p *Parser) Parse(reader io.Reader) <-chan claude.Event {
	events := make(chan claude.Event)

	go func() {
		defer close(events)

		scanner := bufio.NewScanner(reader)

		bufSize := p.BufferSize
		if bufSize <= 0 {
			bufSize = 10 * 1024 * 1024
		}
		buf := make([]byte, 0, 1024*1024)
		scanner.Buffer(buf, bufSize)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var codexEvent CodexEvent
			if err := json.Unmarshal([]byte(line), &codexEvent); err != nil {
				continue
			}

			if event, ok := convertEvent(&codexEvent); ok {
				events <- event
			}
		}
	}()

	return events
}

// convertEvent maps a CodexEvent to a claude.Event.
// Returns false if the event type is not relevant (e.g., turn.started).
func convertEvent(ce *CodexEvent) (claude.Event, bool) {
	switch ce.Type {
	case "thread.started":
		return claude.Event{
			Type:           claude.EventTypeSystem,
			Subtype:        claude.SubtypeInit,
			SessionStarted: true,
		}, true

	case "item.started":
		if ce.Item != nil && ce.Item.Type == "command_execution" {
			return claude.Event{
				Type:        claude.EventTypeAssistant,
				ToolName:    "Bash",
				ToolCommand: ce.Item.Command,
			}, true
		}

	case "item.completed":
		if ce.Item == nil {
			return claude.Event{}, false
		}

		switch ce.Item.Type {
		case "command_execution":
			return claude.Event{
				Type:       claude.EventTypeUser,
				ToolStdout: ce.Item.AggregatedOutput,
			}, true

		case "agent_message":
			return claude.Event{
				Type: claude.EventTypeAssistant,
				Text: ce.Item.Text,
			}, true
		}

	case "turn.completed":
		return claude.Event{
			Type:            claude.EventTypeResult,
			SessionComplete: true,
		}, true
	}

	return claude.Event{}, false
}
