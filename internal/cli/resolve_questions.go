package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func newResolveQuestionsCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "resolve-questions <epic-id>",
		Short: "Interactively resolve deferred questions for an epic",
		Long: `Launch an interactive Claude session to resolve deferred questions logged
during automated epic execution.

During automated runs (epic, run, queue), Claude logs uncertainties and
assumptions to _bmad-output/implementation-artifacts/deferred-questions.md
instead of blocking for user input.

This command starts an interactive Claude session that:
  1. Reads the deferred questions file
  2. Filters questions for the specified epic
  3. Presents each question with its context, decision, and alternatives
  4. Asks you for the correct answer
  5. Applies any necessary code changes based on your answers
  6. Removes resolved questions from the file

Example:
  bmad-automate resolve-questions 3`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			epicID := args[0]

			prompt := fmt.Sprintf(
				"Read the deferred questions file at _bmad-output/implementation-artifacts/deferred-questions.md. "+
					"Filter for questions related to epic %s (story keys starting with \"%s-\"). "+
					"Present each question to me one at a time, showing the context, the decision that was made, "+
					"the alternatives considered, the confidence level, and the files affected. "+
					"For each question, ask me what the correct answer should be. "+
					"After I answer, apply any necessary code changes to the codebase. "+
					"Once all questions for this epic are resolved, remove the resolved entries from deferred-questions.md. "+
					"If there are no questions for this epic, let me know.",
				epicID, epicID,
			)

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
