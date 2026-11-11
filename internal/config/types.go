// Package config provides configuration loading and management for bmad-automate.
//
// Configuration is loaded using Viper, supporting YAML config files and environment
// variable overrides. The package provides sensible defaults that work out of the
// box, with the ability to customize workflows, output formatting, and Claude CLI
// settings.
//
// Key types:
//   - [Config] is the root configuration container with all settings
//   - [Loader] handles Viper-based configuration loading
//   - [WorkflowConfig] defines a single workflow's prompt template
//   - [ClaudeConfig] contains Claude CLI binary settings
//
// Configuration priority (highest to lowest):
//  1. Environment variables (BMAD_ prefix)
//  2. Config file specified by BMAD_CONFIG_PATH
//  3. ./config/workflows.yaml
//  4. [DefaultConfig] defaults
package config

// deferredQuestionsInstruction is appended to workflow prompts so Claude logs
// uncertainties instead of blocking on questions during automated runs.
const deferredQuestionsInstruction = ` DEFERRED QUESTIONS: When you encounter uncertainty where documentation is incomplete or ambiguous and you must assume a decision, do NOT stop. Make your best judgment and proceed, but append the question to _bmad-output/implementation-artifacts/deferred-questions.md using this exact format per question:

## {{.StoryKey}} — [Brief title of the uncertainty]

- **Question:** What you were uncertain about
- **Decision made:** What you chose and why
- **Alternatives:** Other viable options you considered
- **Confidence:** Low / Medium / High
- **Files affected:** List of files impacted by this decision`

// Config represents the root configuration structure.
//
// This is the main configuration container loaded by [Loader] and used throughout
// the application. Use [DefaultConfig] to get sensible defaults.
type Config struct {
	// Workflows maps workflow names to their configurations.
	// Keys are workflow names (e.g., "build", "code-review").
	Workflows map[string]WorkflowConfig `mapstructure:"workflows"`

	// FullCycle defines the steps for full lifecycle execution.
	// Used by run, queue, and epic commands.
	FullCycle FullCycleConfig `mapstructure:"full_cycle"`

	// Claude contains Claude CLI binary configuration.
	Claude ClaudeConfig `mapstructure:"claude"`

	// Output contains terminal output formatting configuration.
	Output OutputConfig `mapstructure:"output"`

	// Codex contains Codex CLI configuration for alternative executor support.
	Codex CodexConfig `mapstructure:"codex"`

	// GitLocal when true modifies git-commit to only commit locally without pushing.
	// Set via --git-local flag on epic, run, and queue commands.
	GitLocal bool `mapstructure:"-"`
}

// WorkflowConfig represents a single workflow configuration.
//
// Each workflow has a prompt template that is expanded with story data
// using Go's text/template package.
type WorkflowConfig struct {
	// PromptTemplate is the Go template string for the workflow prompt.
	// Use {{.StoryKey}} to reference the story key.
	// Example: "Work on story: {{.StoryKey}}"
	PromptTemplate string `mapstructure:"prompt_template"`
}

// FullCycleConfig defines the steps for a full development cycle.
//
// This configuration is used by the run, queue, and epic commands
// to determine the sequence of workflows to execute.
type FullCycleConfig struct {
	// Steps is the ordered list of workflow names to execute.
	// Default: ["build", "git-commit"]
	Steps []string `mapstructure:"steps"`
}

// ClaudeConfig contains Claude CLI configuration.
//
// These settings control how the Claude CLI binary is invoked.
type ClaudeConfig struct {
	// OutputFormat is the output format passed to Claude CLI.
	// Should be "stream-json" for structured event parsing.
	OutputFormat string `mapstructure:"output_format"`

	// BinaryPath is the path to the Claude CLI binary.
	// Default: "claude" (assumes Claude is in PATH).
	// Can be overridden with BMAD_CLAUDE_PATH environment variable.
	BinaryPath string `mapstructure:"binary_path"`
}

// CodexConfig contains Codex CLI configuration.
//
// These settings control how the Codex CLI binary is invoked when used
// as an alternative executor for specific workflows.
type CodexConfig struct {
	// BinaryPath is the path to the Codex CLI binary.
	// Default: "codex" (assumes Codex is in PATH).
	BinaryPath string `mapstructure:"binary_path"`
}

// OutputConfig contains terminal output formatting configuration.
//
// These settings control how Claude's output is formatted in the terminal.
type OutputConfig struct {
	// TruncateLines is the maximum number of lines to display per event.
	// Additional lines are hidden with a "... (N more lines)" indicator.
	// Default: 20
	TruncateLines int `mapstructure:"truncate_lines"`

	// TruncateLength is the maximum length of each output line.
	// Longer lines are truncated with "..." suffix.
	// Default: 60
	TruncateLength int `mapstructure:"truncate_length"`
}

// DefaultConfig returns a new [Config] with sensible defaults.
//
// The defaults include standard workflow prompts for the build, code-review and
// git-commit workflows, as well as Claude CLI and output formatting settings.
// These defaults work out of the box without any configuration file.
func DefaultConfig() *Config {
	return &Config{
		Workflows: map[string]WorkflowConfig{
			"build": {
				PromptTemplate: "/bmad-build-auto - Story: {{.StoryKey}}. Plan the spec, implement every task, run the verification commands, and run the review with auto-fix, all in one unattended pass. Do not ask clarifying questions - use best judgment based on existing patterns." + deferredQuestionsInstruction,
			},
			"code-review": {
				PromptTemplate: "/bmad-code-review - Review story: {{.StoryKey}}. This is an extra layer on top of the review the build already ran. Whenever you are asked how to handle patch findings, always choose option 1, \"Apply every patch\", and never wait for user input. Resolve decision-needed findings with best judgment and record the reasoning instead of halting." + deferredQuestionsInstruction,
			},
			"git-commit": {
				PromptTemplate: "Commit any uncommitted changes for story {{.StoryKey}} with a descriptive commit message following conventional commits format. Part of the work may already be committed by an earlier workflow, in which case commit only what remains.{{if .GitLocal}} Do NOT push to remote. Only commit locally.{{else}} Then push to the current branch.{{end}} Do not ask questions.",
			},
		},
		FullCycle: FullCycleConfig{
			Steps: []string{"build", "git-commit"},
		},
		Claude: ClaudeConfig{
			OutputFormat: "stream-json",
			BinaryPath:   "claude",
		},
		Codex: CodexConfig{
			BinaryPath: "codex",
		},
		Output: OutputConfig{
			TruncateLines:  20,
			TruncateLength: 60,
		},
	}
}

// PromptData contains data for workflow template expansion.
//
// This struct is passed to Go's text/template when expanding workflow prompts.
// Fields are accessible in templates using {{.FieldName}} syntax.
type PromptData struct {
	// StoryKey is the identifier of the story being processed.
	// Access in templates with {{.StoryKey}}.
	StoryKey string

	// GitLocal when true indicates commits should stay local (no push).
	// Access in templates with {{.GitLocal}}.
	GitLocal bool
}
