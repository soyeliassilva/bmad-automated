// Package lifecycle orchestrates story lifecycle execution from current status to done.
//
// The lifecycle package provides [Executor] which runs stories through their complete
// workflow sequence (build->commit) based on current status. Each step updates the
// story status automatically after successful completion.
//
// Key concepts:
//   - Lifecycle steps are determined by [router.GetLifecycle] based on current status
//   - Each step runs a workflow then updates status via [StatusWriter]
//   - Build steps are verified against the story spec before the status advances
//   - Progress can be tracked via [ProgressCallback]
package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bmad-automate/internal/router"
	"bmad-automate/internal/status"
)

// WorkflowRunner is the interface for executing individual workflows.
//
// RunSingle executes a named workflow for a story and returns the exit code.
// An exit code of 0 indicates success; any non-zero value indicates failure.
// The [workflow.Runner] type implements this interface.
type WorkflowRunner interface {
	RunSingle(ctx context.Context, workflowName, storyKey string) int
}

// StatusReader is the interface for looking up story status.
//
// GetStoryStatus retrieves the current [status.Status] for a story key from
// sprint-status.yaml. ReadSpec retrieves the outcome bmad-build-auto recorded in
// the story spec, which is the only reliable signal of whether a build finished.
type StatusReader interface {
	GetStoryStatus(storyKey string) (status.Status, error)
	ReadSpec(storyKey string) (*status.Spec, error)
}

// StatusWriter is the interface for persisting story status updates.
//
// UpdateStatus sets a new [status.Status] for a story after successful workflow completion.
// It returns an error if the status file cannot be written.
type StatusWriter interface {
	UpdateStatus(storyKey string, newStatus status.Status) error
}

// ProgressCallback is invoked before each workflow step begins execution.
//
// The callback receives stepIndex (1-based), totalSteps count, and the workflow name.
// This enables progress reporting in the UI. The callback is optional and can be set
// via [Executor.SetProgressCallback].
type ProgressCallback func(stepIndex, totalSteps int, workflow string)

// Executor orchestrates the complete story lifecycle from current status to done.
//
// Executor uses dependency injection for testability: [WorkflowRunner] executes workflows,
// [StatusReader] looks up current status, and [StatusWriter] persists status updates.
// Use [NewExecutor] to create an instance and [Execute] to run the lifecycle.
type Executor struct {
	runner           WorkflowRunner
	statusReader     StatusReader
	statusWriter     StatusWriter
	progressCallback ProgressCallback
}

// NewExecutor creates a new Executor with the required dependencies.
//
// The runner executes workflows, reader looks up story status, and writer persists
// status updates. Progress callback is not set by default; use [SetProgressCallback]
// to enable progress reporting.
func NewExecutor(runner WorkflowRunner, reader StatusReader, writer StatusWriter) *Executor {
	return &Executor{
		runner:       runner,
		statusReader: reader,
		statusWriter: writer,
	}
}

// SetProgressCallback configures an optional progress callback for workflow execution.
//
// The callback receives the step index (1-based), total step count, and workflow name
// before each workflow begins. This is typically used to display progress information
// in the terminal UI.
func (e *Executor) SetProgressCallback(cb ProgressCallback) {
	e.progressCallback = cb
}

// Execute runs the complete story lifecycle from current status to done.
//
// Execute looks up the story's current status, determines the remaining workflow steps
// via [router.GetLifecycle], and runs each workflow in sequence. After each successful
// workflow, the story status is updated to the next state.
//
// Execute uses fail-fast behavior: it stops on the first error and returns immediately.
// Errors can occur from status lookup failure, workflow execution failure (non-zero exit),
// a build that halted instead of completing, or status update failure. For stories
// already done, Execute returns [router.ErrStoryComplete].
func (e *Executor) Execute(ctx context.Context, storyKey string) error {
	// Get current story status
	currentStatus, err := e.statusReader.GetStoryStatus(storyKey)
	if err != nil {
		return err
	}

	// Get lifecycle steps from current status
	steps, err := router.GetLifecycle(currentStatus)
	if err != nil {
		return err // Returns router.ErrStoryComplete for done stories
	}

	// Get total steps count for progress reporting
	totalSteps := len(steps)

	// Execute each step in sequence
	for i, step := range steps {
		// Call progress callback if set
		if e.progressCallback != nil {
			e.progressCallback(i+1, totalSteps, step.Workflow)
		}

		// Run the workflow
		startedAt := time.Now()
		exitCode := e.runner.RunSingle(ctx, step.Workflow, storyKey)
		if exitCode != 0 {
			return fmt.Errorf("workflow failed: %s returned exit code %d", step.Workflow, exitCode)
		}

		// A build that halted still exits 0, so confirm it actually finished
		if step.Workflow == router.WorkflowBuild {
			if err := e.verifyBuild(storyKey, startedAt); err != nil {
				return err
			}
		}

		// Update status after successful workflow
		if err := e.statusWriter.UpdateStatus(storyKey, step.NextStatus); err != nil {
			return err
		}
	}

	return nil
}

// verifyBuild confirms bmad-build-auto ran to completion rather than halting.
//
// bmad-build-auto ends a blocked run through its HALT protocol, which writes a
// terminal status and a blocking condition into the story spec and then exits
// normally. The CLI exit code is 0 either way, so without this check a story that
// stopped on an intent gap would be recorded as done.
//
// A missing spec is reported and tolerated, since the outcome cannot be read and
// the exit code is the only remaining signal. A spec older than startedAt was left
// behind by an earlier run and says nothing about this one, so it is tolerated the
// same way rather than being read as this run's verdict.
func (e *Executor) verifyBuild(storyKey string, startedAt time.Time) error {
	spec, err := e.statusReader.ReadSpec(storyKey)
	if errors.Is(err, status.ErrSpecNotFound) {
		fmt.Printf("Warning: cannot verify build outcome for %s: %v\n", storyKey, err)
		return nil
	}
	if err != nil {
		return err
	}

	if spec.ModTime.Before(startedAt) {
		fmt.Printf("Warning: cannot verify build outcome for %s: %s predates this run\n", storyKey, spec.Path)
		return nil
	}

	if spec.Status == status.SpecDone {
		return nil
	}

	if spec.Blocking != "" {
		return fmt.Errorf("build halted for %s with status %q: %s (see %s)", storyKey, spec.Status, spec.Blocking, spec.Path)
	}
	return fmt.Errorf("build halted for %s with status %q (see %s)", storyKey, spec.Status, spec.Path)
}

// GetSteps returns the remaining lifecycle steps for a story without executing them.
//
// GetSteps provides dry-run preview functionality, showing what workflows would execute
// and what status transitions would occur. This is useful for displaying the planned
// execution path before actually running workflows.
//
// Returns an error if status lookup fails. For stories already done, returns
// [router.ErrStoryComplete].
func (e *Executor) GetSteps(storyKey string) ([]router.LifecycleStep, error) {
	// Get current story status
	currentStatus, err := e.statusReader.GetStoryStatus(storyKey)
	if err != nil {
		return nil, err
	}

	// Get lifecycle steps from current status
	steps, err := router.GetLifecycle(currentStatus)
	if err != nil {
		return nil, err // Returns router.ErrStoryComplete for done stories
	}

	return steps, nil
}
