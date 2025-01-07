package github

import "os"

// InGithubActions returns true if the process is running in a GitHub Actions
// environment.
func InGithubActions() bool {
	if _, ok := os.LookupEnv("GITHUB_ACTIONS"); ok {
		return true
	}

	return false
}
