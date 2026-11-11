# CLI Reference

Complete command-line interface reference for `bmad-automate`.

## Synopsis

```
bmad-automate [command] [arguments] [flags]
```

## Description

BMAD Automation CLI orchestrates Claude AI to run development workflows including story implementation, code review, and git operations.

## Global Behavior

All commands:

- Load configuration from `config/workflows.yaml` (or `BMAD_CONFIG_PATH`)
- Execute Claude CLI with `--dangerously-skip-permissions` and `--output-format stream-json`
- Display styled terminal output with progress indicators
- Return appropriate exit codes (0 for success, non-zero for failure)

---

## Commands

### build

Plan, implement and review a story in one unattended pass.

**Usage:**

```bash
bmad-automate build <story-key>
```

**Arguments:**
| Argument | Required | Description |
|----------|----------|-------------|
| story-key | Yes | The story identifier (e.g., `PROJ-123`) |

**Example:**

```bash
bmad-automate build PROJ-123
```

**Behavior:**

1. Loads `build` workflow prompt from configuration
2. Expands `{{.StoryKey}}` template with provided story key
3. Executes Claude with `/bmad-build-auto`, which plans the story spec, implements every task, runs the verification commands and runs an adversarial review with auto-fix
4. Commits its own reviewed diff without pushing
5. Displays streaming output

**Requirements:**

- `uv` must be on `PATH`. `bmad-build-auto` renders itself through `uv run _bmad/scripts/render_skill.py` and halts when uv is missing.
- The working tree must be clean before the build starts.

