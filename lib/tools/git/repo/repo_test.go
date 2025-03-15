package repo

import (
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
	bfs "github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	repoPath = "/repo"
)

func TestGitRepoClone(t *testing.T) {
	repo := newRepo(t)

	var opts *gg.CloneOptions
	r := GitRepo{
		gfs:    repo.gfs,
		wfs:    repo.wfs,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		remote: &mocks.GitRemoteInteractorMock{
			CloneFunc: func(s storage.Storer, worktree billy.Filesystem, o *gg.CloneOptions) (*gg.Repository, error) {
				opts = o
				return repo.repo, nil
			},
		},
	}

	err := r.Clone("test.com", "master")
	require.NoError(t, err)

	assert.Equal(t, opts.URL, "test.com")
	assert.Equal(t, opts.ReferenceName, plumbing.ReferenceName("refs/heads/master"))

	head, err := r.raw.Head()
	require.NoError(t, err)

	commit, err := r.GetCommit(head.Hash())
	require.NoError(t, err)
	assert.Equal(t, commit.Message, "test")
}

func TestGitRepoCommit(t *testing.T) {
	t.Run("succcess", func(t *testing.T) {
		repo := newGitRepo(t)
		err := repo.wfs.WriteFile("file.txt", []byte("test"), 0644)
		require.NoError(t, err)

		err = repo.StageFile("file.txt")
		require.NoError(t, err)

		hash, err := repo.Commit("test")
		require.NoError(t, err)

		commit, err := repo.GetCommit(hash)
		require.NoError(t, err)
		assert.Equal(t, commit.Message, "test")
	})
}

func TestGitRepoExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		repo := newGitRepo(t)
		err := repo.wfs.WriteFile("file.txt", []byte("test"), 0644)
		require.NoError(t, err)

		exists, err := repo.Exists("file.txt")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("does not exist", func(t *testing.T) {
		repo := newGitRepo(t)

		exists, err := repo.Exists("file.txt")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestGitRepoGetCurrentBranch(t *testing.T) {
	repo := newGitRepo(t)

	branch, err := repo.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, branch, "master")
}

func TestGitRepoGetCurrentTag(t *testing.T) {
	t.Run("tag exists", func(t *testing.T) {
		repo := newGitRepo(t)

		head, err := repo.Head()
		require.NoError(t, err)
		repo.NewTag(head.Hash(), "test", "test")

		tag, err := repo.GetCurrentTag()
		require.NoError(t, err)
		assert.Equal(t, tag, "test")
	})

	t.Run("tag does not exist", func(t *testing.T) {
		repo := newGitRepo(t)

		tag, err := repo.GetCurrentTag()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tag not found")
		assert.Equal(t, tag, "")
	})
}

func TestGitRepoHasChanges(t *testing.T) {
	t.Run("has changes", func(t *testing.T) {
		repo := newGitRepo(t)

		err := repo.wfs.WriteFile("file.txt", []byte("test"), 0644)
		require.NoError(t, err)

		changes, err := repo.HasChanges()
		require.NoError(t, err)
		assert.True(t, changes)
	})

	t.Run("no changes", func(t *testing.T) {
		repo := newGitRepo(t)

		changes, err := repo.HasChanges()
		require.NoError(t, err)
		assert.False(t, changes)
	})
}

func TestGitRepoInit(t *testing.T) {
	gfs := bfs.NewInMemoryFs()
	wfs := bfs.NewInMemoryFs()

	require.NoError(t, wfs.WriteFile("file.txt", []byte("test"), 0644))

	r := GitRepo{
		gfs:    gfs,
		wfs:    wfs,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	require.NoError(t, r.Init())

	wt, err := r.raw.Worktree()
	require.NoError(t, err)
	st, err := wt.Status()
	require.NoError(t, err)
	assert.Equal(t, st.File("file.txt").Staging, gg.Untracked)
}

func TestGitRepoNewBranch(t *testing.T) {
	repo := newGitRepo(t)

	err := repo.NewBranch("test")
	require.NoError(t, err)

	head, err := repo.raw.Head()
	require.NoError(t, err)
	assert.Equal(t, head.Name().String(), "refs/heads/test")
}

func TestGitRepoNewTag(t *testing.T) {
	repo := newGitRepo(t)

	require.NoError(t, repo.wfs.WriteFile("file.txt", []byte("test"), 0644))

	require.NoError(t, repo.StageFile("file.txt"))

	hash, err := repo.Commit("test")
	require.NoError(t, err)

	tag, err := repo.NewTag(hash, "test", "test")
	require.NoError(t, err)

	assert.Equal(t, tag.Name().String(), "refs/tags/test")
}

func TestGitRepoOpen(t *testing.T) {
	repo := newRepo(t)

	r := GitRepo{
		gfs:    repo.gfs,
		wfs:    repo.wfs,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	err := r.Open()
	require.NoError(t, err)

	head, err := r.raw.Head()
	require.NoError(t, err)

	commit, err := r.GetCommit(head.Hash())
	require.NoError(t, err)
	assert.Equal(t, commit.Message, "test")
}

func TestGitRepoReadFile(t *testing.T) {
	repo := newGitRepo(t)
	c, err := repo.ReadFile("test.txt")
	require.NoError(t, err)
	assert.Equal(t, c, []byte("test"))
}

func TestGitRepoReadDir(t *testing.T) {
	repo := newGitRepo(t)
	require.NoError(t, repo.wfs.MkdirAll("dir", 0755))
	require.NoError(t, repo.wfs.WriteFile("dir/file.txt", []byte("test"), 0644))

	files, err := repo.ReadDir("dir")
	require.NoError(t, err)
	assert.Equal(t, files[0].Name(), "file.txt")
}

func TestGitRepoRemoveFile(t *testing.T) {
	repo := newGitRepo(t)

	require.NoError(t, repo.RemoveFile("test.txt"))
	require.NoError(t, repo.StageFile("test.txt"))

	status, err := repo.worktree.Status()
	require.NoError(t, err)
	require.Equal(t, status.File("test.txt").Staging, gg.Deleted)
}

func TestGitRepoPush(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newGitRepo(t)

		var opts *gg.PushOptions
		auth := &http.BasicAuth{
			Username: "forge",
			Password: "test",
		}

		repo.auth = auth
		repo.remote = &mocks.GitRemoteInteractorMock{
			PushFunc: func(r *gg.Repository, o *gg.PushOptions) error {
				opts = o
				return nil
			},
		}

		err := repo.Push()
		assert.NoError(t, err)
		assert.Equal(t, opts.Auth, auth)
	})

	t.Run("error", func(t *testing.T) {
		repo := newGitRepo(t)

		repo.remote = &mocks.GitRemoteInteractorMock{
			PushFunc: func(r *gg.Repository, o *gg.PushOptions) error {
				return fmt.Errorf("failed to push")
			},
		}

		err := repo.Push()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to push")
	})
}

func TestGitRepoStageFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newGitRepo(t)
		require.NoError(t, repo.wfs.WriteFile("file.txt", []byte("test"), 0644))

		_, err := repo.worktree.Add("file.txt")
		require.NoError(t, err, "failed to add file")

		status, err := repo.worktree.Status()
		require.NoError(t, err)

		assert.Contains(t, status, "file.txt")
	})

	t.Run("file missing", func(t *testing.T) {
		repo := newGitRepo(t)

		err := repo.StageFile("file.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "entry not found")
	})
}

