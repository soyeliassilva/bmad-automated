# Lifecycle Automation Recipes

Full lifecycle execution from current status to done.

---

## Run Full Lifecycle From Any Status

Execute all remaining workflows until the story is complete.

```bash
bmad-automate run AUTH-042
```

Reads current status, runs remaining workflows in sequence, auto-updates status after each step. Stops at `done` or on first failure.

**Example output:**

```
[1/2] build
  ... workflow output ...

[2/2] git-commit
  ... workflow output ...

Story AUTH-042 completed successfully
```

`build` covers planning, implementation, verification and review in one step, so a story that is not yet in `review` only ever has two steps left.

---

## Preview Workflow Sequence

See what would run without executing anything.

```bash
bmad-automate run --dry-run AUTH-042
```

**Example output:**

```
Dry run for story AUTH-042:
  1. build -> review
  2. git-commit -> done
```

A story already at `review` shows `code-review -> done` in place of `build`.

Useful before committing to a long-running lifecycle.

---

## Resume After Failure

When a workflow fails, the tool saves state to `.bmad-state.json`. Re-run to continue from current status.

```bash
# First run fails at build
bmad-automate run AUTH-042
# Error: workflow failed: build returned exit code 1

# Fix the issue, then re-run
bmad-automate run AUTH-042
# Continues from the same status, since a failed step never advances it
```

A build can also stop without failing. `bmad-build-auto` exits 0 when it halts deliberately, so the tool reads the story spec it wrote and refuses to advance the story when the frontmatter status is not `done`:

```
Error: build halted for AUTH-042 with status "blocked": acceptance criteria contradict the architecture (see _bmad-output/implementation-artifacts/spec-AUTH-042-add-login.md)
```

Resolve the blocking condition named in the spec, then re-run.

State file is cleared automatically on successful completion.

---

## Force Fresh Start

Delete the state file to restart from the story's current status.

```bash
rm .bmad-state.json
bmad-automate run AUTH-042
```

Useful when you want to discard partial progress and start over.
