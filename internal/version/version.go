// Package version holds the build's version string, stamped at link time:
//
//	go build -ldflags "-X github.com/floholz/matchowl/internal/version.Version=v1.0.0-alpha.1"
//
// The Makefile derives it from `git describe`, the Docker build from the
// VERSION build-arg CI passes (the release tag). "dev" when unstamped.
package version

var (
	// Version is the release tag (v1.0.0-alpha.1) or a git describe string.
	Version = "dev"
	// Revision is the commit the build came from, when known.
	Revision = ""
)

// Short is the version without a leading "v".
func Short() string {
	if len(Version) > 1 && Version[0] == 'v' {
		return Version[1:]
	}
	return Version
}