func TestGitRepoWriteFile(t *testing.T) {
	repo := newGitRepo(t)

	require.NoError(t, repo.WriteFile("file.txt", []byte("test")))

	status, err := repo.worktree.Status()
	require.NoError(t, err)
	assert.Contains(t, status, "file.txt")
}

type testRepo struct {
	repo     *gg.Repository
	gfs      *bfs.BillyFs
	wfs      *bfs.BillyFs
	worktree *gg.Worktree
}

func newRepo(t *testing.T) testRepo {
	gfs := memfs.New()
	wfs := memfs.New()
	storage := filesystem.NewStorage(gfs, cache.NewObjectLRUDefault())

	repo, err := gg.Init(storage, wfs)
	require.NoError(t, err, "failed to init repo")

	worktree, err := repo.Worktree()
	require.NoError(t, err, "failed to get worktree")

	f, err := wfs.Create("test.txt")
	require.NoError(t, err, "failed to create file")
	defer f.Close()
	_, err = f.Write([]byte("test"))
	require.NoError(t, err, "failed to write to file")

	_, err = worktree.Add("test.txt")
	require.NoError(t, err, "failed to add file")

	status, err := worktree.Status()
	require.NoError(t, err)
	assert.False(t, status.IsClean())
	assert.Contains(t, status, "test.txt")

	_, err = worktree.Commit("test", &gg.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err, "failed to commit")

	return testRepo{
		gfs:      bfs.NewFs(gfs),
		repo:     repo,
		wfs:      bfs.NewFs(wfs),
		worktree: worktree,
	}
}

func newGitRepo(t *testing.T) GitRepo {
	repo := newRepo(t)

	return GitRepo{
		gfs:      repo.gfs,
		wfs:      repo.wfs,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		raw:      repo.repo,
		worktree: repo.worktree,
	}
}
