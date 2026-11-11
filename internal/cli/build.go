package cli

import (
	"github.com/spf13/cobra"
)

func newBuildCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "build <story-key>",
		Short: "Run build workflow",
		Long: `Run the build workflow for the specified story key.

Dispatches bmad-build-auto, which plans the story spec, implements every task,
verifies the result and runs an adversarial review with auto-fix in a single
unattended pass. It commits its own reviewed diff but never pushes.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storyKey := args[0]
			ctx := cmd.Context()
			exitCode := app.Runner.RunSingle(ctx, "build", storyKey)
			if exitCode != 0 {
				cmd.SilenceUsage = true
				return NewExitError(exitCode)
			}
			return nil
		},
	}
}
