package status

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeSpec creates a story spec under the implementation artifacts directory.
func writeSpec(t *testing.T, base, name, body string) string {
	t.Helper()

	dir := filepath.Join(base, DefaultSpecDir)
	require.NoError(t, os.MkdirAll(dir, 0o755))

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))

	return path
}

func TestReadSpec(t *testing.T) {
	tests := []struct {
		name         string
		file         string
		body         string
		storyKey     string
		wantStatus   SpecStatus
		wantBlocking string
	}{
		{
			name:       "completed run",
			file:       "spec-7-1-define-schema.md",
			body:       "---\nstatus: done\nbaseline_commit: abc123\n---\n\n# Define schema\n",
			storyKey:   "7-1-define-schema",
			wantStatus: SpecDone,
		},
		{
			name:         "halted run carries its blocking condition",
			file:         "spec-7-1-define-schema.md",
			storyKey:     "7-1-define-schema",
			body:         "---\nstatus: blocked\n---\n\n## Auto Run Result\n\nStatus: blocked\nBlocking condition: intent gap\n",
			wantStatus:   SpecBlocked,
			wantBlocking: "intent gap",
		},
		{
			name:         "blocking condition survives list and bold markup",
			file:         "spec-7-1-define-schema.md",
			storyKey:     "7-1-define-schema",
			body:         "---\nstatus: blocked\n---\n\n## Auto Run Result\n\n- **Blocking condition:** no epic spec found\n",
			wantStatus:   SpecBlocked,
			wantBlocking: "no epic spec found",
		},
		{
			name:       "slug diverges from the story key but the number still matches",
			file:       "spec-7-1-schema-for-invoices.md",
			storyKey:   "7-1-define-schema",
			body:       "---\nstatus: in-review\n---\n",
			wantStatus: SpecInReview,
		},
		{
			name:       "story key without a numeric story number",
			file:       "spec-hotfix-login-loop.md",
			storyKey:   "hotfix-login-loop",
			body:       "---\nstatus: draft\n---\n",
			wantStatus: SpecDraft,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			writeSpec(t, base, tt.file, tt.body)

			spec, err := NewReader(base).ReadSpec(tt.storyKey)

			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, spec.Status)
			assert.Equal(t, tt.wantBlocking, spec.Blocking)
			assert.Equal(t, tt.file, filepath.Base(spec.Path))
		})
	}
}

func TestReadSpec_NotFound(t *testing.T) {
	base := t.TempDir()
	writeSpec(t, base, "spec-7-2-other-story.md", "---\nstatus: done\n---\n")

	_, err := NewReader(base).ReadSpec("7-1-define-schema")

	assert.ErrorIs(t, err, ErrSpecNotFound)
}

// A story number prefix must not match a longer number: 7-1 is not 7-10.
func TestReadSpec_DoesNotMatchLongerStoryNumber(t *testing.T) {
	base := t.TempDir()
	writeSpec(t, base, "spec-7-10-other-story.md", "---\nstatus: done\n---\n")

	_, err := NewReader(base).ReadSpec("7-1-define-schema")

	assert.ErrorIs(t, err, ErrSpecNotFound)
}

// bmad-build-auto appends -2, -3 when a spec name is taken, so the newest file
// is the one the run just wrote.
func TestReadSpec_PrefersNewestMatch(t *testing.T) {
	base := t.TempDir()
	old := writeSpec(t, base, "spec-7-1-define-schema.md", "---\nstatus: done\n---\n")
	require.NoError(t, os.Chtimes(old, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour)))
	writeSpec(t, base, "spec-7-1-define-schema-2.md", "---\nstatus: blocked\n---\n")

	spec, err := NewReader(base).ReadSpec("7-1-define-schema")

	require.NoError(t, err)
	assert.Equal(t, SpecBlocked, spec.Status)
	assert.Equal(t, "spec-7-1-define-schema-2.md", filepath.Base(spec.Path))
}

func TestReadSpec_MalformedFrontmatter(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "no frontmatter", body: "# Just a heading\n"},
		{name: "unterminated frontmatter", body: "---\nstatus: done\n"},
		{name: "no status field", body: "---\nbaseline_commit: abc123\n---\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			writeSpec(t, base, "spec-7-1-define-schema.md", tt.body)

			_, err := NewReader(base).ReadSpec("7-1-define-schema")

			require.Error(t, err)
			assert.NotErrorIs(t, err, ErrSpecNotFound)
		})
	}
}
