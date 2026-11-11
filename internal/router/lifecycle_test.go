package router

import (
	"errors"
	"testing"

	"bmad-automate/internal/status"
)

func TestGetLifecycle(t *testing.T) {
	buildThenCommit := []LifecycleStep{
		{Workflow: WorkflowBuild, NextStatus: status.StatusReview},
		{Workflow: WorkflowGitCommit, NextStatus: status.StatusDone},
	}

	tests := []struct {
		name      string
		status    status.Status
		wantSteps []LifecycleStep
		wantErr   error
	}{
		{
			name:      "backlog status returns build then commit",
			status:    status.StatusBacklog,
			wantSteps: buildThenCommit,
		},
		{
			name:      "ready-for-dev status returns build then commit",
			status:    status.StatusReadyForDev,
			wantSteps: buildThenCommit,
		},
		{
			name:      "in-progress status returns build then commit",
			status:    status.StatusInProgress,
			wantSteps: buildThenCommit,
		},
		{
			name:   "review status skips build and runs the extra review layer",
			status: status.StatusReview,
			wantSteps: []LifecycleStep{
				{Workflow: WorkflowCodeReview, NextStatus: status.StatusDone},
				{Workflow: WorkflowGitCommit, NextStatus: status.StatusDone},
			},
		},
		{
			name:      "done status returns ErrStoryComplete",
			status:    status.StatusDone,
			wantSteps: nil,
			wantErr:   ErrStoryComplete,
		},
		{
			name:      "unknown status returns ErrUnknownStatus",
			status:    status.Status("invalid"),
			wantSteps: nil,
			wantErr:   ErrUnknownStatus,
		},
		{
			name:      "empty status returns ErrUnknownStatus",
			status:    status.Status(""),
			wantSteps: nil,
			wantErr:   ErrUnknownStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSteps, gotErr := GetLifecycle(tt.status)

			// Check error
			if tt.wantErr != nil {
				if gotErr == nil {
					t.Errorf("GetLifecycle(%q) err = nil, want %v", tt.status, tt.wantErr)
				} else if !errors.Is(gotErr, tt.wantErr) {
					t.Errorf("GetLifecycle(%q) err = %v, want %v", tt.status, gotErr, tt.wantErr)
				}
				return
			}
			if gotErr != nil {
				t.Errorf("GetLifecycle(%q) err = %v, want nil", tt.status, gotErr)
				return
			}

			// Check steps count
			if len(gotSteps) != len(tt.wantSteps) {
				t.Errorf("GetLifecycle(%q) returned %d steps, want %d", tt.status, len(gotSteps), len(tt.wantSteps))
				return
			}

			// Check each step
			for i, wantStep := range tt.wantSteps {
				gotStep := gotSteps[i]
				if gotStep.Workflow != wantStep.Workflow {
					t.Errorf("GetLifecycle(%q) step[%d].Workflow = %q, want %q", tt.status, i, gotStep.Workflow, wantStep.Workflow)
				}
				if gotStep.NextStatus != wantStep.NextStatus {
					t.Errorf("GetLifecycle(%q) step[%d].NextStatus = %q, want %q", tt.status, i, gotStep.NextStatus, wantStep.NextStatus)
				}
			}
		})
	}
}
