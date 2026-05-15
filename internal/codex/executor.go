package codex

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"bmad-automate/internal/claude"
)

// ExecutorConfig contains configuration for creating a [DefaultExecutor].
type ExecutorConfig struct {
	// BinaryPath is the path to the Codex CLI binary.
	// Defaults to "codex" if empty.
	BinaryPath string

	// Parser is the JSONL parser for Codex output.
	// Defaults to a new [Parser] if nil.
	Parser claude.Parser

	// StderrHandler is called for each line written to stderr by Codex.
	// If nil, stderr output is silently discarded.
	StderrHandler func(line string)
}

// DefaultExecutor implements [claude.Executor] by spawning Codex as a subprocess.
//
// It translates prompts from Claude format to Codex format by rewriting
// /bmad- prefixes to $bmad- prefixes, and parses Codex's JSONL output
// into [claude.Event] objects.
type DefaultExecutor struct {
	config ExecutorConfig
	parser claude.Parser
}

// NewExecutor creates a new Codex [DefaultExecutor] with the given configuration.
func NewExecutor(config ExecutorConfig) *DefaultExecutor {
	if config.BinaryPath == "" {
		config.BinaryPath = "codex"
	}

	parser := config.Parser
	if parser == nil {
		parser = NewParser()
	}

	return &DefaultExecutor{
		config: config,
		parser: parser,
	}
}

// rewritePrompt converts Claude slash command prefixes to Codex dollar-sign prefixes.
// e.g., "/bmad-code-review" becomes "$bmad-code-review"
func rewritePrompt(prompt string) string {
	return strings.Replace(prompt, "/bmad-", "$bmad-", -1)
}

// Execute runs Codex with the given prompt and returns a channel of [claude.Event] objects.
func (e *DefaultExecutor) Execute(ctx context.Context, prompt string) (<-chan claude.Event, error) {
	prompt = rewritePrompt(prompt)

	cmd := exec.CommandContext(ctx, e.config.BinaryPath,
		"exec",
		"--dangerously-bypass-approvals-and-sandbox",
		"--json",
		prompt,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start codex: %w", err)
	}

	go e.handleStderr(stderr)

	events := e.parser.Parse(stdout)

	go func() {
		_ = cmd.Wait()
	}()

	return events, nil
}

// ExecuteWithResult runs Codex with the given prompt and waits for completion.
func (e *DefaultExecutor) ExecuteWithResult(ctx context.Context, prompt string, handler claude.EventHandler) (int, error) {
	prompt = rewritePrompt(prompt)

	cmd := exec.CommandContext(ctx, e.config.BinaryPath,
		"exec",
		"--dangerously-bypass-approvals-and-sandbox",
		"--json",
		prompt,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 1, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 1, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("failed to start codex: %w", err)
	}

	go e.handleStderr(stderr)

	events := e.parser.Parse(stdout)
	for event := range events {
		if handler != nil {
			handler(event)
		}
	}

	err = cmd.Wait()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return 1, err
		}
	}

	return exitCode, nil
}

func (e *DefaultExecutor) handleStderr(stderr io.ReadCloser) {
	if e.config.StderrHandler == nil {
		_, _ = io.Copy(io.Discard, stderr)
		return
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		e.config.StderrHandler(scanner.Text())
	}
}
