package convetionalcommit

import (
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/leodido/go-conventionalcommits"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConventionalCommit_IsConventional(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{
			name:    "valid conventional commit",
			message: "feat: add new feature",
			want:    true,
		},
		{
			name:    "valid conventional commit with scope",
			message: "fix(parser): resolve parsing issue",
			want:    true,
		},
		{
			name:    "valid conventional commit with breaking change",
			message: "feat!: breaking change\n\nBREAKING CHANGE: this is a breaking change",
			want:    true,
		},
		{
			name:    "invalid conventional commit",
			message: "just a regular commit message",
			want:    false,
		},
		{
			name:    "empty message",
			message: "",
			want:    false,
		},
		{
			name:    "missing colon",
			message: "feat add new feature",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commit := &object.Commit{
				Hash:    plumbing.NewHash("1234567890abcdef"),
				Message: tt.message,
				Author: object.Signature{
					Name:  "Test Author",
					Email: "test@example.com",
					When:  time.Now(),
				},
			}

			cc := New(commit)
			got := cc.IsConventional()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConventionalCommit_Parse(t *testing.T) {
	t.Run("valid conventional commit", func(t *testing.T) {
		commit := &object.Commit{
			Hash:    plumbing.NewHash("1234567890abcdef"),
			Message: "feat(api): add new endpoint\n\nThis adds a new REST endpoint for user management.",
			Author: object.Signature{
				Name:  "Test Author",
				Email: "test@example.com",
				When:  time.Now(),
			},
		}

		cc := New(commit)
		parsed, err := cc.Parse()

		require.NoError(t, err)
		require.NotNil(t, parsed)

		// Type assert to get the concrete type
		if conventional, ok := parsed.(*conventionalcommits.ConventionalCommit); ok {
			assert.Equal(t, "feat", conventional.Type)
			if conventional.Scope != nil {
				assert.Equal(t, "api", *conventional.Scope)
			}
			assert.Equal(t, "add new endpoint", conventional.Description)
			if conventional.Body != nil {
				assert.Contains(t, *conventional.Body, "This adds a new REST endpoint")
			}
		} else {
			t.Fatal("expected *conventionalcommits.ConventionalCommit type")
		}
	})

	t.Run("invalid conventional commit", func(t *testing.T) {
		commit := &object.Commit{
			Hash:    plumbing.NewHash("1234567890abcdef"),
			Message: "just a regular commit message",
			Author: object.Signature{
				Name:  "Test Author",
				Email: "test@example.com",
				When:  time.Now(),
			},
		}

		cc := New(commit)
		parsed, err := cc.Parse()

		assert.Error(t, err)
		assert.Nil(t, parsed)
	})
}

func TestConventionalCommit_GetCommit(t *testing.T) {
	commit := &object.Commit{
		Hash:    plumbing.NewHash("1234567890abcdef"),
		Message: "feat: add new feature",
		Author: object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
	}

	cc := New(commit)
	got := cc.GetCommit()

	assert.Equal(t, commit, got)
}

func TestFilterConventionalCommits(t *testing.T) {
	// Create test commits
	commits := []*object.Commit{
		{
			Hash:    plumbing.NewHash("1111111111111111"),
			Message: "feat: add new feature",
			Author: object.Signature{
				Name:  "Test Author",
				Email: "test@example.com",
				When:  time.Now(),
			},
		},
		{
			Hash:    plumbing.NewHash("2222222222222222"),
			Message: "fix(parser): resolve parsing issue",
			Author: object.Signature{
				Name:  "Test Author",
				Email: "test@example.com",
				When:  time.Now(),
			},
		},
		{
			Hash:    plumbing.NewHash("3333333333333333"),
			Message: "just a regular commit message",
			Author: object.Signature{
				Name:  "Test Author",
				Email: "test@example.com",
				When:  time.Now(),
			},
		},
		{
			Hash:    plumbing.NewHash("4444444444444444"),
			Message: "docs: update documentation",
			Author: object.Signature{
				Name:  "Test Author",
				Email: "test@example.com",
				When:  time.Now(),
			},
		},
		{
			Hash:    plumbing.NewHash("5555555555555555"),
			Message: "another non-conventional commit",
			Author: object.Signature{
				Name:  "Test Author",
				Email: "test@example.com",
				When:  time.Now(),
			},
		},
	}

	// Filter conventional commits
	result := FilterConventionalCommits(commits)

	// Should return 3 conventional commits
	assert.Len(t, result, 3)

	// Verify the returned commits are conventional
	for _, cc := range result {
		assert.True(t, cc.IsConventional())
	}

	// Verify we got the expected commits
	expectedMessages := []string{
		"feat: add new feature",
		"fix(parser): resolve parsing issue",
		"docs: update documentation",
	}

	resultMessages := make([]string, len(result))
	for i, cc := range result {
		resultMessages[i] = cc.GetCommit().Message
	}

	// Sort both slices to ensure order doesn't matter
	assert.ElementsMatch(t, expectedMessages, resultMessages)
}

func TestFilterConventionalCommits_EmptySlice(t *testing.T) {
	result := FilterConventionalCommits([]*object.Commit{})
	assert.Empty(t, result)
}

func TestFilterConventionalCommits_NoConventional(t *testing.T) {
	commits := []*object.Commit{
		{
			Hash:    plumbing.NewHash("1111111111111111"),
			Message: "just a regular commit message",
			Author: object.Signature{
				Name:  "Test Author",
				Email: "test@example.com",
				When:  time.Now(),
			},
		},
		{
			Hash:    plumbing.NewHash("2222222222222222"),
			Message: "another non-conventional commit",
			Author: object.Signature{
				Name:  "Test Author",
				Email: "test@example.com",
				When:  time.Now(),
			},
		},
	}

	result := FilterConventionalCommits(commits)
	assert.Empty(t, result)
}
