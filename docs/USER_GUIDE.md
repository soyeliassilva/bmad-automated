# User Guide

A practical guide to using `bmad-automate` for automating development workflows.

## Overview

`bmad-automate` is a CLI tool that orchestrates Claude AI to automate repetitive development tasks. It handles:

- Planning, implementing and reviewing a story in one unattended pass
- Running an extra review layer when you want one
- Committing and pushing changes
- Processing multiple stories in batch
- Managing entire epics

## Quick Start

### Prerequisites

1. **Go 1.21+** installed
2. **Claude CLI** installed and configured ([installation guide](https://github.com/anthropics/claude-code))
3. **uv** on `PATH`. The `bmad-build-auto` skill renders itself through `uv run _bmad/scripts/render_skill.py` and halts when uv is missing.
4. **just** command runner (optional but recommended)

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/bmad-automate.git
cd bmad-automate

# Build the binary
just build
# OR
go build -o bmad-automate ./cmd/bmad-automate

# (Optional) Install globally
go install ./cmd/bmad-automate
```

### Verify Installation

```bash
bmad-automate --help
```

## Basic Usage

### Working with Individual Stories

#### 1. Build the Story

Plan, implement and review the story in one pass:

```bash
bmad-automate build PROJ-123
```

This runs Claude with `/bmad-build-auto`, which will:

- Plan the story spec
- Implement every task
- Run the verification commands
- Run an adversarial review with auto-fix
- Commit its own reviewed diff, without pushing

The build needs a clean working tree to start, and it needs `uv` on `PATH`.

#### 2. Review the Code Again (optional)

The build already reviewed its own work. Run this when you want a second pass, or when a story is sitting at status `review`:

```bash
bmad-automate code-review PROJ-123
```

Claude reviews the code and automatically fixes issues.

#### 3. Push

Commit whatever is still uncommitted and push to remote:

```bash
bmad-automate git-commit PROJ-123
```

After a build there is usually only the push left to do.

### Status-Based Automation

Instead of manually running each step, let `bmad-automate` determine what to do:

```bash
bmad-automate run PROJ-123
```

The `run` command now executes the **complete lifecycle** from the story's current status all the way to `done`. It auto-updates the status after each successful step and stops at completion or on the first failure.

| Story Status    | Remaining Lifecycle                |
| --------------- | ---------------------------------- |
| `backlog`       | build -> git-commit -> done        |
| `ready-for-dev` | build -> git-commit -> done        |
| `in-progress`   | build -> git-commit -> done        |
| `review`        | code-review -> git-commit -> done  |
| `done`          | No action (story already complete) |

`build` sets the story to `review`, and `git-commit` sets it to `done`. On the review route, both `code-review` and `git-commit` set `done`.

### Full Lifecycle Execution

The `run` command processes stories through their entire remaining lifecycle automatically:

```bash
bmad-automate run PROJ-123
```

**Example output:**

```
[1/2] build
  ... workflow output ...

[2/2] git-commit
  ... workflow output ...

Story PROJ-123 completed successfully
```

**How it works:**

1. Reads the story's current status from `sprint-status.yaml`
2. Determines remaining lifecycle steps based on status
3. Executes each workflow in sequence
4. After a `build` step, checks the story spec to confirm the build really finished
5. Updates status in `sprint-status.yaml` after each successful step
6. Stops at `done` or on first failure

**Halted builds:**

The Claude CLI exits 0 even when `bmad-build-auto` deliberately halts, so the exit code on its own cannot tell a finished story from a blocked one. After every `build` step the tool reads the story spec that `bmad-build-auto` wrote under `_bmad-output/implementation-artifacts/` and checks the `status` in its frontmatter.

If that status is not `done`, the run stops and the story is left where it was in `sprint-status.yaml`:

```
Error: build halted for PROJ-123 with status "blocked": intent gap in acceptance criteria (see _bmad-output/implementation-artifacts/spec-PROJ-123-add-auth.md)
```

Read the spec, resolve the blocking condition, then run the story again. If no spec is found, or the only spec predates the build that just ran, the tool prints a warning and falls back to the exit code.

### Dry Run Mode

Preview what workflows will run without executing them:

```bash
bmad-automate run --dry-run PROJ-123
```

**Example output:**

```
Dry run for story PROJ-123:
  1. build -> review
  2. git-commit -> done
```

Dry run is available for all lifecycle commands:

```bash
bmad-automate run --dry-run PROJ-123           # Single story
bmad-automate queue --dry-run PROJ-123 PROJ-124  # Multiple stories
bmad-automate epic --dry-run 05                # All stories in epic
```

### Error Recovery

When a workflow fails, the tool saves execution state so you can resume from the point of failure:

**State file:** `.bmad-state.json`

```json
{
	"story_key": "PROJ-123",
	"step_index": 1,
	"total_steps": 2,
	"start_status": "backlog"
}
```

**Resuming after failure:**

```bash
# Workflow fails at step 1 (build)
bmad-automate run PROJ-123
# Error: workflow failed: build returned exit code 1

# Fix the issue, then re-run
bmad-automate run PROJ-123
# Continues from current status (no work is lost)
```

The state file is automatically cleared on successful completion.

### Batch Processing

Process multiple stories at once:

```bash
bmad-automate queue PROJ-123 PROJ-124 PROJ-125
```

The queue:

- Runs each story through its **full lifecycle** to completion
- Auto-updates status after each successful workflow step
- Skips completed stories
- Stops on first failure
- Shows a summary at the end

Use `--dry-run` to preview the full execution plan:

```bash
bmad-automate queue --dry-run PROJ-123 PROJ-124 PROJ-125
```

### Processing an Epic

Run all stories in an epic through their full lifecycle:

```bash
bmad-automate epic 05
```

This finds all stories matching the pattern `05-{N}-*` (e.g., `05-01-auth`, `05-02-dashboard`), sorts them by story number, and runs each through its complete lifecycle to completion.

Use `--dry-run` to preview:

```bash
bmad-automate epic --dry-run 05
```

### Ad-Hoc Prompts

Run any prompt directly:

```bash
bmad-automate raw "List all TODO comments in the codebase"
bmad-automate raw "What tests are missing coverage?"
```

## Configuration

### Config File Location

By default, configuration is loaded from `config/workflows.yaml`.

Override with environment variable:

```bash
export BMAD_CONFIG_PATH=/path/to/custom/config.yaml
bmad-automate run PROJ-123
```

### Customizing Workflows

Edit `config/workflows.yaml` to customize workflow prompts:

```yaml
workflows:
  build:
    prompt_template: |
      /bmad-build-auto - Story: {{.StoryKey}}.
      Plan the spec, implement every task, run the verification commands,
      and run the review with auto-fix, all in one unattended pass.
      Do not ask clarifying questions - use best judgment.

  code-review:
    prompt_template: |
      /bmad-code-review - Review story {{.StoryKey}}.
      This is an extra layer on top of the review the build already ran.
      Auto-fix all issues immediately.

  git-commit:
    prompt_template: |
      Commit any uncommitted changes for {{.StoryKey}}.
      Use conventional commit format.
      Push to current branch.
```

### Template Variables

| Variable        | Description                         |
| --------------- | ----------------------------------- |
| `{{.StoryKey}}` | The story key passed to the command |

### Output Settings

Control how much output is displayed:

```yaml
output:
  truncate_lines: 20 # Max lines for tool output
  truncate_length: 60 # Max chars for command headers
```

### Claude Settings

Customize Claude execution:

```yaml
claude:
  binary_path: claude # Path to Claude binary
  output_format: stream-json # Output format (don't change)
```

Or use environment variables:

```bash
export BMAD_CLAUDE_PATH=/usr/local/bin/claude
```

## Sprint Status File

### File Location

The tool reads story status from:

```
_bmad-output/implementation-artifacts/sprint-status.yaml
```

### File Format

```yaml
development_status:
  PROJ-123: ready-for-dev
  PROJ-124: in-progress
  PROJ-125: review
  PROJ-126: done
  05-01-auth: backlog
  05-02-dashboard: ready-for-dev
```

### Valid Status Values

| Status          | Meaning                                 |
| --------------- | --------------------------------------- |
| `backlog`       | Not started                             |
| `ready-for-dev` | Ready to be built                       |
| `in-progress`   | Currently being implemented             |
| `review`        | Built, waiting on extra review and push |
| `done`          | Complete                                |

Do not confuse these with the story spec. `sprint-status.yaml` uses `backlog`, `ready-for-dev`, `in-progress`, `review` and `done`. The spec that `bmad-build-auto` writes into `_bmad-output/implementation-artifacts/` carries its own frontmatter status: `draft`, `ready-for-dev`, `in-progress`, `in-review`, `blocked` or `done`. The tool only reads the spec status to check whether a build finished, and it never writes it.

## Workflow Patterns

### Pattern 1: Sequential Development

Run each step manually for full control:

```bash
bmad-automate build PROJ-123
# Read the story spec and test the implementation manually
bmad-automate code-review PROJ-123
# Verify fixes
bmad-automate git-commit PROJ-123
```

### Pattern 2: Status-Driven

Let the tool figure out what to do:

```bash
# Run whatever step is needed next
bmad-automate run PROJ-123

# Run again after updating status
bmad-automate run PROJ-123
```

### Pattern 3: Batch Sprint

Process an entire sprint's stories:

```bash
bmad-automate queue SPRINT-1 SPRINT-2 SPRINT-3 SPRINT-4 SPRINT-5
```

### Pattern 4: Epic Processing

Process all stories in an epic:

```bash
bmad-automate epic 05
```

### Pattern 5: Investigation

Use raw prompts for ad-hoc tasks:

```bash
# Understand the codebase
bmad-automate raw "Explain the authentication flow"

# Find issues
bmad-automate raw "What tests have the most failures?"

# Generate reports
bmad-automate raw "Create a summary of recent changes"
```

## Understanding Output

### Tool Invocations

When Claude uses tools, you'll see formatted output:

```
┌─ Bash ─────────────────────────────────────────────────────────
│  List project files
│  $ ls -la
└────────────────────────────────────────────────────────────────

total 48
drwxr-xr-x  12 user  staff   384 Jan  8 10:00 .
drwxr-xr-x   5 user  staff   160 Jan  8 09:00 ..
...
```

### Progress Indicators

| Symbol | Meaning     |
| ------ | ----------- |
| ●      | In progress |
| ✓      | Success     |
| ✗      | Failure     |
| ○      | Skipped     |

### Queue Summary

After processing multiple stories:

```
Summary:
  PROJ-123  ✓  1m 23s
  PROJ-124  ✓  2m 45s
  PROJ-125  ○  skipped (done)
  PROJ-126  ✗  failed at build (45s)
```

## Error Handling

### Exit Codes

| Code | Meaning                             |
| ---- | ----------------------------------- |
| 0    | Success                             |
| 1    | General error                       |
| N    | Claude's exit code (passed through) |

### Resume Capability

When a lifecycle execution fails, the tool persists state to `.bmad-state.json`. This enables resume from the point of failure:

1. **Failure occurs** - State is saved with current story and step index
2. **Fix the issue** - Address whatever caused the failure
3. **Re-run the command** - Execution continues from current status

The state file is automatically deleted after successful lifecycle completion.

### State File Location

```
.bmad-state.json   # In working directory (hidden file)
```

You can safely delete this file to force a fresh start from the story's current status.

### Common Issues

**Claude not found:**

```
Error: failed to start claude: exec: "claude": executable file not found in $PATH
```

Solution: Install Claude CLI or set `BMAD_CLAUDE_PATH`.

**Config not found:**

```
Error: error loading config: open config/workflows.yaml: no such file or directory
```

Solution: Create the config file or set `BMAD_CONFIG_PATH`.

**Story not in status file:**

```
Error: story PROJ-999 not found in sprint status
```

Solution: Add the story to `sprint-status.yaml`.

**Unknown status:**

```
Error: unknown status value: invalid-status
```

Solution: Use a valid status: `backlog`, `ready-for-dev`, `in-progress`, `review`, or `done`.

**Build halts right away:**

`bmad-build-auto` renders itself with `uv run _bmad/scripts/render_skill.py` and halts when `uv` is not on `PATH`. It also halts when the working tree is not clean.

Solution: install `uv`, commit or stash anything outstanding, then run the story again.

## Tips and Best Practices

### 1. Start Small

Begin with single story commands to understand the workflow:

```bash
bmad-automate build TEST-001
```

### 2. Review Output

Always review Claude's output before moving to the next step. The streaming output shows exactly what Claude is doing.

### 3. Use Descriptive Story Keys

Story keys appear in commits and prompts. Use meaningful identifiers:

- Good: `AUTH-001`, `FEAT-user-profile`, `BUG-login-fix`
- Bad: `x`, `test`, `123`

### 4. Customize Prompts for Your Project

Edit the prompt templates to match your project's conventions, coding standards, and requirements.

### 5. Keep Status File Updated

The status file is the source of truth for automation. Keep it current as stories progress.

### 6. Handle Failures Gracefully

When a queue stops on failure:

1. Review the error
2. Fix the issue manually or adjust the story
3. Update the status file
4. Re-run the queue (completed stories will be skipped)

### 7. Use Raw for Exploration

Before starting a story, use raw prompts to understand the codebase:

```bash
bmad-automate raw "What files would I need to change to add user authentication?"
```
