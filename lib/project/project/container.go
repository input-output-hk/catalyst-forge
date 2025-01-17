package project

import (
	"fmt"
	"regexp"
	"strings"
)

// GenerateContainerName generates the container name for the project.
// If the name is not provided, the project name is used.
func GenerateContainerName(p *Project, name string, registry string) string {
	var n string
	if name == "" {
		n = p.Name
	} else {
		n = name
	}

	var repo string
	if strings.Contains(p.Blueprint.Global.Repo.Name, "/") {
		repo = strings.Split(p.Blueprint.Global.Repo.Name, "/")[1]
	} else {
		repo = p.Blueprint.Global.Repo.Name
	}

	var container string
	if registry != "" {
		if isGHCRRegistry(registry) {
			container = fmt.Sprintf("%s/%s", strings.TrimSuffix(registry, "/"), n)
		} else {
			container = fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(registry, "/"), repo, n)
		}
	} else {
		container = n
	}

	return container
}

// isGHCRRegistry checks if the registry is a GHCR registry.
func isGHCRRegistry(registry string) bool {
	return regexp.MustCompile(`^ghcr\.io/[a-zA-Z0-9](?:-?[a-zA-Z0-9])*`).MatchString(registry)
}
