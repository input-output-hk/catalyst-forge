package earthly

import (
	"fmt"
	"runtime"
)

// getBuildPlatform returns the current build platform for Earthly.
func GetBuildPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		return fmt.Sprintf("linux/%s", runtime.GOARCH)
	case "windows":
		return fmt.Sprintf("linux/%s", runtime.GOARCH)
	default:
		return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	}
}
