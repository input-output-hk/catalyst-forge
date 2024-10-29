package argo

import "os"

// InArgo returns true if the code is running in an Argo CD environment.
func InArgo() bool {
	if _, ok := os.LookupEnv("ARGOCD_APP_NAME"); ok {
		return true
	}

	return false
}
