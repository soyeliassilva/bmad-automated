package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"bmad-automate/internal/lifecycle"
	"bmad-automate/internal/router"
)

func newQueueCommand(app *App) *cobra.Command {
	var dryRun bool
	var gitLocal bool

	cmd := &cobra.Command{
		Use:   "queue <story-key> [story-key...]",
		Short: "Run full lifecycle for multiple stories",
		Long: `Run the complete lifecycle for multiple stories to completion.

Each story is run to completion before moving to the next.

For each story, executes all remaining workflows based on its current status:
  - backlog       → create-story → dev-story → code-review → git-commit → done
  - ready-for-dev → dev-story → code-review → git-commit → done
  - in-progress   → dev-story → code-review → git-commit → done
  - review        → code-review → git-commit → done
  - done          → skipped (story already complete)

The queue stops on the first failure. Done stories are skipped and do not cause failure.
Status is updated in sprint-status.yaml after each successful workflow.

Use --dry-run to preview workflows without executing them.

Example:
  bmad-automate queue 6-5 6-6 6-7 6-8`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Apply git-local flag to config
			if gitLocal {
				app.Config.GitLocal = true
			}

			// Create lifecycle executor with app dependencies
			executor := lifecycle.NewExecutor(app.Runner, app.StatusReader, app.StatusWriter)

			// Handle dry-run mode
			if dryRun {
				return runQueueDryRun(cmd, executor, args)
			}

			// Execute full lifecycle for each story in order
			for _, storyKey := range args {
				err := executor.Execute(ctx, storyKey)
				if err != nil {
					cmd.SilenceUsage = true
					if errors.Is(err, router.ErrStoryComplete) {
						fmt.Printf("Story %s is already complete, skipping\n", storyKey)
						continue
					}
					fmt.Printf("Error running lifecycle for story %s: %v\n", storyKey, err)
					return NewExitError(1)
				}
				fmt.Printf("Story %s completed successfully\n", storyKey)
			}

			fmt.Printf("All %d stories processed\n", len(args))
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview workflows without executing them")
	cmd.Flags().BoolVar(&gitLocal, "git-local", false, "Commit locally without pushing to remote")

	return cmd
}

func runQueueDryRun(cmd *cobra.Command, executor *lifecycle.Executor, storyKeys []string) error {
	fmt.Printf("Dry run for %d stories:\n", len(storyKeys))

	totalWorkflows := 0
	storiesWithWork := 0
	storiesComplete := 0

	for _, storyKey := range storyKeys {
		fmt.Println()
		fmt.Printf("Story %s:\n", storyKey)

		steps, err := executor.GetSteps(storyKey)
		if err != nil {
			if errors.Is(err, router.ErrStoryComplete) {
				fmt.Printf("  (already complete)\n")
				storiesComplete++
				continue
			}
			cmd.SilenceUsage = true
			fmt.Printf("  Error: %v\n", err)
			return NewExitError(1)
		}

		for i, step := range steps {
			fmt.Printf("  %d. %s → %s\n", i+1, step.Workflow, step.NextStatus)
		}
		totalWorkflows += len(steps)
		storiesWithWork++
	}

	fmt.Println()
	if storiesComplete > 0 {
		fmt.Printf("Total: %d workflows across %d stories (%d already complete)\n", totalWorkflows, storiesWithWork, storiesComplete)
	} else {
		fmt.Printf("Total: %d workflows across %d stories\n", totalWorkflows, storiesWithWork)
	}

	return nil
}
