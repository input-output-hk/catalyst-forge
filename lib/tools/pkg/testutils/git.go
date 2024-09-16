package testutils

import (
	"testing"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
)

// InMemRepo represents an in-memory git repository.
type InMemRepo struct {
	Fs       billy.Filesystem
	Repo     *git.Repository
	Worktree *git.Worktree
}

// AddFile creates a file in the repository and adds it to the worktree.
func (r *InMemRepo) AddFile(t *testing.T, path, content string) {
	file, err := r.Fs.Create(path)
	assert.NoError(t, err, "failed to create file")

	_, err = file.Write([]byte(content))
	assert.NoError(t, err, "failed to write to file")

	err = file.Close()
	assert.NoError(t, err, "failed to close file")

	_, err = r.Worktree.Add("example.txt")
	assert.NoError(t, err, "failed to add file")
}

// Commit creates a commit in the repository.
func (r *InMemRepo) Commit(t *testing.T, message string) plumbing.Hash {
	commit, err := r.Worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	assert.NoError(t, err, "failed to commit")

	return commit
}

// Tag creates a tag in the repository.
func (r *InMemRepo) Tag(t *testing.T, commit plumbing.Hash, name, message string) *plumbing.Reference {
	tag, err := r.Repo.CreateTag(name, commit, &git.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
			When:  time.Now(),
		},
		Message: message,
	})
	assert.NoError(t, err, "failed to create tag")

	return tag
}

// NewInMemRepo creates a new in-memory git repository.
func NewInMemRepo(t *testing.T) InMemRepo {
	fs := memfs.New()
	repo, err := git.Init(memory.NewStorage(), fs)
	assert.NoError(t, err, "failed to init repo")

	worktree, err := repo.Worktree()
	assert.NoError(t, err, "failed to get worktree")

	return InMemRepo{
		Fs:       fs,
		Repo:     repo,
		Worktree: worktree,
	}
}
