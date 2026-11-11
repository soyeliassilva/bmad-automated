# Basic Workflow Recipes

Single-story commands for step-by-step development.

---

## Build a Story

Plan, implement and review a story in one unattended pass.

```bash
bmad-automate build AUTH-042
```

Claude writes the story spec, implements every task, runs the verification commands, then runs an adversarial review with auto-fix. It commits its own reviewed diff but never pushes, so it needs a clean working tree to start. Status moves to `review` on success.

If the build halts instead of finishing, it still exits 0 and records the reason in the story spec. `run`, `queue` and `epic` read that spec and stop rather than treating the story as done.

---

## Review Code Changes

Run an extra review layer on story changes with auto-fix.

```bash
bmad-automate code-review AUTH-042
```

Claude reviews for quality issues, security vulnerabilities, and missing tests. Automatically applies fixes. This sits on top of the review `build` already ran, so it is optional after a build.

---

## Commit and Push

Create a conventional commit and push to remote.

```bash
bmad-automate git-commit AUTH-042
```

Claude commits anything left uncommitted using conventional format (e.g., `feat(auth): add login endpoint`), then pushes to the current branch. After a build there is usually only the push left to do.

---

## Run Ad-hoc Prompts

Execute arbitrary prompts for investigation or exploration.

```bash
# Explore the codebase
bmad-automate raw "What files handle user authentication?"

# Find issues
bmad-automate raw "List all TODO comments in this project"

# Generate reports
bmad-automate raw "Summarize test coverage by package"
```

Useful for understanding code before starting work or investigating issues.
