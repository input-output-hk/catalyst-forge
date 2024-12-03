package git

import (
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker/mocks"
	"github.com/stretchr/testify/assert"
)

func TestAddAll(t *testing.T) {
	r := testutils.NewInMemRepo(t)

	r.AddFile(t, "test.txt", "test")
	r.Commit(t, "Initial commit")

	r.CreateFile(t, "test2.txt", "test")

	err := AddAll(r.Repo)
	assert.NoError(t, err)

	status, err := r.Worktree.Status()
	assert.NoError(t, err)
	assert.NotEqual(t, status.File("test2.txt").Staging, git.Untracked)
}

func TestBranchExists(t *testing.T) {
	r := testutils.NewInMemRepo(t)

	r.AddFile(t, "test.txt", "test")
	r.Commit(t, "Initial commit")

	exists, err := BranchExists(r.Repo, "master")
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = BranchExists(r.Repo, "test-branch")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCheckoutBranch(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		opts     []GitCheckoutOption
		dirty    bool
		validate func(t *testing.T, r *git.Repository, err error)
	}{
		{
			name:   "simple",
			branch: "test-branch",
			opts:   []GitCheckoutOption{GitCheckoutCreate()},
			validate: func(t *testing.T, r *git.Repository, err error) {
				assert.NoError(t, err)

				head, err := r.Head()
				assert.NoError(t, err)
				assert.Equal(t, plumbing.NewBranchReferenceName("test-branch"), head.Name())
			},
		},
		{
			name:   "force clean",
			branch: "test-branch",
			opts:   []GitCheckoutOption{GitCheckoutCreate(), GitCheckoutForceClean()},
			dirty:  true,
			validate: func(t *testing.T, r *git.Repository, err error) {
				assert.Error(t, err)
			},
		},
		{
			name:   "no create",
			branch: "test-branch",
			validate: func(t *testing.T, r *git.Repository, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := testutils.NewInMemRepo(t)

			r.AddFile(t, "test.txt", "test")
			r.Commit(t, "Initial commit")

			if tt.dirty {
				r.AddFile(t, "test2.txt", "test")
			}

			err := CheckoutBranch(r.Repo, tt.branch, tt.opts...)
			tt.validate(t, r.Repo, err)
		})
	}
}

func TestCommitAll(t *testing.T) {
	r := testutils.NewInMemRepo(t)

	r.AddFile(t, "test.txt", "test")
	r.Commit(t, "Initial commit")

	r.CreateFile(t, "test2.txt", "test")

	err := CommitAll(r.Repo, "Test commit", &git.CommitOptions{})
	assert.NoError(t, err)

	status, err := r.Worktree.Status()
	assert.NoError(t, err)
	assert.True(t, status.IsClean())

	head, err := r.Repo.Head()
	assert.NoError(t, err)
	commit, err := r.Repo.CommitObject(head.Hash())
	assert.NoError(t, err)
	assert.Equal(t, "Test commit", commit.Message)
}

func TestFindGitRoot(t *testing.T) {
	tests := []struct {
		name      string
		start     string
		dirs      []string
		want      string
		expectErr bool
	}{
		{
			name:  "simple",
			start: "/tmp/test1/test2/test3",
			dirs: []string{
				"/tmp/test1/test2",
				"/tmp/test1",
				"/tmp/.git",
				"/",
			},
			want:      "/tmp",
			expectErr: false,
		},
		{
			name:  "no git root",
			start: "/tmp/test1/test2/test3",
			dirs: []string{
				"/tmp/test1/test2",
				"/tmp/test1",
				"/",
			},
			want:      "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lastPath string
			w := &mocks.ReverseWalkerMock{
				WalkFunc: func(startPath string, endPath string, callback walker.WalkerCallback) error {
					for _, dir := range tt.dirs {
						err := callback(dir, walker.FileTypeDir, func() (walker.FileSeeker, error) {
							return nil, nil
						})

						if errors.Is(err, io.EOF) {
							lastPath = dir
							return nil
						} else if err != nil {
							return err
						}
					}
					return nil
				},
			}

			got, err := FindGitRoot(tt.start, w)
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}
			assert.Equal(t, tt.want, got)
			assert.Equal(t, lastPath, filepath.Join(tt.want, ".git"))
		})
	}
}

func TestGetCurrentBranch(t *testing.T) {
	r := testutils.NewInMemRepo(t)

	r.AddFile(t, "test.txt", "test")
	r.Commit(t, "Initial commit")

	r.Worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("test-branch"),
		Create: true,
	})

	got, err := GetCurrentBranch(r.Repo)
	assert.NoError(t, err)
	assert.Equal(t, "test-branch", got)

	head, err := r.Repo.Head()
	assert.NoError(t, err)
	ref := r.Tag(t, head.Hash(), "test-tag", "Test tag")
	r.Worktree.Checkout(&git.CheckoutOptions{
		Hash: ref.Hash(),
	})

	_, err = GetCurrentBranch(r.Repo)
	assert.Error(t, err)
}

func TestHasChanges(t *testing.T) {
	r := testutils.NewInMemRepo(t)

	r.AddFile(t, "test.txt", "test")
	r.Commit(t, "Initial commit")

	hasChanges, err := HasChanges(r.Repo)
	assert.NoError(t, err)
	assert.False(t, hasChanges)

	r.CreateFile(t, "test2.txt", "test")

	hasChanges, err = HasChanges(r.Repo)
	assert.NoError(t, err)
	assert.True(t, hasChanges)
}
