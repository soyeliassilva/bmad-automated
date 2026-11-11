package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bmad-automate/internal/config"
	"bmad-automate/internal/status"
)

// TestEpicCommand_FullLifecycleExecution tests that epic command executes the full lifecycle for each story
func TestEpicCommand_FullLifecycleExecution(t *testing.T) {
	tests := []struct {
		name              string
		epicID            string
		statusYAML        string
		expectedWorkflows []string
		expectedStatuses  []StatusUpdate
		expectError       bool
		failOnWorkflow    string
	}{
		{
			name:   "epic with 2 backlog stories runs full lifecycle for each",
			epicID: "6",
			statusYAML: `development_status:
  6-1-first: backlog
  6-2-second: backlog`,
			// Each backlog story should run build then commit
			expectedWorkflows: []string{
				// Story 6-1
				"build", "git-commit",
				// Story 6-2
				"build", "git-commit",
			},
			expectedStatuses: []StatusUpdate{
				// Story 6-1-first
				{StoryKey: "6-1-first", NewStatus: status.StatusReview},
				{StoryKey: "6-1-first", NewStatus: status.StatusDone},
				// Story 6-2-second
				{StoryKey: "6-2-second", NewStatus: status.StatusReview},
				{StoryKey: "6-2-second", NewStatus: status.StatusDone},
			},
			expectError: false,
		},
		{
			name:   "epic with mixed statuses runs appropriate remaining workflows",
			epicID: "6",
			statusYAML: `development_status:
  6-1-first: backlog
  6-2-second: ready-for-dev
  6-3-third: review`,
			expectedWorkflows: []string{
				// Story 6-1 (backlog): build then commit
				"build", "git-commit",
				// Story 6-2 (ready-for-dev): build then commit
				"build", "git-commit",
				// Story 6-3 (review): code-review then commit
				"code-review", "git-commit",
			},
			expectedStatuses: []StatusUpdate{
				// Story 6-1-first
				{StoryKey: "6-1-first", NewStatus: status.StatusReview},
				{StoryKey: "6-1-first", NewStatus: status.StatusDone},
				// Story 6-2-second
				{StoryKey: "6-2-second", NewStatus: status.StatusReview},
				{StoryKey: "6-2-second", NewStatus: status.StatusDone},
				// Story 6-3-third
				{StoryKey: "6-3-third", NewStatus: status.StatusDone},
				{StoryKey: "6-3-third", NewStatus: status.StatusDone},
			},
			expectError: false,
		},
		{
			name:   "epic with done story skips done and runs others",
			epicID: "6",
			statusYAML: `development_status:
  6-1-first: backlog
  6-2-done: done
  6-3-third: ready-for-dev`,
			// Done story is skipped, others run build then commit
			expectedWorkflows: []string{
				// Story 6-1 (backlog): build then commit
				"build", "git-commit",
				// Story 6-2 (done): skipped, no workflows
				// Story 6-3 (ready-for-dev): build then commit
				"build", "git-commit",
			},
			expectedStatuses: []StatusUpdate{
				// Story 6-1-first
				{StoryKey: "6-1-first", NewStatus: status.StatusReview},
				{StoryKey: "6-1-first", NewStatus: status.StatusDone},
				// Story 6-3-third
				{StoryKey: "6-3-third", NewStatus: status.StatusReview},
				{StoryKey: "6-3-third", NewStatus: status.StatusDone},
			},
			expectError: false,
		},
		{
			name:   "story failure mid-lifecycle stops processing",
			epicID: "6",
			statusYAML: `development_status:
  6-1-first: backlog
  6-2-second: backlog`,
			failOnWorkflow: "git-commit",
			// First story: build succeeds, git-commit fails, stops
			expectedWorkflows: []string{"build", "git-commit"},
			expectedStatuses: []StatusUpdate{
				// Story 6-1-first
				{StoryKey: "6-1-first", NewStatus: status.StatusReview},
			},
			expectError: true,
		},
		{
			name:   "all stories done returns success with no workflows",
			epicID: "6",
			statusYAML: `development_status:
  6-1-first: done
  6-2-second: done`,
			expectedWorkflows: nil,
			expectedStatuses:  nil,
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			createSprintStatusFile(t, tmpDir, tt.statusYAML)

			mockRunner := &MockWorkflowRunner{
				FailOnWorkflow: tt.failOnWorkflow,
			}
			mockWriter := &MockStatusWriter{}
			statusReader := status.NewReader(tmpDir)

			app := &App{
				Config:       config.DefaultConfig(),
				StatusReader: statusReader,
				StatusWriter: mockWriter,
				Runner:       mockRunner,
			}

			rootCmd := NewRootCommand(app)
			outBuf := &bytes.Buffer{}
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(outBuf)
			rootCmd.SetArgs([]string{"epic", tt.epicID})

			err := rootCmd.Execute()

			if tt.expectError {
				require.Error(t, err)
				code, ok := IsExitError(err)
				assert.True(t, ok, "error should be an ExitError")
				assert.Equal(t, 1, code)
			} else {
				assert.NoError(t, err)
			}

			// Verify workflows were executed in order
			assert.Equal(t, tt.expectedWorkflows, mockRunner.ExecutedWorkflows,
				"workflows should be executed in lifecycle order for each story")

			// Verify status updates occurred after each workflow
			if tt.expectedStatuses != nil {
				require.Len(t, mockWriter.Updates, len(tt.expectedStatuses),
					"should have correct number of status updates")
				for i, expected := range tt.expectedStatuses {
					assert.Equal(t, expected.StoryKey, mockWriter.Updates[i].StoryKey,
						"status update %d should be for story %s", i, expected.StoryKey)
					assert.Equal(t, expected.NewStatus, mockWriter.Updates[i].NewStatus,
						"status update %d should be %s", i, expected.NewStatus)
				}
			} else {
				assert.Empty(t, mockWriter.Updates, "should have no status updates")
			}
		})
	}
}

