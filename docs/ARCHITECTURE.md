# Architecture Documentation

Comprehensive architecture documentation for `bmad-automate`.

## System Overview

`bmad-automate` is a Go CLI tool that orchestrates Claude AI to automate development workflows. It spawns Claude as a subprocess, parses streaming JSON output, and displays formatted results.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              bmad-automate                                  │
│                                                                             │
│  ┌─────────────┐    ┌──────────────┐    ┌─────────────┐    ┌────────────┐  │
│  │  CLI Layer  │───▶│  Lifecycle   │───▶│   Workflow  │───▶│   Claude   │  │
│  │   (Cobra)   │    │  (Executor)  │    │   (Runner)  │    │ (Executor) │  │
│  └─────────────┘    └──────────────┘    └─────────────┘    └────────────┘  │
│         │                  │                   │                  │        │
│         ▼                  ▼                   ▼                  ▼        │
│  ┌─────────────┐    ┌──────────────┐    ┌─────────────┐    ┌────────────┐  │
│  │   Config    │    │    State     │    │   Status    │    │   Output   │  │
│  │   (Viper)   │    │  (Manager)   │    │   (Reader)  │    │  (Printer) │  │
│  └─────────────┘    └──────────────┘    └─────────────┘    └────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
                          ┌───────────────────────┐
                          │      Claude CLI       │
                          │  (External Process)   │
                          └───────────────────────┘
```

## Architecture Pattern

**Pattern:** Layered CLI Application with Dependency Injection

**Key Characteristics:**

- Single executable with subcommands
- Subprocess orchestration (wraps Claude CLI)
- Lifecycle-driven execution with state persistence for resume
- Event-driven streaming output
- Interface-based design for testability

## Layer Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     Entry Point Layer                           │
│                   cmd/bmad-automate/main.go                     │
│                       main() → cli.Execute()                    │
└─────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                        CLI Layer                                │
│                      internal/cli/                              │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ App struct (Dependency Injection Container)               │  │
│  │   - Config         *config.Config                         │  │
│  │   - Executor       claude.Executor                        │  │
│  │   - Printer        output.Printer                         │  │
│  │   - Runner         *workflow.Runner                       │  │
│  │   - Queue          *workflow.QueueRunner                  │  │
│  │   - StatusReader   *status.Reader                         │  │
│  │   - Lifecycle      *lifecycle.Executor                    │  │
│  │   - StateManager   *state.Manager                         │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  Commands: build, code-review, git-commit,                      │
│            run, queue, epic, raw                                │
└─────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Lifecycle Layer                              │
│                   internal/lifecycle/                           │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Executor                                               │    │
│  │    - Execute(ctx, storyKey) error                       │    │
│  │    - GetSteps(storyKey) ([]LifecycleStep, error)        │    │
│  │    - SetProgressCallback(ProgressCallback)              │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  Dependencies: WorkflowRunner, StatusReader, StatusWriter       │
└─────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Workflow Layer                              │
│                    internal/workflow/                           │
│                                                                 │
│  ┌─────────────────────┐     ┌─────────────────────────────┐    │
│  │  Runner             │     │  QueueRunner                │    │
│  │   - RunSingle()     │     │   - RunQueueWithStatus()    │    │
│  │   - RunRaw()        │     └─────────────────────────────┘    │
│  │   - RunFullCycle()  │                                        │
│  └─────────────────────┘                                        │
└─────────────────────────────────────────────────────────────────┘
                                  │
       ┌──────────────────────────┼──────────────────────────┐
       ▼                          ▼                          ▼
┌───────────────────┐  ┌───────────────────┐  ┌───────────────────┐
│  Claude Layer     │  │  Output Layer     │  │  Config Layer     │
│  internal/claude/ │  │  internal/output/ │  │  internal/config/ │
│                   │  │                   │  │                   │
│  - Executor       │  │  - Printer        │  │  - Loader         │
│  - Parser         │  │  - Styles         │  │  - Config         │
│  - Event          │  │                   │  │  - GetPrompt()    │
└───────────────────┘  └───────────────────┘  └───────────────────┘
         │
         ▼
┌───────────────────────────────────────────────────────────────┐
│                    External: Claude CLI                       │
│                                                               │
│   claude --dangerously-skip-permissions                       │
│          -p "<prompt>"                                        │
│          --output-format stream-json                          │
└───────────────────────────────────────────────────────────────┘

                    Support Layers
       ┌──────────────────────────┬──────────────────────────┐
       ▼                          ▼                          ▼
┌───────────────────┐  ┌───────────────────┐  ┌───────────────────┐
│  State Layer      │  │  Status Layer     │  │  Router Layer     │
│  internal/state/  │  │  internal/status/ │  │  internal/router/ │
│                   │  │                   │  │                   │
│  - Manager        │  │  - Reader         │  │  - GetWorkflow()  │
│  - Save()         │  │  - GetStoryStatus │  │  - GetLifecycle() │
│  - Load()         │  │  - ReadSpec       │  │  - LifecycleStep  │
│  - Clear()        │  │                   │  │                   │
└───────────────────┘  └───────────────────┘  └───────────────────┘
```

