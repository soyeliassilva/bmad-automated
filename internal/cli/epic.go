package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"bmad-automate/internal/lifecycle"
	"bmad-automate/internal/router"
)

func newEpicCommand(app *App) *cobra.Command {
	var dryRun bool
	var gitLocal bool
	var reviewer string
	var autoRetry bool

	cmd := &cobra.Command{
		Use:   "epic <epic-id> [epic-id...]",
		Short: "Run full lifecycle for all stories in one or more epics",
		Long: `Run the complete lifecycle for all stories in one or more epics to completion.

Finds all stories matching the pattern {epic-id}-{N}-* where N is numeric,
sorts them by story number, and runs each to completion before moving to the next.
When multiple epic IDs are provided, epics are processed in the order given.

For each story, executes all remaining workflows based on its current status:
  - backlog       → create-story → dev-story → code-review → git-commit → done
  - ready-for-dev → dev-story → code-review → git-commit → done
  - in-progress   → dev-story → code-review → git-commit → done
  - review        → code-review → git-commit → done
  - done          → skipped (story already complete)

The epic command stops on the first failure. Done stories are skipped and do not cause failure.
Status is updated in sprint-status.yaml after each successful workflow.

Use --dry-run to preview workflows without executing them.

Examples:
  bmad-automate epic 6
  bmad-automate epic 2 3 4
  bmad-automate epic 2 3 4 --git-local --reviewer codex --auto-retry`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Collect stories from all epics in order
			var epics []epicStoryGroup

			for _, epicID := range args {
				storyKeys, err := app.StatusReader.GetEpicStories(epicID)
				if err != nil {
					cmd.SilenceUsage = true
					return NewExitError(1)
				}
				epics = append(epics, epicStoryGroup{epicID: epicID, storyKeys: storyKeys})
			}

			// Apply git-local flag to config
			if gitLocal {
				app.Config.GitLocal = true
			}

			// Apply auto-retry if set
			if autoRetry {
				applyAutoRetry(app)
			}

			// Apply reviewer override if set
			if err := applyReviewerOverride(app, reviewer); err != nil {
				cmd.SilenceUsage = true
				return err
			}

			// Create lifecycle executor with app dependencies
			executor := lifecycle.NewExecutor(app.Runner, app.StatusReader, app.StatusWriter)

			// Handle dry-run mode
			if dryRun {
				return runEpicsDryRun(cmd, executor, epics)
			}

			// Execute full lifecycle for each story in each epic
			totalStories := 0
			for _, ep := range epics {
				if len(args) > 1 {
					fmt.Printf("=== Epic %s ===\n", ep.epicID)
				}
				for _, storyKey := range ep.storyKeys {
					err := executor.Execute(ctx, storyKey)
					if err != nil {
						cmd.SilenceUsage = true
						if errors.Is(err, router.ErrStoryComplete) {
							fmt.Printf("Story %s is already complete, skipping\n", storyKey)
							totalStories++
							continue
						}
						fmt.Printf("Error running lifecycle for story %s: %v\n", storyKey, err)
						return NewExitError(1)
					}
					fmt.Printf("Story %s completed successfully\n", storyKey)
					totalStories++
				}
			}

			if len(args) > 1 {
				fmt.Printf("All %d stories across %d epics processed\n", totalStories, len(args))
			} else {
				fmt.Printf("All %d stories processed\n", totalStories)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview workflows without executing them")
	cmd.Flags().BoolVar(&gitLocal, "git-local", false, "Commit locally without pushing to remote")
	cmd.Flags().StringVar(&reviewer, "reviewer", "", "Use alternative CLI for code-review (e.g., codex)")
	cmd.Flags().BoolVar(&autoRetry, "auto-retry", false, "Automatically wait and retry when rate limited")

	return cmd
}

type epicStoryGroup struct {
	epicID    string
	storyKeys []string
}

func runEpicsDryRun(cmd *cobra.Command, executor *lifecycle.Executor, epics []epicStoryGroup) error {
	multiEpic := len(epics) > 1

	if multiEpic {
		epicIDs := make([]string, len(epics))
		for i, ep := range epics {
			epicIDs[i] = ep.epicID
		}
		fmt.Printf("Dry run for epics %s:\n", strings.Join(epicIDs, ", "))
	} else {
		fmt.Printf("Dry run for epic %s:\n", epics[0].epicID)
	}

	totalWorkflows := 0
	storiesWithWork := 0
	storiesComplete := 0

	for _, ep := range epics {
		if multiEpic {
			fmt.Printf("\n=== Epic %s ===\n", ep.epicID)
		}

		for _, storyKey := range ep.storyKeys {
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
	}

	fmt.Println()
	if storiesComplete > 0 {
		fmt.Printf("Total: %d workflows across %d stories (%d already complete)\n", totalWorkflows, storiesWithWork, storiesComplete)
	} else {
		fmt.Printf("Total: %d workflows across %d stories\n", totalWorkflows, storiesWithWork)
	}

	return nil
}
