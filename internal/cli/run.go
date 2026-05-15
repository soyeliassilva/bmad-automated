package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"bmad-automate/internal/lifecycle"
	"bmad-automate/internal/router"
)

func newRunCommand(app *App) *cobra.Command {
	var dryRun bool
	var gitLocal bool

	cmd := &cobra.Command{
		Use:   "run <story-key>",
		Short: "Run the full story lifecycle to completion",
		Long: `Run the complete lifecycle for a story from its current status to done.

The command executes all remaining workflows based on the story's current status:
  - backlog       → create-story → dev-story → code-review → git-commit → done
  - ready-for-dev → dev-story → code-review → git-commit → done
  - in-progress   → dev-story → code-review → git-commit → done
  - review        → code-review → git-commit → done
  - done          → no action (story already complete)

Status is updated in sprint-status.yaml after each successful workflow.

Use --dry-run to preview workflows without executing them.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storyKey := args[0]
			ctx := cmd.Context()

			// Apply git-local flag to config
			if gitLocal {
				app.Config.GitLocal = true
			}

			// Create lifecycle executor with app dependencies
			executor := lifecycle.NewExecutor(app.Runner, app.StatusReader, app.StatusWriter)

			// Handle dry-run mode
			if dryRun {
				steps, err := executor.GetSteps(storyKey)
				if err != nil {
					cmd.SilenceUsage = true
					if errors.Is(err, router.ErrStoryComplete) {
						fmt.Printf("Story is already complete, no workflows to run\n")
						return nil
					}
					fmt.Printf("Error: %v\n", err)
					return NewExitError(1)
				}

				fmt.Printf("Dry run for story %s:\n", storyKey)
				for i, step := range steps {
					fmt.Printf("  %d. %s → %s\n", i+1, step.Workflow, step.NextStatus)
				}
				return nil
			}

			// Set up progress callback to show step progress
			executor.SetProgressCallback(func(stepIndex, totalSteps int, workflow string) {
				app.Printer.StepStart(stepIndex, totalSteps, workflow)
			})

			// Execute the full lifecycle
			err := executor.Execute(ctx, storyKey)
			if err != nil {
				cmd.SilenceUsage = true
				if errors.Is(err, router.ErrStoryComplete) {
					fmt.Printf("Story %s is already complete, no action needed\n", storyKey)
					return nil
				}
				fmt.Printf("Error: %v\n", err)
				return NewExitError(1)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview workflows without executing them")
	cmd.Flags().BoolVar(&gitLocal, "git-local", false, "Commit locally without pushing to remote")

	return cmd
}