## Package Dependencies

```
cmd/bmad-automate/main.go
         │
         ▼
    internal/cli (Cobra commands)
         │
         ├──► internal/lifecycle (lifecycle orchestration)
         │         │
         │         ├──► internal/router (GetLifecycle for step sequences)
         │         │
         │         └──► internal/workflow (WorkflowRunner for execution)
         │
         ├──► internal/state (execution state persistence)
         │
         ├──► internal/workflow (single workflow orchestration)
         │         │
         │         ├──► internal/claude (Claude execution + JSON parsing)
         │         │
         │         ├──► internal/output (terminal formatting)
         │         │
         │         └──► internal/config (configuration)
         │
         ├──► internal/status (sprint status reading)
         │
         ├──► internal/router (workflow routing)
         │
         └──► internal/config (Viper configuration)
```

## Data Flow Diagram

### Single Workflow Execution

```
┌──────────────────────────────────────────────────────────────────────────┐
│  User: bmad-automate build PROJ-123                                      │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  1. CLI Layer                                                            │
│     - Cobra parses command and arguments                                 │
│     - Routes to build command handler                                    │
│     - Handler calls: runner.RunSingle(ctx, "build", "PROJ-123")          │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  2. Config Layer                                                         │
│     - config.GetPrompt("build", "PROJ-123")                              │
│     - Template: "/bmad-build-auto - Story: {{.StoryKey}}. ..."           │
│     - Expanded: "/bmad-build-auto - Story: PROJ-123. ..."                │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  3. Claude Layer                                                         │
│     - executor.ExecuteWithResult(ctx, prompt, handler)                   │
│     - Spawns: claude --dangerously-skip-permissions -p "..." ...         │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  4. Parser Layer                                                         │
│     - Reads JSON lines from stdout                                       │
│     - Converts StreamEvent → Event                                       │
│     - Emits events via channel                                           │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  5. Output Layer                                                         │
│     - handler(event) called for each event                               │
│     - printer.Text(msg) for text content                                 │
│     - printer.ToolUse(...) for tool invocations                          │
│     - printer.ToolResult(...) for tool results                           │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  6. Exit                                                                 │
│     - Claude subprocess completes                                        │
│     - Exit code propagated to CLI                                        │
│     - CLI returns ExitError or nil                                       │
└──────────────────────────────────────────────────────────────────────────┘
```

### Status-Based Routing (run command)

