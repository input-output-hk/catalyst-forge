package testutils

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err, "failed to create file")

	_, err = file.Write([]byte(content))
	require.NoError(t, err, "failed to write to file")

	err = file.Close()
	require.NoError(t, err, "failed to close file")

	_, err = r.Worktree.Add(path)
	require.NoError(t, err, "failed to add file")
}

// AddExistingFile adds an existing file to the worktree.
func (r *InMemRepo) AddExistingFile(t *testing.T, path string) {
	_, err := r.Worktree.Add(path)
	require.NoError(t, err, "failed to add file")
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
	require.NoError(t, err, "failed to commit")

	return commit
}

// CreateFile creates a file in the repository.
func (r *InMemRepo) CreateFile(t *testing.T, path, content string) {
	file, err := r.Fs.Create(path)
	require.NoError(t, err, "failed to create file")

	_, err = file.Write([]byte(content))
	require.NoError(t, err, "failed to write to file")

	err = file.Close()
	require.NoError(t, err, "failed to close file")
}

// Exists checks if a file exists in the repository.
func (r *InMemRepo) Exists(t *testing.T, path string) bool {
	_, err := r.Fs.Stat(path)
	if err == os.ErrNotExist {
		return false
	} else if err != nil {
		t.Fatalf("failed to check if file exists: %v", err)
	}

	return true
}

// MkdirAll creates a directory in the repository.
func (r *InMemRepo) MkdirAll(t *testing.T, path string) {
	require.NoError(t, r.Fs.MkdirAll(path, 0755), "failed to create directory")
}

func (r *InMemRepo) ReadFile(t *testing.T, path string) []byte {
	file, err := r.Fs.Open(path)
	require.NoError(t, err, "failed to open file")

	contents, err := io.ReadAll(file)
	require.NoError(t, err, "failed to read file")
	require.NoError(t, file.Close(), "failed to close file")

	return contents
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
	require.NoError(t, err, "failed to create tag")

	return tag
}

// NewInMemRepo creates a new in-memory git repository.
func NewInMemRepo(t *testing.T) InMemRepo {
	fs := memfs.New()
	repo, err := git.Init(memory.NewStorage(), fs)
	assert.NoError(t, err, "failed to init repo")

	worktree, err := repo.Worktree()
	require.NoError(t, err, "failed to get worktree")

	return InMemRepo{
		Fs:       fs,
		Repo:     repo,
		Worktree: worktree,
	}
}
