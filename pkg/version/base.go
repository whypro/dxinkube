package version

// Base version information.
//
// This is the fallback data used when version information from git is not
// provided via go ldflags. It provides an approximation of the Monos
// version for ad-hoc builds (e.g. `go build`) that cannot get the version
// information from git.
//
// If you are looking at these fields in the git tree, they look
// strange. They are modified on the fly by the build process. The
// in-tree values are dummy values used for "git archive", which also
// works for GitHub tar downloads.
//
// When releasing a new Monos version, this file is updated by
// build/mark_new_version.sh to reflect the new version, and then a
// git annotated tag (using format vX.Y where X == Major version and Y
// == Minor version) is created to point to the commit that updates
// pkg/version/base.go
var (
	// Update this whenever making a new release.
	// The version is of the format Major.Minor.Patch[-Prerelease][+BuildMetadata]
	//
	// Increment major number for new feature additions and behavioral changes.
	// Increment minor number for bug fixes and performance enhancements.
	// Increment patch number for critical fixes to existing releases.
	gitVersion          = "v0.2.5+$Format:%h$"
	gitCommit    string = "$Format:%H$"    // sha1 from git, output of $(git rev-parse HEAD)
	gitTreeState string = "not a git tree" // state of git tree, either "clean" or "dirty"

	buildDate string = "1970-01-01T00:00:00Z" // build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
)