```
┌────────────────────────────────────────────────────────────────────────────┐
│  User: bmad-automate run PROJ-123                                          │
└────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  1. Status Reader                                                          │
│     - Read: _bmad-output/implementation-artifacts/sprint-status.yaml       │
│     - Get status for PROJ-123: "ready-for-dev"                             │
└────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  2. Router                                                                 │
│     - router.GetWorkflow("ready-for-dev") → "build"                        │
│                                                                            │
│     Routing Table:                                                         │
│       backlog       → build                                                │
│       ready-for-dev → build                                                │
│       in-progress   → build                                                │
│       review        → code-review                                          │
│       done          → ErrStoryComplete                                     │
└────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  3. Workflow Execution                                                     │
│     - runner.RunSingle(ctx, "build", "PROJ-123")                           │
│     - (same flow as single workflow execution)                             │
└────────────────────────────────────────────────────────────────────────────┘
```

### Queue Processing

```
┌────────────────────────────────────────────────────────────────────────────┐
│  User: bmad-automate queue PROJ-123 PROJ-124 PROJ-125                      │
└────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  QueueRunner.RunQueueWithStatus()                                          │
│                                                                            │
│  for each story:                                                           │
│    ┌────────────────────────────────────────────────────────────────────┐  │
│    │  1. Get status from sprint-status.yaml                             │  │
│    │  2. If "done" → skip                                               │  │
│    │  3. Route to workflow via router                                   │  │
│    │  4. Execute workflow                                               │  │
│    │  5. If exit code != 0 → stop queue                                 │  │
│    │  6. Record result (success/failure/skipped, duration)              │  │
│    └────────────────────────────────────────────────────────────────────┘  │
│                                                                            │
│  Print summary with all results                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

### Lifecycle Execution Flow

The lifecycle executor runs stories through their complete workflow sequence from current status to "done".

```
┌────────────────────────────────────────────────────────────────────────────┐
│  User: bmad-automate run PROJ-123                                          │
│  (Full lifecycle execution)                                                │
└────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  1. Get Current Status                                                     │
│     - statusReader.GetStoryStatus("PROJ-123") → "backlog"                  │
└────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  2. Get Lifecycle Steps                                                    │
│     - router.GetLifecycle("backlog") → 2 steps                             │
│                                                                            │
│     Steps:                                                                 │
│       1. build       → review                                              │
│       2. git-commit  → done                                                │
└────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  3. Execute Steps Loop                                                     │
│                                                                            │
│     for each step:                                                         │
│       ┌────────────────────────────────────────────────────────────────┐   │
│       │  a. Call progressCallback(stepIndex, totalSteps, workflow)     │   │
│       │  b. runner.RunSingle(ctx, workflow, storyKey)                  │   │
│       │  c. If exit code != 0 → return error (fail-fast)               │   │
│       │  d. After a build, read the story spec and stop unless it      │   │
│       │     recorded status "done"                                     │   │
│       │  e. statusWriter.UpdateStatus(storyKey, nextStatus)            │   │
│       └────────────────────────────────────────────────────────────────┘   │
│                                                                            │
│     Success: all steps completed                                           │
│     Failure: stops at first error                                          │
└────────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    ▼                               ▼
           ┌─────────────────┐             ┌─────────────────┐
           │     Success     │             │     Failure     │
           │                 │             │                 │
           │  Story is done  │             │  Save state     │
           │  Clear state    │             │  Report error   │
           └─────────────────┘             │  Exit with code │
                                           └─────────────────┘
```

**Why a build is verified against the story spec.** The Claude CLI exits 0 even
when `bmad-build-auto` stops on purpose. A blocked run ends through the skill's
HALT protocol, which records the reason and returns normally, so the exit code
cannot tell a finished story from a blocked one. After a build step the executor
reads the terminal status out of the story spec frontmatter that
`bmad-build-auto` wrote, and refuses to advance `sprint-status.yaml` unless that
status is `done`. Anything else fails the lifecycle and reports the blocking
condition along with the spec path. A missing spec is reported as a warning and
the exit code is used instead, because no better signal is available.

**Two status vocabularies.** They overlap but are not interchangeable, and each
file is written by exactly one side:

| File                   | Written by        | Values                                                                  |
| ---------------------- | ----------------- | ----------------------------------------------------------------------- |
| `sprint-status.yaml`   | `bmad-automate`   | `backlog`, `ready-for-dev`, `in-progress`, `review`, `done`             |
| story spec frontmatter | `bmad-build-auto` | `draft`, `ready-for-dev`, `in-progress`, `in-review`, `blocked`, `done` |

### State Persistence

The state package enables resume functionality when lifecycle execution fails.

```
┌────────────────────────────────────────────────────────────────────────────┐
│                         State File: .bmad-state.json                       │
│                                                                            │
│  Location: Working directory (hidden file)                                 │
│  Format: JSON                                                              │
│                                                                            │
│  {                                                                         │
│    "story_key": "PROJ-123",                                                │
│    "step_index": 0,           // 0-based, step that failed                 │
│    "total_steps": 2,          // total steps in lifecycle                  │
│    "start_status": "backlog"  // status when execution began               │
│  }                                                                         │
└────────────────────────────────────────────────────────────────────────────┘

                         Save/Load Flow