`bmad-build-auto` records its outcome in a story spec under `_bmad-output/implementation-artifacts/`. The `run`, `queue` and `epic` commands read that spec to tell a finished story from a halted one. See [run](#run).

---

### code-review

Run an extra review layer on a story's changes.

**Usage:**

```bash
bmad-automate code-review <story-key>
```

**Arguments:**
| Argument | Required | Description |
|----------|----------|-------------|
| story-key | Yes | The story identifier |

**Example:**

```bash
bmad-automate code-review PROJ-123
```

**Behavior:**

1. Loads `code-review` workflow prompt
2. Executes Claude with `/bmad-code-review` to review code changes
3. Automatically applies fixes when issues are found

This runs on top of the review `build` already ran, so it is only needed when a story is at status `review` or when you want a second opinion.

---

### git-commit

Commit leftover changes for a story and push.

**Usage:**

```bash
bmad-automate git-commit <story-key>
```

**Arguments:**
| Argument | Required | Description |
|----------|----------|-------------|
| story-key | Yes | The story identifier |

**Example:**

```bash
bmad-automate git-commit PROJ-123
```

**Behavior:**

1. Loads `git-commit` workflow prompt
2. Executes Claude to commit whatever is still uncommitted, using conventional commit format
3. Pushes to the current branch

`build` commits its own reviewed diff without pushing, so after a build this step usually only has the push left to do.

---

### run

Execute the full lifecycle for a story from its current status to done.

**Usage:**

```bash
bmad-automate run [--dry-run] <story-key>
```

**Arguments:**
| Argument | Required | Description |
|----------|----------|-------------|
| story-key | Yes | The story identifier |

**Flags:**
| Flag | Description |
|------|-------------|
| `--dry-run` | Preview workflow sequence without execution |

**Example:**

```bash
# Run full lifecycle
bmad-automate run PROJ-123

# Preview what would run
bmad-automate run --dry-run PROJ-123
```

**Lifecycle Routing:**

The `run` command executes all remaining workflows to completion:

| Story Status    | Remaining Lifecycle                |
| --------------- | ---------------------------------- |
| `backlog`       | build -> git-commit -> done        |
| `ready-for-dev` | build -> git-commit -> done        |
| `in-progress`   | build -> git-commit -> done        |
| `review`        | code-review -> git-commit -> done  |
| `done`          | No action (story already complete) |

**Behavior:**

1. Reads story status from `_bmad-output/implementation-artifacts/sprint-status.yaml`
2. Determines remaining lifecycle steps based on status
3. Executes each workflow in sequence
4. After a `build` step, reads the story spec to confirm the build finished
5. Auto-updates status in `sprint-status.yaml` after each successful step
6. Stops at `done` or on first failure

**Build Verification:**

The Claude CLI exits 0 even when `bmad-build-auto` deliberately halts, so the exit code alone cannot tell a finished story from a blocked one. After the `build` step the executor globs `_bmad-output/implementation-artifacts/spec-<story-id>-*.md`, takes the most recently modified match, and reads the `status` field from its frontmatter. `<story-id>` is the story key up to and including its story number, so `7-1-define-schema` is looked up as `spec-7-1-*.md`.

If that status is anything other than `done`, the run stops with an error naming the status and the blocking condition, and `sprint-status.yaml` is left untouched:

```
Error: build halted for PROJ-123 with status "blocked": intent gap in acceptance criteria (see _bmad-output/implementation-artifacts/spec-PROJ-123-add-auth.md)
```

If no spec file matches, a warning is printed and the exit code is used instead. The same applies when the spec is older than the build that just ran, since a leftover spec from an earlier run says nothing about this one.

**Dry Run Output:**

```
Dry run for story PROJ-123:
  1. build -> review
  2. git-commit -> done
```

---

### queue

Run full lifecycle for multiple stories in batch.

**Usage:**

```bash
bmad-automate queue [--dry-run] <story-key> [story-key...]
```

**Arguments:**
| Argument | Required | Description |
|----------|----------|-------------|
| story-key | Yes | One or more story identifiers |

**Flags:**
| Flag | Description |
|------|-------------|
| `--dry-run` | Preview workflow sequence without execution |

**Example:**

```bash
# Run full lifecycle for each story
bmad-automate queue PROJ-123 PROJ-124 PROJ-125

# Preview what would run
bmad-automate queue --dry-run PROJ-123 PROJ-124 PROJ-125
```

**Behavior:**

1. Processes each story through its **full lifecycle** to completion
2. Auto-updates status after each successful workflow step
3. Skips stories with status `done`
4. Stops on first failure
5. Displays summary with timing for each story

**Output:**

```
Queue: 3 stories [PROJ-123, PROJ-124, PROJ-125]

[1/3] PROJ-123
  ... workflow output ...

[2/3] PROJ-124
  ... workflow output ...

Summary:
  PROJ-123  ✓  1m 23s
  PROJ-124  ✓  2m 45s
  PROJ-125  ○  skipped (done)
```

**Dry Run Output:**

```
Dry run for 3 stories:

Story PROJ-123:
  1. build -> review
  2. git-commit -> done

Story PROJ-124:
  (already complete)

Story PROJ-125:
  1. code-review -> done
  2. git-commit -> done

Total: 4 workflows across 2 stories (1 already complete)
```

---

### epic

Run full lifecycle for all stories in an epic.

**Usage:**

```bash
bmad-automate epic [--dry-run] <epic-id>
```

**Arguments:**
| Argument | Required | Description |
|----------|----------|-------------|
| epic-id | Yes | The epic identifier |

**Flags:**
| Flag | Description |
|------|-------------|
| `--dry-run` | Preview workflow sequence without execution |

**Example:**

```bash
# Run full lifecycle for all stories in epic
bmad-automate epic 05

# Preview what would run
bmad-automate epic --dry-run 05
```

**Story Discovery:**

Stories are discovered from `sprint-status.yaml` using the pattern:

```
{epic-id}-{story-number}-*
```

For epic `05`, this matches:

- `05-01-implement-auth`
- `05-02-add-dashboard`
- `05-03-fix-navigation`

Stories are sorted by story number and processed in order.

**Behavior:**

1. Finds all stories matching the epic pattern
2. Sorts by story number
3. Runs each story through its **full lifecycle** to completion
4. Auto-updates status after each successful workflow step
5. Stops on first failure

---

### raw

Execute an arbitrary prompt with Claude.

**Usage:**

```bash
bmad-automate raw <prompt>
```

**Arguments:**
| Argument | Required | Description |
|----------|----------|-------------|
| prompt | Yes | The prompt text (can be multiple words) |

**Example:**

```bash
bmad-automate raw "List all Go files in the project"
bmad-automate raw Explain the architecture of this codebase
```

**Behavior:**

1. Joins all arguments into a single prompt
2. Executes Claude directly with the prompt
3. Does not use any workflow templates

---

## Exit Codes

| Code | Meaning                                              |
| ---- | ---------------------------------------------------- |
| 0    | Success                                              |
| 1    | General error (config load failure, unknown command) |
| N    | Claude exit code (passed through from Claude CLI)    |

---

## Environment Variables

| Variable           | Description                | Default                   |
| ------------------ | -------------------------- | ------------------------- |
| `BMAD_CONFIG_PATH` | Path to configuration file | `./config/workflows.yaml` |
| `BMAD_CLAUDE_PATH` | Path to Claude binary      | `claude` (from PATH)      |

---

## Configuration File

The default configuration file is `config/workflows.yaml`:

```yaml
workflows:
  build:
    prompt_template: "/bmad-build-auto - Story: {{.StoryKey}}"

  code-review:
    prompt_template: "/bmad-code-review - Review story: {{.StoryKey}}"

  git-commit:
    prompt_template: "Commit changes for {{.StoryKey}}"

full_cycle:
  steps:
    - build
    - git-commit

claude:
  output_format: stream-json
  binary_path: claude

output:
  truncate_lines: 20 # Max lines to show for tool output
  truncate_length: 60 # Max chars for command header
```

### Template Variables

| Variable        | Description                         |
| --------------- | ----------------------------------- |
| `{{.StoryKey}}` | The story key passed to the command |

---

## Sprint Status File

The `run`, `queue`, and `epic` commands read story status from:

```
_bmad-output/implementation-artifacts/sprint-status.yaml
```

**Format:**

```yaml
development_status:
  PROJ-123: ready-for-dev
  PROJ-124: in-progress
  PROJ-125: done
```

**Valid Status Values:**

- `backlog` - Story not yet started
- `ready-for-dev` - Story ready for implementation
- `in-progress` - Story being implemented
- `review` - Story in code review
- `done` - Story complete

These are the only values written to `sprint-status.yaml`. The story spec that `bmad-build-auto` writes uses its own separate vocabulary (`draft`, `ready-for-dev`, `in-progress`, `in-review`, `blocked`, `done`) and is only read to verify a build.

---

## State File

The lifecycle executor persists execution state for error recovery.

**Location:**

```
.bmad-state.json   # In working directory (hidden file)
```

**Format:**

```json
{
	"story_key": "PROJ-123",
	"step_index": 1,
	"total_steps": 2,
	"start_status": "backlog"
}
```

**Fields:**
| Field | Description |
|-------|-------------|
| `story_key` | The story being processed |
| `step_index` | 0-based index of the current/failed step |
| `total_steps` | Total steps in the lifecycle sequence |
| `start_status` | The story's status when execution began |

**Lifecycle:**

1. **Saved on failure** - State is written when a workflow step fails
2. **Used on resume** - On re-run, execution continues from current status
3. **Cleared on success** - State file is deleted after successful lifecycle completion

**Notes:**

- The state file is optional - deleting it forces a fresh start from current status
- State is written atomically (temp file + rename) to prevent corruption
- Each story has its own state; queue/epic commands process stories sequentially

---

## Examples

### Basic Workflow

```bash
# Step-by-step workflow
bmad-automate build PROJ-123
bmad-automate git-commit PROJ-123

# Optional extra review layer before committing
bmad-automate code-review PROJ-123
```

### Status-Based Automation

```bash
# Let the tool determine the right workflow
bmad-automate run PROJ-123

# Process multiple stories
bmad-automate queue PROJ-123 PROJ-124 PROJ-125

# Process an entire epic
bmad-automate epic 05
```

### Ad-Hoc Tasks

```bash
# Run arbitrary prompts
bmad-automate raw "What is the test coverage?"
bmad-automate raw "Find all TODO comments"
```

### Custom Configuration

```bash
# Use custom config file
BMAD_CONFIG_PATH=/path/to/config.yaml bmad-automate run PROJ-123

# Use custom Claude binary
BMAD_CLAUDE_PATH=/usr/local/bin/claude bmad-automate build PROJ-123
```
