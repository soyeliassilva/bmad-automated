package cli

import (
	"github.com/spf13/cobra"
)

func newGitCommitCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "git-commit <story-key>",
		Short: "Commit leftover changes for a story and push",
		Long: `Commit any uncommitted changes for the specified story and push to the current branch.

bmad-build-auto commits its own reviewed diff without pushing, so after a build
this usually only has the push left to do.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storyKey := args[0]
			ctx := cmd.Context()
			exitCode := app.Runner.RunSingle(ctx, "git-commit", storyKey)
			if exitCode != 0 {
				cmd.SilenceUsage = true
				return NewExitError(exitCode)
			}
			return nil
		},
	}
}
