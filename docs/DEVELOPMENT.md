# Development Guide

Complete guide for developing and extending `bmad-automate`.

## Development Setup

### Prerequisites

- **Go 1.21+** - [Install Go](https://go.dev/dl/)
- **just** - Task runner ([Install just](https://github.com/casey/just))
- **golangci-lint** - Linter ([Install golangci-lint](https://golangci-lint.run/))
- **Claude CLI** - For integration testing
- **uv** - Required at runtime: bmad-build-auto renders itself through `uv run _bmad/scripts/render_skill.py` and halts when `uv` is not on PATH ([Install uv](https://docs.astral.sh/uv/))

### Clone and Build

```bash
git clone https://github.com/yourusername/bmad-automate.git
cd bmad-automate

# Install dependencies
go mod download

# Build
just build
# OR
go build -o bmad-automate ./cmd/bmad-automate
```

### Verify Setup

```bash
# Run tests
just test

# Run linter
just lint

# Run all checks
just check
```

## Project Structure

```
bmad-automate/
├── cmd/
│   └── bmad-automate/
│       └── main.go              # Entry point
│
├── internal/
│   ├── cli/                     # CLI commands (Cobra)
│   │   ├── root.go              # Root command, dependency injection
│   │   ├── build.go             # build command
│   │   ├── code_review.go       # code-review command
│   │   ├── git_commit.go        # git-commit command
│   │   ├── run.go               # run command (status-based)
│   │   ├── queue.go             # queue command (batch)
│   │   ├── epic.go              # epic command
│   │   ├── raw.go               # raw command
│   │   ├── errors.go            # ExitError type
│   │   └── *_test.go            # Tests
│   │
│   ├── claude/                  # Claude CLI integration
│   │   ├── types.go             # Event types
│   │   ├── client.go            # Executor interface and impl
│   │   ├── parser.go            # JSON stream parser
│   │   └── *_test.go            # Tests
│   │
│   ├── config/                  # Configuration
│   │   ├── types.go             # Config types
│   │   ├── config.go            # Loader
│   │   └── config_test.go       # Tests
│   │
│   ├── output/                  # Terminal output
│   │   ├── printer.go           # Printer interface and impl
│   │   ├── styles.go            # Lipgloss styles
│   │   └── printer_test.go      # Tests
│   │
│   ├── workflow/                # Workflow orchestration
│   │   ├── workflow.go          # Runner
│   │   ├── steps.go             # Step types
│   │   ├── queue.go             # QueueRunner
│   │   └── workflow_test.go     # Tests
│   │
│   ├── lifecycle/               # Lifecycle execution (v1.1)
│   │   ├── executor.go          # Executor for full lifecycle
│   │   └── executor_test.go     # Tests
│   │
│   ├── state/                   # State persistence (v1.1)
│   │   ├── state.go             # Manager for save/load/clear
│   │   └── state_test.go        # Tests
│   │
│   ├── status/                  # Sprint status
│   │   ├── types.go             # Status types
│   │   ├── reader.go            # YAML reader
│   │   ├── spec.go              # Story spec frontmatter reader
│   │   └── *_test.go            # Tests
│   │
│   └── router/                  # Workflow routing
│       ├── router.go            # GetWorkflow function
│       ├── lifecycle.go         # GetLifecycle function (v1.1)
│       └── *_test.go            # Tests
│
├── config/
│   └── workflows.yaml           # Default configuration
│
├── docs/                        # Documentation
│
├── justfile                     # Task definitions
├── go.mod                       # Go module
├── go.sum                       # Dependencies
├── README.md                    # Project readme
├── CONTRIBUTING.md              # Contribution guide
└── CLAUDE.md                    # Claude Code instructions
```

## Available Tasks

```bash
just              # List all tasks
just build        # Build binary to ./bmad-automate
just test         # Run all tests
just test-verbose # Run tests with verbose output
just test-pkg ./internal/claude  # Test specific package
just test-coverage # Generate coverage.html
just lint         # Run golangci-lint
just fmt          # Format code
just vet          # Run go vet
just check        # Run fmt, vet, and test
just clean        # Remove build artifacts
just run --help   # Build and run with arguments
```

## Adding a New Command

### 1. Create the Command File

Create `internal/cli/my_command.go`:

```go
package cli

import (
    "github.com/spf13/cobra"
)

func newMyCommand(app *App) *cobra.Command {
    return &cobra.Command{
        Use:   "my-command <arg>",
        Short: "Brief description",
        Long:  `Detailed description of what the command does.`,
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            arg := args[0]

            // Use app dependencies
            exitCode := app.Runner.RunSingle(cmd.Context(), "my-workflow", arg)

            if exitCode != 0 {
                return NewExitError(exitCode)
            }
            return nil
        },
    }
}
```

### 2. Register the Command

Add to `internal/cli/root.go` in `NewRootCommand()`:

```go
rootCmd.AddCommand(
    // ... existing commands ...
    newMyCommand(app),
)
```

### 3. Add Workflow Configuration

Add to `config/workflows.yaml`:

```yaml
workflows:
  my-workflow:
    prompt_template: "Prompt for {{.StoryKey}}"
```

### 4. Write Tests

Create `internal/cli/my_command_test.go`:

```go
package cli

import (
    "testing"

    "github.com/stretchr/testify/assert"

    "bmad-automate/internal/claude"
    "bmad-automate/internal/config"
)

func TestMyCommand(t *testing.T) {
    cfg := config.DefaultConfig()
    cfg.Workflows["my-workflow"] = config.WorkflowConfig{
        PromptTemplate: "Test prompt for {{.StoryKey}}",
    }

    mock := &claude.MockExecutor{
        Events:   []claude.Event{},
        ExitCode: 0,
    }

    // Create test app with mock
    app := &App{
        Config:   cfg,
        Executor: mock,
        // ... other dependencies
    }

    rootCmd := NewRootCommand(app)
    rootCmd.SetArgs([]string{"my-command", "TEST-123"})

    err := rootCmd.Execute()

    assert.NoError(t, err)
    assert.Contains(t, mock.RecordedPrompts[0], "TEST-123")
}
```

## Testing

### Test Patterns

#### Table-Driven Tests

```go
func TestGetWorkflow(t *testing.T) {
    tests := []struct {
        name     string
        status   status.Status
        want     string
        wantErr  error
    }{
        {
            name:   "backlog routes to build",
            status: status.StatusBacklog,
            want:   "build",
        },
        {
            name:    "done returns error",
            status:  status.StatusDone,
            wantErr: ErrStoryComplete,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := GetWorkflow(tt.status)

            if tt.wantErr != nil {
                assert.ErrorIs(t, err, tt.wantErr)
                return
            }

            assert.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

#### Using MockExecutor

```go
func TestRunnerRunSingle(t *testing.T) {
    // Setup mock with predetermined responses
    mock := &claude.MockExecutor{
        Events: []claude.Event{
            {Type: claude.EventTypeSystem, SessionStarted: true},
            {Type: claude.EventTypeAssistant, Text: "Working..."},
            {Type: claude.EventTypeResult, SessionComplete: true},
        },
        ExitCode: 0,
    }

    // Capture output
    var buf bytes.Buffer
    printer := output.NewPrinterWithWriter(&buf)

    runner := workflow.NewRunner(mock, printer, config.DefaultConfig())

    exitCode := runner.RunSingle(context.Background(), "build", "TEST-123")

    assert.Equal(t, 0, exitCode)
    assert.Contains(t, mock.RecordedPrompts[0], "TEST-123")
}
```

#### Testing Output

```go
func TestPrinterText(t *testing.T) {
    var buf bytes.Buffer
    printer := output.NewPrinterWithWriter(&buf)

    printer.Text("Hello, World!")

    output := buf.String()
    assert.Contains(t, output, "Hello, World!")
}
```

### Running Tests

```bash
# All tests
just test

# Specific package
just test-pkg ./internal/claude

# With coverage
just test-coverage
open coverage.html

# Verbose output
just test-verbose
```

## Adding a New Package

### 1. Create Package Directory

```bash
mkdir internal/mypackage
```

### 2. Create Package Files

`internal/mypackage/mypackage.go`:

```go
// Package mypackage provides functionality for...
package mypackage

// MyType represents...
type MyType struct {
    Field string
}

// NewMyType creates a new MyType.
func NewMyType() *MyType {
    return &MyType{}
}

// DoSomething performs...
func (m *MyType) DoSomething() error {
    return nil
}
```

### 3. Create Test File

`internal/mypackage/mypackage_test.go`:

```go
package mypackage

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestMyType_DoSomething(t *testing.T) {
    m := NewMyType()
    err := m.DoSomething()
    assert.NoError(t, err)
}
```

### 4. Wire into Application

If the package needs application-level integration, add it to the `App` struct in `internal/cli/root.go`.

## Extending Claude Integration

### Adding New Event Types

1. Update `internal/claude/types.go`:

```go
// Add new fields to Event struct
type Event struct {
    // ... existing fields ...
    NewField string
}
```

2. Update `NewEventFromStream()` to populate the new field.

3. Add convenience method if needed:

```go
func (e Event) IsNewType() bool {
    return e.NewField != ""
}
```

### Custom Event Handler

```go
func customHandler(event claude.Event) {
    switch {
    case event.IsText():
        fmt.Println("Text:", event.Text)
    case event.IsToolUse():
        fmt.Printf("Tool: %s\n", event.ToolName)
    case event.IsToolResult():
        fmt.Println("Result:", event.ToolStdout)
    }
}

exitCode, _ := executor.ExecuteWithResult(ctx, prompt, customHandler)
```

## Code Style

### Go Conventions

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting (`just fmt`)
- Run `go vet` for static analysis (`just vet`)
- Use `golangci-lint` for comprehensive linting (`just lint`)

### Project Conventions

- **Package names**: Short, lowercase, no underscores
- **Interfaces**: Single-method interfaces end in `-er` (e.g., `Executor`, `Parser`)
- **Errors**: Use sentinel errors for expected conditions
- **Testing**: Table-driven tests with descriptive names

### Documentation

- All exported types and functions should have doc comments
- Package doc comments go in a `doc.go` file or the main file
- Use `// Comment` style, not `/* */`

## Debugging

### Enable Verbose Output

```bash
# Run with verbose test output
just test-verbose

# Build and run with debug info
go build -gcflags="all=-N -l" -o bmad-automate ./cmd/bmad-automate
```

### Inspect Claude Communication

The executor logs stderr if configured:

```go
executor := claude.NewExecutor(claude.ExecutorConfig{
    StderrHandler: func(line string) {
        fmt.Fprintln(os.Stderr, "[CLAUDE STDERR]", line)
    },
})
```

### Capture All Output

```go
var stdout, stderr bytes.Buffer
// Use custom writers to capture output for debugging
```

## Release Process

### 1. Update Version

Update version in relevant files if versioned.

### 2. Run Full Check

```bash
just check
just lint
```

### 3. Build for Distribution

```bash
# Build for current platform
just build

# Cross-compile (if needed)
GOOS=linux GOARCH=amd64 go build -o bmad-automate-linux-amd64 ./cmd/bmad-automate
GOOS=darwin GOARCH=arm64 go build -o bmad-automate-darwin-arm64 ./cmd/bmad-automate
```

### 4. Tag Release

```bash
git tag v1.0.0
git push origin v1.0.0
```

## Common Development Tasks

### Adding a Configuration Option

1. Add field to appropriate struct in `internal/config/types.go`
2. Add default value in `internal/config/config.go`
3. Add YAML key in `config/workflows.yaml`
4. Update documentation

### Adding Terminal Output Style

1. Define style in `internal/output/styles.go`
2. Add method to `Printer` interface
3. Implement in `DefaultPrinter`
4. Add test

### Adding a Status Type

1. Add constant to `internal/status/types.go`
2. Update `IsValid()` method
3. Add routing in `internal/router/router.go`
4. Add tests
