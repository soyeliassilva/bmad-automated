// Package status provides functionality for reading and writing sprint status YAML files.
//
// The sprint-status.yaml file tracks the development status of stories throughout
// their lifecycle. Each story progresses through statuses: backlog -> ready-for-dev ->
// in-progress -> review -> done.
//
// Key types:
//   - [Status] - Story development status enum with validation
//   - [SprintStatus] - Parsed representation of sprint-status.yaml
//   - [Reader] - Reads and queries sprint status from YAML files
//   - [Writer] - Updates status values while preserving YAML formatting
//
// The package uses yaml.v3's Node API for writes to preserve comments, ordering,
// and formatting in the status file.
package status

// Status represents a story's development status in the workflow lifecycle.
//
// A story progresses through statuses as it moves through development:
// backlog -> ready-for-dev -> in-progress -> review -> done.
//
// This is the sprint-status.yaml vocabulary. Story specs written by
// bmad-build-auto use a different one, see [SpecStatus].
//
// The status determines which workflow command is executed when running a story.
// See the internal/router package for status-to-workflow mapping.
type Status string

// Status constants define the valid development statuses in the story lifecycle.
const (
	// StatusBacklog indicates a story that has not been started.
	// Stories in backlog trigger the build workflow.
	StatusBacklog Status = "backlog"

	// StatusReadyForDev indicates a story is ready for development.
	// Stories in ready-for-dev trigger the build workflow.
	StatusReadyForDev Status = "ready-for-dev"

	// StatusInProgress indicates a story is actively being developed.
	// Stories in progress trigger the build workflow to continue work.
	StatusInProgress Status = "in-progress"

	// StatusReview indicates a story is ready for code review.
	// Stories in review trigger the code-review workflow.
	StatusReview Status = "review"

	// StatusDone indicates a story has completed all workflow steps.
	// Stories marked done are skipped in queue and epic operations.
	StatusDone Status = "done"
)

// IsValid reports whether the status is one of the known valid status values.
// It returns true for backlog, ready-for-dev, in-progress, review, and done.
func (s Status) IsValid() bool {
	switch s {
	case StatusBacklog, StatusReadyForDev, StatusInProgress, StatusReview, StatusDone:
		return true
	default:
		return false
	}
}

// SprintStatus represents the parsed contents of a sprint-status.yaml file.
//
// The file structure contains a development_status map where keys are story
// identifiers (e.g., "7-1-define-schema") and values are their current [Status].
type SprintStatus struct {
	// DevelopmentStatus maps story keys to their current development status.
	// Story keys follow the pattern: {epicID}-{storyNum}-{description}.
	DevelopmentStatus map[string]Status `yaml:"development_status"`
}