┌─────────────────┐                           ┌─────────────────┐
│  On Failure     │                           │  On Resume      │
│                 │                           │                 │
│  1. Save state  │                           │  1. Load state  │
│     to .json    │                           │     from .json  │
│                 │                           │                 │
│  2. Exit with   │                           │  2. Continue    │
│     error code  │                           │     from step   │
└─────────────────┘                           │                 │
                                              │  3. On success, │
                                              │     clear state │
                                              └─────────────────┘

                     Atomic Write Pattern

┌────────────────────────────────────────────────────────────────────────────┐
│  Manager.Save(state)                                                       │
│                                                                            │
│  1. Write to temporary file: .bmad-state.json.tmp                          │
│  2. Rename temp to final: .bmad-state.json                                 │
│                                                                            │
│  This temp + rename pattern ensures crash safety:                          │
│  - File is either fully written or not present                             │
│  - Never left in a corrupted partial state                                 │
└────────────────────────────────────────────────────────────────────────────┘
```

## Component Diagrams

### Claude Execution Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        DefaultExecutor                                  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ ExecuteWithResult(ctx, prompt, handler)
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  exec.CommandContext()                                                  │
│                                                                         │
│  cmd := exec.CommandContext(ctx, "claude",                              │
│      "--dangerously-skip-permissions",                                  │
│      "-p", prompt,                                                      │
│      "--output-format", "stream-json")                                  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
              ┌─────────────────────┴─────────────────────┐
              │                                           │
              ▼                                           ▼
┌─────────────────────────┐               ┌─────────────────────────────┐
│  stdout (JSON stream)   │               │  stderr (error output)      │
│                         │               │                             │
│  parser.Parse(stdout)   │               │  StderrHandler(line)        │
│          │              │               │  (logs to os.Stderr)        │
│          ▼              │               │                             │
│  chan Event             │               └─────────────────────────────┘
│          │              │
│          ▼              │
│  for event := range     │
│    handler(event)       │
└─────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  cmd.Wait()                                                             │
│  - Wait for Claude to complete                                          │
│  - Extract exit code from ExitError                                     │
│  - Return (exitCode, error)                                             │
└─────────────────────────────────────────────────────────────────────────┘
```

### Event Processing Pipeline

```
                      Claude CLI stdout
                            │
                            │ {"type":"system","subtype":"init",...}
                            │ {"type":"assistant","message":{"content":[...]}}
                            │ {"type":"user","tool_use_result":{...}}
                            │ {"type":"result",...}
                            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        Parser.Parse()                                   │
│                                                                         │
│  bufio.Scanner reads JSON lines                                         │
│  json.Unmarshal → StreamEvent                                           │
│  NewEventFromStream → Event                                             │
└─────────────────────────────────────────────────────────────────────────┘
                            │
                            │ Event{Type, Subtype, Text, ToolName, ...}
                            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      Event Handler                                      │
│                                                                         │
│  switch:                                                                │
│    event.IsText()       → printer.Text(event.Text)                      │
│    event.IsToolUse()    → printer.ToolUse(name, desc, cmd, path)        │
│    event.IsToolResult() → printer.ToolResult(stdout, stderr, limit)     │
└─────────────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     Styled Terminal Output                              │
│                                                                         │
│  ┌─ Bash ──────────────────────────────────────────────────────────┐    │
│  │  List files in directory                                        │    │
│  │  $ ls -la                                                       │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                         │
│  file1.txt                                                              │
│  file2.txt                                                              │
│  ...                                                                    │
└─────────────────────────────────────────────────────────────────────────┘
```

