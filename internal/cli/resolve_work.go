package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func newResolveWorkCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "resolve-work [epic-id]",
		Short: "Interactively resolve deferred work items",
		Long: `Launch an interactive Claude session to resolve deferred work items logged
during automated code reviews.

Code reviews log technical debt, edge cases, missing validations, and other
follow-up items to _bmad-output/implementation-artifacts/deferred-work.md.

This command starts an interactive Claude session that:
  1. Reads the deferred work file
  2. Optionally filters by epic ID if provided
  3. Presents each item with its context
  4. Asks you whether to resolve it now, dismiss it, or defer it further
  5. For items you choose to resolve, applies the necessary code changes
  6. Removes resolved or dismissed items from the file

Examples:
  bmad-automate resolve-work      # Review all deferred work
  bmad-automate resolve-work 2    # Review only epic 2 deferred work`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var prompt string

			if len(args) == 1 {
				epicID := args[0]
				prompt = fmt.Sprintf(
					"Read the deferred work file at _bmad-output/implementation-artifacts/deferred-work.md. "+
						"Filter for items related to epic %s (sections mentioning stories starting with \"%s-\"). "+
						"Present each item to me one at a time, showing the full context of the issue. "+
						"For each item, ask me whether I want to: [R] Resolve it now (apply the fix), "+
						"[D] Dismiss it (remove from the file, not needed), or [S] Skip it (keep for later). "+
						"For items I choose to resolve, apply the necessary code changes to the codebase. "+
						"Once done, remove all resolved and dismissed items from deferred-work.md, keeping skipped items. "+
						"If there are no items for this epic, let me know.",
					epicID, epicID,
				)
			} else {
				prompt = "Read the deferred work file at _bmad-output/implementation-artifacts/deferred-work.md. " +
					"Present each item to me one at a time, showing the full context of the issue. " +
					"For each item, ask me whether I want to: [R] Resolve it now (apply the fix), " +
					"[D] Dismiss it (remove from the file, not needed), or [S] Skip it (keep for later). " +
					"For items I choose to resolve, apply the necessary code changes to the codebase. " +
					"Once done, remove all resolved and dismissed items from deferred-work.md, keeping skipped items. " +
					"If the file is empty or does not exist, let me know."
			}

			binaryPath := app.Config.Claude.BinaryPath
			claudeCmd := exec.Command(binaryPath, prompt)
			claudeCmd.Stdin = os.Stdin
			claudeCmd.Stdout = os.Stdout
			claudeCmd.Stderr = os.Stderr

			if err := claudeCmd.Run(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					return NewExitError(exitErr.ExitCode())
				}
				cmd.SilenceUsage = true
				return fmt.Errorf("failed to start claude: %w", err)
			}

			return nil
		},
	}
}
