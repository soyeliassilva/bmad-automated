# Package Documentation

Complete API reference for all internal packages in `bmad-automate`.

## Package Overview

| Package                 | Location              | Purpose                                            |
| ----------------------- | --------------------- | -------------------------------------------------- |
| [cli](#cli)             | `internal/cli/`       | CLI commands, dependency injection, error handling |
| [claude](#claude)       | `internal/claude/`    | Claude CLI execution and JSON parsing              |
| [config](#config)       | `internal/config/`    | Configuration loading and template expansion       |
| [output](#output)       | `internal/output/`    | Terminal formatting and styling                    |
| [workflow](#workflow)   | `internal/workflow/`  | Workflow orchestration                             |
| [lifecycle](#lifecycle) | `internal/lifecycle/` | Story lifecycle orchestration                      |
| [state](#state)         | `internal/state/`     | Lifecycle state persistence for resume             |
| [status](#status)       | `internal/status/`    | Sprint status and story spec reading               |
| [router](#router)       | `internal/router/`    | Workflow routing based on status                   |

---

## cli

**Package:** `internal/cli`

Command-line interface implementation using Cobra framework.

### Types

#### App

Dependency injection container holding all application dependencies.

```go
type App struct {
    Config       *config.Config      // Configuration settings
    Executor     claude.Executor     // Claude CLI executor
    Printer      output.Printer      // Terminal output formatter
    Runner       *workflow.Runner    // Workflow orchestrator
    Queue        *workflow.QueueRunner  // Batch processor
    StatusReader *status.Reader      // Sprint status reader
}
```

#### ExecuteResult

Result of CLI execution for testability.

```go
type ExecuteResult struct {
    ExitCode int    // Exit code (0 = success)
    Err      error  // Error if any
}
```

#### ExitError

Custom error type wrapping exit codes for Cobra compatibility.

```go
type ExitError struct {
    Code int
}

func (e *ExitError) Error() string
```

### Functions

#### NewApp

Creates a new application with all dependencies wired up.

```go
func NewApp(cfg *config.Config) *App
```

**Parameters:**

- `cfg` - Configuration loaded from file/environment

**Returns:**

- Fully wired `*App` with Executor, Printer, Runner, Queue, and StatusReader

**Example:**

```go
cfg, _ := config.NewLoader().Load()
app := cli.NewApp(cfg)
```

#### NewRootCommand

Creates the root Cobra command with all subcommands registered.

```go
func NewRootCommand(app *App) *cobra.Command
```

**Parameters:**

- `app` - Application with dependencies

**Returns:**

- Root Cobra command

#### RunWithConfig

Testable core that creates app and executes with given config.

```go
func RunWithConfig(cfg *config.Config) ExecuteResult
```

**Parameters:**

- `cfg` - Configuration to use

**Returns:**

- `ExecuteResult` with exit code and error

#### Run

Loads config and runs CLI, returning result.

```go
func Run() ExecuteResult
```

**Returns:**

- `ExecuteResult` with exit code and error

#### Execute

Entry point called by main(). Handles os.Exit().

```go
func Execute()
```

#### NewExitError

Creates an exit error with the given code.

```go
func NewExitError(code int) *ExitError
```

#### IsExitError

Type assertion helper for exit errors.

```go
func IsExitError(err error) (int, bool)
```

**Returns:**

- Exit code and true if error is ExitError
- 0 and false otherwise

---

## claude

**Package:** `internal/claude`

Claude CLI interaction, subprocess execution, and JSON parsing.

### Types

#### EventType

Type of event received from Claude.

```go
type EventType string

const (
    EventTypeSystem    EventType = "system"
    EventTypeAssistant EventType = "assistant"
    EventTypeUser      EventType = "user"
    EventTypeResult    EventType = "result"
)
```

#### StreamEvent

Raw JSON event from Claude's streaming output.

```go
type StreamEvent struct {
    Type          string          `json:"type"`
    Subtype       string          `json:"subtype,omitempty"`
    Message       *MessageContent `json:"message,omitempty"`
    ToolUseResult *ToolResult     `json:"tool_use_result,omitempty"`
}
```

#### MessageContent

Content of a message from Claude.

```go
type MessageContent struct {
    Content []ContentBlock `json:"content,omitempty"`
}
```

#### ContentBlock

Single block of content (text or tool use).

```go
type ContentBlock struct {
    Type  string     `json:"type"`      // "text" or "tool_use"
    Text  string     `json:"text,omitempty"`
    Name  string     `json:"name,omitempty"`
    Input *ToolInput `json:"input,omitempty"`
}
```

#### ToolInput

Input parameters for a tool invocation.

```go
type ToolInput struct {
    Command     string `json:"command,omitempty"`
    Description string `json:"description,omitempty"`
    FilePath    string `json:"file_path,omitempty"`
    Content     string `json:"content,omitempty"`
}
```

#### ToolResult

Result of a tool execution.

```go
type ToolResult struct {
    Stdout      string `json:"stdout,omitempty"`
    Stderr      string `json:"stderr,omitempty"`
    Interrupted bool   `json:"interrupted,omitempty"`
}
```

#### Event

Parsed event with convenience methods.

```go
type Event struct {
    Raw *StreamEvent

    // Parsed fields
    Type    EventType
    Subtype string

    // Text content
    Text string

    // Tool use
    ToolName        string
    ToolDescription string
    ToolCommand     string
    ToolFilePath    string

    // Tool result
    ToolStdout      string
    ToolStderr      string
    ToolInterrupted bool

    // Session state
    SessionStarted  bool
    SessionComplete bool
}
```

**Methods:**

```go
// IsText returns true if event contains text content
func (e Event) IsText() bool

// IsToolUse returns true if event is a tool invocation
func (e Event) IsToolUse() bool

// IsToolResult returns true if event contains tool result
func (e Event) IsToolResult() bool
```

#### Executor

Interface for running Claude CLI.

```go
type Executor interface {
    // Execute runs Claude and returns event channel (fire-and-forget)
    Execute(ctx context.Context, prompt string) (<-chan Event, error)

    // ExecuteWithResult runs Claude and waits for completion
    ExecuteWithResult(ctx context.Context, prompt string, handler EventHandler) (int, error)
}

// EventHandler is called for each event
type EventHandler func(event Event)
```

#### ExecutorConfig

Configuration for the Claude executor.

```go
type ExecutorConfig struct {
    BinaryPath    string              // Path to claude binary (default: "claude")
    OutputFormat  string              // Output format (default: "stream-json")
    Parser        Parser              // JSON parser (default: DefaultParser)
    StderrHandler func(line string)   // Handler for stderr lines
}
```

#### DefaultExecutor

Real implementation using os/exec.

```go
type DefaultExecutor struct {
    config ExecutorConfig
    parser Parser
}
```

#### MockExecutor

Test implementation for unit tests.

```go
type MockExecutor struct {
    Events          []Event   // Events to return
    Error           error     // Error to return
    ExitCode        int       // Exit code to return
    RecordedPrompts []string  // Captured prompts for assertions
}
```

#### Parser

Interface for parsing JSON output.

```go
type Parser interface {
    Parse(reader io.Reader) <-chan Event
}
```

#### DefaultParser

Standard parser implementation.

```go
type DefaultParser struct {
    BufferSize int  // Scanner buffer size (default: 10MB)
}
```

### Functions

#### NewEventFromStream

Creates an Event from a raw StreamEvent.

```go
func NewEventFromStream(raw *StreamEvent) Event
```

#### NewExecutor

Creates a new DefaultExecutor.

```go
func NewExecutor(config ExecutorConfig) *DefaultExecutor
```

**Example:**

```go
executor := claude.NewExecutor(claude.ExecutorConfig{
    BinaryPath:   "claude",
    OutputFormat: "stream-json",
    StderrHandler: func(line string) {
        fmt.Fprintln(os.Stderr, line)
    },
})
```

#### NewParser

Creates a new DefaultParser.

```go
func NewParser() *DefaultParser
```

#### ParseSingle

Parses a single JSON line into an Event.

```go
func ParseSingle(line string) (Event, error)
```

---

## config

**Package:** `internal/config`

Configuration loading via Viper with Go template expansion.

### Types

#### Config

Root configuration structure.

```go
type Config struct {
    Workflows map[string]WorkflowConfig
    FullCycle FullCycleConfig
    Claude    ClaudeConfig
    Output    OutputConfig
}
```

#### WorkflowConfig

Configuration for a single workflow.

```go
type WorkflowConfig struct {
    PromptTemplate string  // Go template with {{.StoryKey}}
}
```

#### FullCycleConfig

Configuration for full cycle execution.

```go
type FullCycleConfig struct {
    Steps []string  // e.g., ["build", "git-commit"]
}
```

#### ClaudeConfig

Claude CLI settings.

```go
type ClaudeConfig struct {
    OutputFormat string  // "stream-json"
    BinaryPath   string  // "claude"
}
```

#### OutputConfig

Output formatting settings.

```go
type OutputConfig struct {
    TruncateLines  int  // Max lines for tool output (default: 20)
    TruncateLength int  // Max chars for headers (default: 60)
}
```

#### PromptData

Data passed to prompt templates.

```go
type PromptData struct {
    StoryKey string
}
```

#### Loader

Configuration loader using Viper.

```go
type Loader struct {
    v *viper.Viper
}
```

### Functions

#### NewLoader

Creates a new configuration loader.

```go
func NewLoader() *Loader
```

#### Load

Loads configuration from defaults, file, and environment.

```go
func (l *Loader) Load() (*Config, error)
```

**Returns:**

- Merged configuration
- Error if loading fails

#### LoadFromFile

Loads configuration from a specific file.

```go
func (l *Loader) LoadFromFile(path string) (*Config, error)
```

#### GetPrompt

Expands a workflow prompt template with data.

```go
func (c *Config) GetPrompt(workflowName, storyKey string) (string, error)
```

**Parameters:**

- `workflowName` - Name of workflow (e.g., "build")
- `storyKey` - Story key to substitute

**Returns:**

- Expanded prompt string
- Error if workflow not found or template fails

**Example:**

```go
prompt, err := cfg.GetPrompt("build", "PROJ-123")
// "/bmad-build-auto - Story: PROJ-123. ..."
```

#### GetFullCycleSteps

Returns the list of steps for full cycle execution.

```go
func (c *Config) GetFullCycleSteps() []string
```

#### DefaultConfig

Returns built-in default configuration.

```go
func DefaultConfig() *Config
```

#### MustLoad

Loads configuration or panics.

```go
func MustLoad() *Config
```

---

## output

**Package:** `internal/output`

Terminal output formatting using Lipgloss.

### Types

#### StepResult

Result of a single workflow step.

```go
type StepResult struct {
    Name     string
    Duration time.Duration
    Success  bool
}
```

#### StoryResult

Result of processing a story in queue.

```go
type StoryResult struct {
    Key      string
    Success  bool
    Duration time.Duration
    FailedAt string  // Step that failed (if any)
    Skipped  bool    // True if story was skipped (done status)
}
```

#### Printer

Interface for terminal output.

```go
type Printer interface {
    // Session lifecycle
    SessionStart()
    SessionEnd(duration time.Duration, success bool)

    // Step progress
    StepStart(step, total int, name string)
    StepEnd(duration time.Duration, success bool)

    // Tool usage
    ToolUse(name, description, command, filePath string)
    ToolResult(stdout, stderr string, truncateLines int)

    // Content
    Text(message string)
    Divider()

    // Full cycle
    CycleHeader(storyKey string)
    CycleSummary(storyKey string, steps []StepResult, totalDuration time.Duration)
    CycleFailed(storyKey string, failedStep string, duration time.Duration)

    // Queue
    QueueHeader(count int, stories []string)
    QueueStoryStart(index, total int, storyKey string)
    QueueSummary(results []StoryResult, allKeys []string, totalDuration time.Duration)

    // Command info
    CommandHeader(label, prompt string, truncateLength int)
    CommandFooter(duration time.Duration, success bool, exitCode int)
}
```

#### DefaultPrinter

Lipgloss-based printer implementation.

```go
type DefaultPrinter struct {
    out io.Writer
}
```

### Functions

#### NewPrinter

Creates a printer that writes to stdout.

```go
func NewPrinter() *DefaultPrinter
```

#### NewPrinterWithWriter

Creates a printer that writes to a custom writer (for testing).

```go
func NewPrinterWithWriter(w io.Writer) *DefaultPrinter
```

**Example:**

```go
var buf bytes.Buffer
printer := output.NewPrinterWithWriter(&buf)
printer.Text("Hello")
fmt.Println(buf.String())
```

### Styles (from styles.go)

Visual elements used by the printer:

**Colors:**

- Primary (blue) - Headers, labels
- Success (green) - Success indicators
- Error (red) - Error indicators
- Warning (orange) - Warnings
- Muted (gray) - Secondary text
- Highlight (purple) - Tool names

**Icons:**

- `CheckIcon` (✓) - Success
- `CrossIcon` (✗) - Failure
- `PendingIcon` (○) - Pending/skipped
- `ProgressIcon` (●) - In progress

---

## workflow

**Package:** `internal/workflow`

Workflow orchestration and batch processing.

### Types

#### Step

Represents a workflow step.

```go
type Step struct {
    Name   string
    Prompt string
}
```

#### StepResult

Result of executing a step.

```go
type StepResult struct {
    Name     string
    Duration time.Duration
    ExitCode int
    Success  bool
}
```

#### Runner

Executes workflows using Claude.

```go
type Runner struct {
    executor claude.Executor
    printer  output.Printer
    config   *config.Config
}
```

#### QueueRunner

Batch processor for multiple stories.

```go
type QueueRunner struct {
    runner *Runner
}
```

### Functions

#### NewRunner

Creates a new workflow runner.

```go
func NewRunner(executor claude.Executor, printer output.Printer, cfg *config.Config) *Runner
```

#### RunSingle

Executes a single workflow step.

```go
func (r *Runner) RunSingle(ctx context.Context, workflowName, storyKey string) int
```

**Parameters:**

- `ctx` - Context for cancellation
- `workflowName` - Workflow to run (e.g., "build")
- `storyKey` - Story key for template expansion

**Returns:**

- Exit code (0 = success)

#### RunRaw

Executes an arbitrary prompt.

```go
func (r *Runner) RunRaw(ctx context.Context, prompt string) int
```

#### RunFullCycle

Executes all steps in full cycle sequence.

```go
func (r *Runner) RunFullCycle(ctx context.Context, storyKey string) int
```

#### NewQueueRunner

Creates a new queue runner.

```go
func NewQueueRunner(runner *Runner) *QueueRunner
```

#### RunQueueWithStatus

Processes multiple stories using status-based routing.

```go
func (q *QueueRunner) RunQueueWithStatus(ctx context.Context, storyKeys []string, statusReader *status.Reader) int
```

**Parameters:**

- `ctx` - Context for cancellation
- `storyKeys` - List of story keys to process
- `statusReader` - Reader for sprint status file

**Returns:**

- Exit code (0 = all successful)

**Behavior:**

- Reads status for each story
- Routes to appropriate workflow
- Skips "done" stories
- Stops on first failure

---

## lifecycle

**Package:** `internal/lifecycle`

Story lifecycle orchestration from current status to done.

The lifecycle package provides an Executor that runs stories through their complete workflow sequence (build->commit) based on current status. Each step updates the story status automatically after successful completion.

### Types

#### Executor

Orchestrates the complete story lifecycle from current status to done.

```go
type Executor struct {
    runner           WorkflowRunner
    statusReader     StatusReader
    statusWriter     StatusWriter
    progressCallback ProgressCallback
}
```

Executor uses dependency injection for testability: WorkflowRunner executes workflows, StatusReader looks up current status, and StatusWriter persists status updates.

#### WorkflowRunner

Interface for executing individual workflows.

```go
type WorkflowRunner interface {
    RunSingle(ctx context.Context, workflowName, storyKey string) int
}
```

RunSingle executes a named workflow for a story and returns the exit code. An exit code of 0 indicates success; any non-zero value indicates failure. The `workflow.Runner` type implements this interface.

#### StatusReader

Interface for looking up story status.

```go
type StatusReader interface {
    GetStoryStatus(storyKey string) (status.Status, error)
    ReadSpec(storyKey string) (*status.Spec, error)
}
```

GetStoryStatus retrieves the current status for a story key. Returns an error if the story cannot be found or the status file is invalid. ReadSpec retrieves the outcome `bmad-build-auto` recorded in the story spec, which is the only reliable signal of whether a build finished.

#### StatusWriter

Interface for persisting story status updates.

```go
type StatusWriter interface {
    UpdateStatus(storyKey string, newStatus status.Status) error
}
```

UpdateStatus sets a new status for a story after successful workflow completion. Returns an error if the status file cannot be written.

#### ProgressCallback

Callback invoked before each workflow step begins execution.

```go
type ProgressCallback func(stepIndex, totalSteps int, workflow string)
```

**Parameters:**

- `stepIndex` - 1-based index of the current step
- `totalSteps` - Total number of steps in the lifecycle
- `workflow` - Name of the workflow about to execute

The callback is optional and can be set via SetProgressCallback.

### Functions

#### NewExecutor

Creates a new Executor with the required dependencies.

```go
func NewExecutor(runner WorkflowRunner, reader StatusReader, writer StatusWriter) *Executor
```

**Parameters:**

- `runner` - Executes workflows
- `reader` - Looks up story status
- `writer` - Persists status updates

**Returns:**

- Configured `*Executor` ready for use

#### SetProgressCallback

Configures an optional progress callback for workflow execution.

```go
func (e *Executor) SetProgressCallback(cb ProgressCallback)
```

**Parameters:**

- `cb` - Callback to invoke before each workflow step

#### Execute

Runs the complete story lifecycle from current status to done.

```go
func (e *Executor) Execute(ctx context.Context, storyKey string) error
```

**Parameters:**

- `ctx` - Context for cancellation
- `storyKey` - Story identifier to process

**Returns:**

- `nil` on success
- Error if status lookup, workflow execution, or status update fails
- Error if a build halted instead of completing
- `router.ErrStoryComplete` for stories already done

**Behavior:**

- Looks up story's current status
- Determines remaining workflow steps via `router.GetLifecycle`
- Runs each workflow in sequence
- Verifies a build against the story spec before advancing the status, because a halted build still exits 0
- Updates status after each successful workflow
- Stops on first error (fail-fast)

#### GetSteps

Returns the remaining lifecycle steps for a story without executing them.

```go
func (e *Executor) GetSteps(storyKey string) ([]router.LifecycleStep, error)
```

**Parameters:**

- `storyKey` - Story identifier to check

**Returns:**

- Slice of `router.LifecycleStep` showing planned execution path
- Error if status lookup fails
- `router.ErrStoryComplete` for stories already done

**Example:**

```go
executor := lifecycle.NewExecutor(runner, statusReader, statusWriter)
executor.SetProgressCallback(func(step, total int, workflow string) {
    fmt.Printf("Step %d/%d: %s\n", step, total, workflow)
})

err := executor.Execute(ctx, "PROJ-123")
if errors.Is(err, router.ErrStoryComplete) {
    fmt.Println("Story already complete")
}
```

---

## state

**Package:** `internal/state`

Lifecycle state persistence for resume functionality.

When a lifecycle execution fails (e.g., due to a Claude CLI error), the state is saved to disk so that execution can be resumed from the point of failure rather than starting over from the beginning. This is particularly valuable for long-running story lifecycles.

### Constants

```go
const StateFileName = ".bmad-state.json"
```

StateFileName is the name of the state file in the working directory. It is a hidden file (prefixed with ".") to avoid cluttering the directory. The file contains JSON-encoded State data.

```go
var ErrNoState = errors.New("no state file exists")
```

ErrNoState is a sentinel error returned by Manager.Load when no state file exists. Callers should treat this as a signal to start a fresh execution rather than as an error condition.

### Types

#### State

Represents the persisted lifecycle execution state.

```go
type State struct {
    StoryKey    string `json:"story_key"`     // Story being processed
    StepIndex   int    `json:"step_index"`    // 0-based index of next step
    TotalSteps  int    `json:"total_steps"`   // Total lifecycle steps
    StartStatus string `json:"start_status"`  // Status when execution began
}
```

**Fields:**

- `StoryKey` - Identifier of the story being processed
- `StepIndex` - 0-based index of the step that failed or is next to execute
- `TotalSteps` - Total number of steps in the lifecycle sequence (for progress display)
- `StartStatus` - Story's status when execution began (for debugging context)

#### Manager

Handles state persistence operations.

```go
type Manager struct {
    dir string  // Working directory for state file
}
```

The Manager uses a directory-based approach where the state file is stored in a configurable directory. This enables testability by allowing tests to use temporary directories.

### Functions

#### NewManager

Creates a new state manager for the given directory.

```go
func NewManager(dir string) *Manager
```

**Parameters:**

- `dir` - Working directory where state file will be stored (use "." for current directory)

**Returns:**

- Configured `*Manager` ready for use

#### Save

Persists the state to disk atomically.

```go
func (m *Manager) Save(state State) error
```

**Parameters:**

- `state` - State to persist

**Returns:**

- `nil` on success
- Error if marshaling or writing fails

**Behavior:**

The state is first written to a temporary file, then renamed to the final location. This temp file + rename pattern ensures crash safety: the state file is either fully written or not present, never corrupted.

#### Load

Reads the state from disk.

```go
func (m *Manager) Load() (State, error)
```

**Returns:**

- Loaded `State` and `nil` on success
- Empty `State` and `ErrNoState` if no state file exists
- Empty `State` and error for read or parse failures

#### Clear

Removes the state file if it exists.

```go
func (m *Manager) Clear() error
```

**Returns:**

- `nil` on success (including when file doesn't exist)
- Error if removal fails

This should be called after successful lifecycle completion to clean up. The method is idempotent.

#### Exists

Returns true if a state file exists.

```go
func (m *Manager) Exists() bool
```

Quick check to determine if there is saved state to resume from, without loading and parsing the full state data.

#### statePath

Returns the full path to the state file.

```go
func (m *Manager) statePath() string
```

Internal helper that joins the directory with StateFileName.

**Example:**

```go
manager := state.NewManager(".")

// Save state when lifecycle fails
st := state.State{
    StoryKey:    "PROJ-123",
    StepIndex:   2,
    TotalSteps:  4,
    StartStatus: "backlog",
}
if err := manager.Save(st); err != nil {
    log.Fatal(err)
}

// Load state to resume
loaded, err := manager.Load()
if errors.Is(err, state.ErrNoState) {
    fmt.Println("No state to resume, starting fresh")
} else if err != nil {
    log.Fatal(err)
} else {
    fmt.Printf("Resuming %s from step %d\n", loaded.StoryKey, loaded.StepIndex)
}

// Clear state after successful completion
manager.Clear()
```

---

## status

**Package:** `internal/status`

Sprint status file and story spec reading.

### Types

#### Status

Story development status.

```go
type Status string

const (
    StatusBacklog     Status = "backlog"
    StatusReadyForDev Status = "ready-for-dev"
    StatusInProgress  Status = "in-progress"
    StatusReview      Status = "review"
    StatusDone        Status = "done"
)
```

**Methods:**

```go
// IsValid returns true if status is a valid value
func (s Status) IsValid() bool
```

#### SprintStatus

Structure from sprint-status.yaml.

```go
type SprintStatus struct {
    DevelopmentStatus map[string]Status `yaml:"development_status"`
}
```

#### SpecStatus

Terminal status `bmad-build-auto` records in a story spec's frontmatter.

```go
type SpecStatus string

const (
    SpecDraft       SpecStatus = "draft"
    SpecReadyForDev SpecStatus = "ready-for-dev"
    SpecInProgress  SpecStatus = "in-progress"
    SpecInReview    SpecStatus = "in-review"
    SpecBlocked     SpecStatus = "blocked"
    SpecDone        SpecStatus = "done"
)
```

This is a different vocabulary from `Status`, which tracks sprint-status.yaml. The two overlap on ready-for-dev, in-progress and done, and diverge on draft, blocked and in-review (sprint-status.yaml calls the latter "review"). Only `SpecDone` means the run implemented, reviewed and committed the story.

#### Spec

Outcome `bmad-build-auto` recorded for a story.

```go
type Spec struct {
    Path     string      // Spec file the status was read from
    Status   SpecStatus  // Frontmatter status value
    Blocking string      // Blocking condition, empty unless the run halted
    ModTime  time.Time   // When the spec was last written
}
```

#### Reader

Reader for sprint status file and story specs.

```go
type Reader struct {
    basePath string
}
```

### Constants

```go
const DefaultStatusPath = "_bmad-output/implementation-artifacts/sprint-status.yaml"
const DefaultSpecDir = "_bmad-output/implementation-artifacts"
```

Sentinel error for spec lookups.

```go
var ErrSpecNotFound = errors.New("no story spec found")
```

### Functions

#### NewReader

Creates a reader with optional base path.

```go
func NewReader(basePath string) *Reader
```

**Parameters:**

- `basePath` - Base directory (empty string uses current directory)

#### Read

Reads the full sprint status file.

```go
func (r *Reader) Read() (*SprintStatus, error)
```

#### GetStoryStatus

Gets status for a specific story.

```go
func (r *Reader) GetStoryStatus(storyKey string) (Status, error)
```

#### ReadSpec

Gets the outcome `bmad-build-auto` recorded for a story.

```go
func (r *Reader) ReadSpec(storyKey string) (*Spec, error)
```

**Parameters:**

- `storyKey` - Story key whose spec should be read

**Returns:**

- `*Spec` with the frontmatter status and any blocking condition
- `ErrSpecNotFound` if no spec file matches the story key
- Error if the spec cannot be read or its frontmatter cannot be parsed

`bmad-build-auto` derives the spec filename from the clarified intent and leads the slug with the story's tracking identifier, so the file is located by globbing on that identifier rather than by an exact name. When several specs match, the most recently modified one wins. Exit code 0 alone does not prove a build succeeded, because a deliberate HALT also exits 0, so callers check this status before advancing a story.

#### GetEpicStories

Gets all stories in an epic, sorted by story number.

```go
func (r *Reader) GetEpicStories(epicID string) ([]string, error)
```

**Parameters:**

- `epicID` - Epic identifier (e.g., "05")

**Returns:**

- Sorted list of story keys matching pattern `{epicID}-{storyNum}-*`

**Example:**

```go
reader := status.NewReader("")
stories, err := reader.GetEpicStories("05")
// ["05-01-auth", "05-02-dashboard", "05-03-tests"]
```

---

## router

**Package:** `internal/router`

Workflow routing based on story status.

### Constants

Workflow names emitted by the router. These match keys in the workflows configuration.

```go
const (
    WorkflowBuild      = "build"
    WorkflowCodeReview = "code-review"
    WorkflowGitCommit  = "git-commit"
)
```

`WorkflowBuild` dispatches `bmad-build-auto`, which plans, implements and reviews a story in one unattended pass. `WorkflowCodeReview` dispatches `bmad-code-review`, an optional extra review layer on top of the one the build already ran. `WorkflowGitCommit` commits any leftover changes and pushes, since `bmad-build-auto` commits its own reviewed diff but never pushes.

### Variables

Sentinel errors for routing decisions.

```go
var (
    ErrStoryComplete = errors.New("story is complete, no workflow needed")
    ErrUnknownStatus = errors.New("unknown status value")
)
```

### Functions

#### GetWorkflow

Returns the workflow name for a given status.

```go
func GetWorkflow(s status.Status) (string, error)
```

**Routing Table:**

| Status          | Workflow         | Error              |
| --------------- | ---------------- | ------------------ |
| `backlog`       | `"build"`        | nil                |
| `ready-for-dev` | `"build"`        | nil                |
| `in-progress`   | `"build"`        | nil                |
| `review`        | `"code-review"`  | nil                |
| `done`          | `""`             | `ErrStoryComplete` |
| other           | `""`             | `ErrUnknownStatus` |

**Example:**

```go
workflow, err := router.GetWorkflow(status.StatusReadyForDev)
// workflow = "build", err = nil

workflow, err := router.GetWorkflow(status.StatusDone)
// workflow = "", err = ErrStoryComplete
```