## Sequence Diagrams

### Complete Workflow Execution

```
User          CLI           Config        Runner        Executor       Parser        Printer
 │             │              │             │              │              │             │
 │─run PROJ-123│              │             │              │              │             │
 │             │              │             │              │              │             │
 │             │──GetPrompt()─▶             │              │              │             │
 │             │              │             │              │              │             │
 │             │◀─prompt──────│             │              │              │             │
 │             │              │             │              │              │             │
 │             │──RunSingle()──────────────▶│              │              │             │
 │             │              │             │              │              │             │
 │             │              │             │──SessionStart()─────────────────────────▶│
 │             │              │             │              │              │             │
 │             │              │             │──ExecuteWithResult()─▶│              │
 │             │              │             │              │              │             │
 │             │              │             │              │──Parse()────▶│             │
 │             │              │             │              │              │             │
 │             │              │             │              │◀─Event───────│             │
 │             │              │             │              │              │             │
 │             │              │             │◀─handler(event)─│              │             │
 │             │              │             │              │              │             │
 │             │              │             │──Text()───────────────────────────────────▶│
 │             │              │             │              │              │             │
 │             │              │             │◀─Event───────│              │             │
 │             │              │             │              │              │             │
 │             │              │             │──ToolUse()───────────────────────────────▶│
 │             │              │             │              │              │             │
 │             │              │             │◀─Event───────│              │              │
 │             │              │             │              │              │             │
 │             │              │             │──ToolResult()────────────────────────────▶│
 │             │              │             │              │              │             │
 │             │              │             │◀─exitCode────│              │             │
 │             │              │             │              │              │             │
 │             │              │             │──SessionEnd()────────────────────────────▶│
 │             │              │             │              │              │             │
 │             │◀─exitCode────│             │              │              │             │
 │             │              │             │              │              │             │
 │◀─Exit(code)─│              │             │              │              │             │
```

## Key Interfaces

### Executor Interface

```go
// Executor runs Claude CLI and returns streaming events.
type Executor interface {
    // Execute runs Claude with the given prompt and returns a channel of events.
    Execute(ctx context.Context, prompt string) (<-chan Event, error)

    // ExecuteWithResult runs Claude and waits for completion.
    ExecuteWithResult(ctx context.Context, prompt string, handler EventHandler) (int, error)
}
```

### Printer Interface

```go
// Printer handles terminal output formatting.
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

    // Queue output
    QueueHeader(count int, stories []string)
    QueueStoryStart(index, total int, storyKey string)
    QueueSummary(results []StoryResult, allKeys []string, totalDuration time.Duration)
}
```

### Parser Interface

```go
// Parser reads Claude's streaming JSON output.
type Parser interface {
    Parse(reader io.Reader) <-chan Event
}
```

### Lifecycle Interfaces

```go
// WorkflowRunner is the interface for executing individual workflows.
// Implemented by workflow.Runner.
type WorkflowRunner interface {
    RunSingle(ctx context.Context, workflowName, storyKey string) int
}

// StatusReader is the interface for looking up story status.
// Implemented by status.Reader.
type StatusReader interface {
    GetStoryStatus(storyKey string) (status.Status, error)
    ReadSpec(storyKey string) (*status.Spec, error)
}

// StatusWriter is the interface for persisting story status updates.
// Implemented by status.Reader (which handles both read and write).
type StatusWriter interface {
    UpdateStatus(storyKey string, newStatus status.Status) error
}

// ProgressCallback is invoked before each workflow step begins.
type ProgressCallback func(stepIndex, totalSteps int, workflow string)
```

