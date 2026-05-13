# bmad-automate

A CLI tool for automating [BMAD-METHOD](https://github.com/bmad-code-org/BMAD-METHOD) development workflows with Claude AI.

## Overview

`bmad-automate` orchestrates Claude to run [BMAD-METHOD](https://github.com/bmad-code-org/BMAD-METHOD) development workflows including story creation, implementation, code review, and git operations. It's designed to automate repetitive development tasks by delegating them to Claude with predefined prompts.

## Features

- **Workflow Automation** - Run predefined workflows (create-story, dev-story, code-review, git-commit)
- **Status-Based Routing** - Automatically determines next workflow based on story status
- **Full Lifecycle Execution** - Run a story from current status to completion with a single command
- **Epic Processing** - Process all stories in an epic in order
- **Queue Processing** - Process multiple stories in batch
- **Deferred Questions** - Claude logs uncertainties to a file instead of blocking, so you can review and resolve them after automation completes
- **Deferred Work Resolution** - Interactively triage technical debt and follow-up items logged during code reviews
- **Dry Run Mode** - Preview workflows without executing them
- **Configurable Prompts** - Customize workflow prompts via YAML configuration
- **Streaming Output** - Real-time feedback from Claude's execution
- **Styled Terminal Output** - Clean, readable output with progress indicators

## Installation

### Prerequisites

- Go 1.21 or later
- [Claude CLI](https://github.com/anthropics/claude-code) installed and configured
- [just](https://github.com/casey/just) (optional, for running tasks)

### From Source

```bash
git clone https://github.com/yourusername/bmad-automate.git
cd bmad-automate
go install ./cmd/bmad-automate
```

Or using just:

```bash
just install
```

### Build Only

```bash
just build
# Binary will be created as ./bmad-automate
```

## Usage

### Single Workflow Commands

```bash
# Create a story definition
bmad-automate create-story <story-key> # eg 1-5

# Implement a story
bmad-automate dev-story <story-key>

# Run code review
bmad-automate code-review <story-key>

# Commit and push changes
bmad-automate git-commit <story-key>
```

### Full Lifecycle

Run a story from its current status to completion:

```bash
bmad-automate run <story-key>
```

This executes all remaining workflows based on the story's current status:

- `backlog` → create-story → dev-story → code-review → git-commit → done
- `ready-for-dev` → dev-story → code-review → git-commit → done
- `in-progress` → dev-story → code-review → git-commit → done
- `review` → code-review → git-commit → done
- `done` → skipped (story already complete)

Status is automatically updated in `sprint-status.yaml` after each successful workflow.

Preview what workflows would run without executing them:

```bash
bmad-automate run <story-key> --dry-run
```

### Epic Processing

Run the full lifecycle for all stories in an epic:

```bash
bmad-automate epic <epic-id>
```

This finds all stories matching the pattern `{epic-id}-{N}-*` (where N is numeric), sorts them by story number, and runs each to completion before moving to the next.

Example:

```bash
bmad-automate epic 6
# Runs 6-1-*, 6-2-*, 6-3-*, etc. each to completion in order
```

The epic command stops on the first failure. Done stories are skipped.

#### Dry Run

Preview what workflows would run without executing them:

```bash
bmad-automate epic 6 --dry-run
```

### Queue Processing

Run the full lifecycle for multiple stories in batch:

```bash
bmad-automate queue <story-key> [story-key...]
```

Each story is run to completion before moving to the next. The queue stops on the first failure. Done stories are skipped.

Example:

```bash
bmad-automate queue 6-5 6-6 6-7 6-8
```

Preview what workflows would run without executing them:

```bash
bmad-automate queue 6-5 6-6 6-7 --dry-run
```

### Resolving Deferred Questions

During automated runs, Claude logs uncertainties and assumptions to `_bmad-output/implementation-artifacts/deferred-questions.md` instead of blocking for input. After an epic completes, review and resolve those questions interactively:

```bash
bmad-automate resolve-questions <epic-id>
```

This launches an interactive Claude session that walks you through each question one at a time, asks for your answer, and applies any necessary code changes.

### Resolving Deferred Work

Code reviews log technical debt, edge cases, and follow-up items to `_bmad-output/implementation-artifacts/deferred-work.md`. Triage these items interactively:

```bash
# Review all deferred work
bmad-automate resolve-work

# Review only items from a specific epic
bmad-automate resolve-work <epic-id>
```

For each item, you choose to:
- **[R] Resolve** - Apply the fix now
- **[D] Dismiss** - Remove it, not needed
- **[S] Skip** - Keep it for later

### Raw Prompts

Run an arbitrary prompt:

```bash
bmad-automate raw "List all Go files in the project"
```

### Help

```bash
bmad-automate --help
bmad-automate <command> --help
```

## Configuration

### Config File

Create a `config/workflows.yaml` file to customize workflow prompts:

```yaml
workflows:
  create-story:
    prompt_template: "Your custom prompt for {{.StoryKey}}"

  dev-story:
    prompt_template: "Your dev prompt for {{.StoryKey}}"

  code-review:
    prompt_template: "Your review prompt for {{.StoryKey}}"

  git-commit:
    prompt_template: "Your commit prompt for {{.StoryKey}}"

full_cycle:
  steps:
    - create-story
    - dev-story
    - code-review
    - git-commit

claude:
  output_format: stream-json
  binary_path: claude

output:
  truncate_lines: 20
  truncate_length: 60
```

### Environment Variables

| Variable           | Description                | Default                   |
| ------------------ | -------------------------- | ------------------------- |
| `BMAD_CONFIG_PATH` | Path to custom config file | `./config/workflows.yaml` |
| `BMAD_CLAUDE_PATH` | Path to Claude binary      | `claude`                  |

### Sprint Status File

The `run`, `queue`, and `epic` commands read and update story status from:

```
_bmad-output/implementation-artifacts/sprint-status.yaml
```

Example format:

```yaml
development_status:
  6-1-setup-project: done
  6-2-add-feature: in-progress
  6-3-fix-bug: backlog
```

Valid status values:

| Status          | Description                       |
| --------------- | --------------------------------- |
| `backlog`       | Story not yet started             |
| `ready-for-dev` | Story ready for implementation    |
| `in-progress`   | Story currently being implemented |
| `review`        | Story in code review              |
| `done`          | Story complete                    |

## Development

### Prerequisites

- Go 1.21+
- [just](https://github.com/casey/just) command runner
- [golangci-lint](https://golangci-lint.run/) (for linting)

### Available Tasks

```bash
just              # Show all available tasks
just build        # Build the binary
just test         # Run all tests
just test-verbose # Run tests with verbose output
just test-coverage # Generate coverage report
just lint         # Run linter
just fmt          # Format code
just vet          # Run go vet
just check        # Run fmt, vet, and test
just clean        # Remove build artifacts
```

### Project Structure

```
bmad-automate/
├── cmd/bmad-automate/     # Application entry point
├── config/                # Default configuration
├── internal/
│   ├── cli/               # Cobra CLI commands
│   ├── claude/            # Claude client and JSON parser
│   ├── config/            # Configuration loading (Viper)
│   ├── lifecycle/         # Story lifecycle execution
│   ├── output/            # Terminal output formatting
│   ├── router/            # Status-based workflow routing
│   ├── state/             # State machine definitions
│   ├── status/            # Sprint status file reader/writer
│   └── workflow/          # Workflow orchestration
├── justfile               # Task runner configuration
└── README.md
```

### Testing

Run tests:

```bash
just test
```

Run tests with coverage:

```bash
just test-coverage
# Open coverage.html in your browser
```

Test a specific package:

```bash
just test-pkg ./internal/claude
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
