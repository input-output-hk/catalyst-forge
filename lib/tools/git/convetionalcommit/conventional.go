package convetionalcommit

import (
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/leodido/go-conventionalcommits"
	"github.com/leodido/go-conventionalcommits/parser"
)

// ConventionalCommit is a helper type for working with conventional commits.
// It wraps the go-conventionalcommits package and provides a simpler interface.
type ConventionalCommit struct {
	commit *object.Commit
	parsed conventionalcommits.Message
	err    error
}

// New creates a new ConventionalCommit instance from a git commit object.
func New(commit *object.Commit) *ConventionalCommit {
	cc := &ConventionalCommit{
		commit: commit,
	}

	// Parse the commit message
	m := parser.NewMachine(conventionalcommits.WithTypes(conventionalcommits.TypesConventional))
	parsed, err := m.Parse([]byte(commit.Message))

	cc.parsed = parsed
	cc.err = err

	return cc
}

// IsConventional returns true if the commit message follows the conventional commits specification.
func (cc *ConventionalCommit) IsConventional() bool {
	return cc.err == nil && cc.parsed != nil
}

// Parse returns the parsed conventional commit data and any parsing error.
// If the commit is not conventional, the error will be non-nil.
func (cc *ConventionalCommit) Parse() (conventionalcommits.Message, error) {
	return cc.parsed, cc.err
}

// GetCommit returns the underlying git commit object.
func (cc *ConventionalCommit) GetCommit() *object.Commit {
	return cc.commit
}

// FilterConventionalCommits takes a slice of git commits and returns only those that
// follow the conventional commits specification. Commits that fail parsing are ignored.
func FilterConventionalCommits(commits []*object.Commit) []*ConventionalCommit {
	var conventionalCommits []*ConventionalCommit

	for _, commit := range commits {
		cc := New(commit)
		if cc.IsConventional() {
			conventionalCommits = append(conventionalCommits, cc)
		}
	}

	return conventionalCommits
}