### Lifecycle Executor

```go
// Executor orchestrates the complete story lifecycle.
type Executor struct {
    runner           WorkflowRunner
    statusReader     StatusReader
    statusWriter     StatusWriter
    progressCallback ProgressCallback
}

// Execute runs the complete story lifecycle from current status to done.
// Fail-fast: stops on first error.
func (e *Executor) Execute(ctx context.Context, storyKey string) error

// GetSteps returns the remaining lifecycle steps without executing.
// Useful for dry-run preview.
func (e *Executor) GetSteps(storyKey string) ([]LifecycleStep, error)

// SetProgressCallback configures progress reporting.
func (e *Executor) SetProgressCallback(cb ProgressCallback)
```

### State Manager

```go
// Manager handles state persistence operations.
type Manager struct {
    dir string  // Working directory for state file
}

// Save persists state atomically (temp file + rename).
func (m *Manager) Save(state State) error

// Load reads state from disk. Returns ErrNoState if not found.
func (m *Manager) Load() (State, error)

// Clear removes the state file. Idempotent.
func (m *Manager) Clear() error

// Exists returns true if a state file exists.
func (m *Manager) Exists() bool
```

## Testing Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Test Setup                                     │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  MockExecutor                                                   │    │
│  │    - Events []Event     (predetermined events to return)        │    │
│  │    - ExitCode int       (exit code to return)                   │    │
│  │    - RecordedPrompts    (capture prompts for assertions)        │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  Test Printer (NewPrinterWithWriter)                            │    │
│  │    - Writes to bytes.Buffer instead of os.Stdout                │    │
│  │    - Allows output capture and verification                     │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  Test Config                                                    │    │
│  │    - DefaultConfig() provides sensible defaults                 │    │
│  │    - Custom configs for specific test scenarios                 │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

## Error Handling

```
┌────────────────────────────────────────────────────────────────────────┐
│                        Error Flow                                      │
│                                                                        │
│  Command Handler                                                       │
│       │                                                                │
│       │  Error occurs                                                  │
│       ▼                                                                │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Return ExitError{Code: N}                                      │   │
│  │  - Wraps exit code for Cobra compatibility                      │   │
│  │  - Implements error interface                                   │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│       │                                                                │
│       ▼                                                                │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  RunWithConfig()                                                │   │
│  │  - Calls IsExitError() to extract code                          │   │
│  │  - Returns ExecuteResult{ExitCode, Err}                         │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│       │                                                                │
│       ▼                                                                │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Execute()                                                      │   │
│  │  - Calls os.Exit(code) for non-zero codes                       │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────────────────────┘
```

## Configuration Loading

```
┌────────────────────────────────────────────────────────────────────────┐
│                     Configuration Priority                             │
│                                                                        │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  1. Environment Variables (BMAD_*)                              │   │
│  │     - BMAD_CONFIG_PATH → custom config file                     │   │
│  │     - BMAD_CLAUDE_PATH → custom Claude binary                   │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                          │                                             │
│                          ▼                                             │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  2. Config File                                                 │   │
│  │     - $BMAD_CONFIG_PATH if set                                  │   │
│  │     - OR ./config/workflows.yaml                                │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                          │                                             │
│                          ▼                                             │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  3. Default Configuration                                       │   │
│  │     - Built-in defaults via DefaultConfig()                     │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                        │
│  Result: Merged Config struct                                          │
└────────────────────────────────────────────────────────────────────────┘
```

## Design Principles

1. **Dependency Injection** - All dependencies injected via App struct
2. **Interface Segregation** - Small, focused interfaces (Executor, Printer, Parser)
3. **Single Responsibility** - Each package has one clear purpose
4. **State Persistence** - Resume capability via atomic state file writes
5. **Event-Driven Processing** - Stream-based handling of Claude output
6. **Testability First** - Interfaces and mocks for isolated testing
7. **Graceful Degradation** - Queue continues processing, skips completed stories
8. **Fail-Fast Execution** - Stop immediately on error, save state for resume
