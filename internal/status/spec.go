package status

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultSpecDir is the directory bmad-build-auto writes story specs into,
// relative to the project root.
const DefaultSpecDir = "_bmad-output/implementation-artifacts"

// SpecStatus is the terminal status bmad-build-auto records in a story spec's
// frontmatter.
//
// This is a different vocabulary from [Status], which tracks sprint-status.yaml.
// The two overlap on ready-for-dev, in-progress and done, and diverge on draft,
// blocked and in-review (sprint-status.yaml calls the latter "review").
type SpecStatus string

// SpecStatus values written by bmad-build-auto.
const (
	// SpecDraft means planning started but never reached the ready standard.
	SpecDraft SpecStatus = "draft"

	// SpecReadyForDev means the spec passed the ready standard. It is terminal
	// only when the run was told to halt after planning.
	SpecReadyForDev SpecStatus = "ready-for-dev"

	// SpecInProgress means implementation started and did not finish.
	SpecInProgress SpecStatus = "in-progress"

	// SpecInReview means the review pass started and did not finish.
	SpecInReview SpecStatus = "in-review"

	// SpecBlocked means the run halted deliberately. The spec carries a blocking
	// condition explaining why.
	SpecBlocked SpecStatus = "blocked"

	// SpecDone means the run completed: implemented, reviewed and committed.
	SpecDone SpecStatus = "done"
)

// ErrSpecNotFound indicates no story spec could be located for a story key.
//
// Callers cannot verify the outcome of a build in this case and must fall back
// to the process exit code.
var ErrSpecNotFound = errors.New("no story spec found")

// Spec is the outcome bmad-build-auto recorded for a story.
type Spec struct {
	// Path is the spec file the status was read from.
	Path string

	// Status is the frontmatter status value.
	Status SpecStatus

	// Blocking is the blocking condition from the "Auto Run Result" section.
	// Empty unless the run halted.
	Blocking string

	// ModTime is when the spec was last written. Callers use it to tell a spec
	// this run produced from one left behind by an earlier run.
	ModTime time.Time
}

// ReadSpec returns the outcome bmad-build-auto recorded for a story key.
//
// bmad-build-auto derives its own spec filename from the clarified intent, leading
// the slug with the story's tracking identifier, so the file is located by globbing
// on that identifier rather than by an exact name. When several specs match, the
// most recently modified one wins, which is the one the run just wrote.
//
// Returns [ErrSpecNotFound] when no spec matches. Exit code 0 alone does not mean
// the build succeeded: a deliberate HALT also exits 0, so callers must check the
// status here before advancing a story.
func (r *Reader) ReadSpec(storyKey string) (*Spec, error) {
	path, err := r.findSpec(storyKey)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat story spec: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read story spec: %w", err)
	}

	specStatus, err := parseFrontmatterStatus(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return &Spec{
		Path:     path,
		Status:   specStatus,
		Blocking: parseBlockingCondition(data),
		ModTime:  info.ModTime(),
	}, nil
}

// findSpec locates the most recently modified spec file for a story key.
func (r *Reader) findSpec(storyKey string) (string, error) {
	prefix := "spec-" + specKeyPrefix(storyKey)
	pattern := filepath.Join(r.basePath, DefaultSpecDir, prefix+"*.md")

	globbed, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to search for story spec: %w", err)
	}

	// The glob is deliberately loose, so drop anything where the prefix runs into
	// a longer identifier: story 7-1 must not pick up spec-7-10-other-story.md.
	var matches []string
	for _, m := range globbed {
		rest := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(m), prefix), ".md")
		if rest == "" || strings.HasPrefix(rest, "-") {
			matches = append(matches, m)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("%w for story %s (looked for %s)", ErrSpecNotFound, storyKey, pattern)
	}

	newest := matches[0]
	newestTime := int64(0)
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil {
			continue
		}
		if t := info.ModTime().UnixNano(); t > newestTime {
			newest, newestTime = m, t
		}
	}

	return newest, nil
}

// specKeyPrefix reduces a story key to the tracking identifier bmad-build-auto
// leads its slug with: "7-1-define-schema" becomes "7-1". Keys that do not carry
// a numeric story number are used whole.
func specKeyPrefix(storyKey string) string {
	parts := strings.Split(storyKey, "-")
	if len(parts) < 2 {
		return storyKey
	}
	if _, err := strconv.Atoi(parts[1]); err != nil {
		return storyKey
	}
	return parts[0] + "-" + parts[1]
}

// parseFrontmatterStatus extracts the status field from a spec's YAML frontmatter.
func parseFrontmatterStatus(data []byte) (SpecStatus, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return "", errors.New("story spec has no frontmatter")
	}

	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return "", errors.New("story spec frontmatter is unterminated")
	}

	var frontmatter struct {
		Status SpecStatus `yaml:"status"`
	}
	if err := yaml.Unmarshal([]byte(text[4:4+end]), &frontmatter); err != nil {
		return "", fmt.Errorf("failed to parse story spec frontmatter: %w", err)
	}
	if frontmatter.Status == "" {
		return "", errors.New("story spec frontmatter has no status")
	}

	return frontmatter.Status, nil
}

// parseBlockingCondition extracts the blocking condition bmad-build-auto writes
// under the "Auto Run Result" heading. Returns an empty string when absent.
func parseBlockingCondition(data []byte) string {
	const marker = "Blocking condition:"

	// The line may be plain, a bullet, or bold, depending on which HALT branch
	// wrote it, so strip markdown emphasis and any bullet before matching.
	unstyle := strings.NewReplacer("*", "", "_", "", "`", "")

	for _, line := range strings.Split(string(data), "\n") {
		cleaned := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(unstyle.Replace(line)), "-"))
		if after, ok := strings.CutPrefix(cleaned, marker); ok {
			return strings.TrimSpace(after)
		}
	}

	return ""
}