func TestEpicCommand_NoStoriesFoundReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	createSprintStatusFile(t, tmpDir, `development_status:
  7-1-other-epic: backlog`)

	mockRunner := &MockWorkflowRunner{}
	mockWriter := &MockStatusWriter{}
	statusReader := status.NewReader(tmpDir)

	app := &App{
		Config:       config.DefaultConfig(),
		StatusReader: statusReader,
		StatusWriter: mockWriter,
		Runner:       mockRunner,
	}

	rootCmd := NewRootCommand(app)
	outBuf := &bytes.Buffer{}
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(outBuf)
	rootCmd.SetArgs([]string{"epic", "6"})

	err := rootCmd.Execute()

	require.Error(t, err)
	code, ok := IsExitError(err)
	assert.True(t, ok, "error should be an ExitError")
	assert.Equal(t, 1, code)

	// No workflows should have been executed
	assert.Empty(t, mockRunner.ExecutedWorkflows)
}

func TestEpicCommand_MultipleEpics(t *testing.T) {
	tests := []struct {
		name              string
		epicIDs           []string
		statusYAML        string
		expectedWorkflows []string
		expectedStatuses  []StatusUpdate
		expectError       bool
		failOnWorkflow    string
	}{
		{
			name:    "two epics run stories in epic order",
			epicIDs: []string{"2", "3"},
			statusYAML: `development_status:
  2-1-auth: backlog
  2-2-login: backlog
  3-1-dashboard: backlog`,
			expectedWorkflows: []string{
				// Epic 2 stories first
				"build", "git-commit",
				"build", "git-commit",
				// Then epic 3
				"build", "git-commit",
			},
			expectedStatuses: []StatusUpdate{
				// Story 2-1-auth
				{StoryKey: "2-1-auth", NewStatus: status.StatusReview},
				{StoryKey: "2-1-auth", NewStatus: status.StatusDone},
				// Story 2-2-login
				{StoryKey: "2-2-login", NewStatus: status.StatusReview},
				{StoryKey: "2-2-login", NewStatus: status.StatusDone},
				// Story 3-1-dashboard
				{StoryKey: "3-1-dashboard", NewStatus: status.StatusReview},
				{StoryKey: "3-1-dashboard", NewStatus: status.StatusDone},
			},
		},
		{
			name:    "three epics with mixed statuses",
			epicIDs: []string{"2", "3", "4"},
			statusYAML: `development_status:
  2-1-auth: done
  3-1-dashboard: ready-for-dev
  4-1-deploy: review`,
			expectedWorkflows: []string{
				// Epic 2: done, skipped
				// Epic 3: ready-for-dev
				"build", "git-commit",
				// Epic 4: review
				"code-review", "git-commit",
			},
			expectedStatuses: []StatusUpdate{
				// Story 3-1-dashboard
				{StoryKey: "3-1-dashboard", NewStatus: status.StatusReview},
				{StoryKey: "3-1-dashboard", NewStatus: status.StatusDone},
				// Story 4-1-deploy
				{StoryKey: "4-1-deploy", NewStatus: status.StatusDone},
				{StoryKey: "4-1-deploy", NewStatus: status.StatusDone},
			},
		},
		{
			name:    "failure in second epic stops processing",
			epicIDs: []string{"2", "3"},
			statusYAML: `development_status:
  2-1-auth: backlog
  3-1-dashboard: backlog`,
			failOnWorkflow: "git-commit",
			// Epic 2 story: build OK, git-commit fails
			expectedWorkflows: []string{"build", "git-commit"},
			expectedStatuses: []StatusUpdate{
				// Story 2-1-auth
				{StoryKey: "2-1-auth", NewStatus: status.StatusReview},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			createSprintStatusFile(t, tmpDir, tt.statusYAML)

			mockRunner := &MockWorkflowRunner{
				FailOnWorkflow: tt.failOnWorkflow,
			}
			mockWriter := &MockStatusWriter{}
			statusReader := status.NewReader(tmpDir)

			app := &App{
				Config:       config.DefaultConfig(),
				StatusReader: statusReader,
				StatusWriter: mockWriter,
				Runner:       mockRunner,
			}

			rootCmd := NewRootCommand(app)
			outBuf := &bytes.Buffer{}
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(outBuf)

			args := append([]string{"epic"}, tt.epicIDs...)
			rootCmd.SetArgs(args)

			err := rootCmd.Execute()

			if tt.expectError {
				require.Error(t, err)
				code, ok := IsExitError(err)
				assert.True(t, ok)
				assert.Equal(t, 1, code)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedWorkflows, mockRunner.ExecutedWorkflows)

			if tt.expectedStatuses != nil {
				require.Len(t, mockWriter.Updates, len(tt.expectedStatuses))
				for i, expected := range tt.expectedStatuses {
					assert.Equal(t, expected.StoryKey, mockWriter.Updates[i].StoryKey)
					assert.Equal(t, expected.NewStatus, mockWriter.Updates[i].NewStatus)
				}
			}
		})
	}
}

func TestEpicCommand_MultipleEpics_SecondEpicNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	createSprintStatusFile(t, tmpDir, `development_status:
  2-1-auth: backlog`)

	mockRunner := &MockWorkflowRunner{}
	mockWriter := &MockStatusWriter{}
	statusReader := status.NewReader(tmpDir)

	app := &App{
		Config:       config.DefaultConfig(),
		StatusReader: statusReader,
		StatusWriter: mockWriter,
		Runner:       mockRunner,
	}

	rootCmd := NewRootCommand(app)
	outBuf := &bytes.Buffer{}
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(outBuf)
	rootCmd.SetArgs([]string{"epic", "2", "99"})

	err := rootCmd.Execute()

	require.Error(t, err)
	code, ok := IsExitError(err)
	assert.True(t, ok)
	assert.Equal(t, 1, code)

	// No workflows should execute since epic discovery fails before execution
	assert.Empty(t, mockRunner.ExecutedWorkflows)
}
