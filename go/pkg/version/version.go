package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the current version of the application
	// This will be overridden by ldflags during build
	Version = "dev"

	// GitCommit is the git commit hash
	// This will be overridden by ldflags during build
	GitCommit = "unknown"

	// BuildDate is the build date
	// This will be overridden by ldflags during build
	BuildDate = "unknown"
)

// BuildInfo represents build information
type BuildInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// GetBuildInfo returns the build information
func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string
func (b BuildInfo) String() string {
	return fmt.Sprintf("Version: %s\nGit Commit: %s\nBuild Date: %s\nGo Version: %s\nPlatform: %s",
		b.Version, b.GitCommit, b.BuildDate, b.GoVersion, b.Platform)
}