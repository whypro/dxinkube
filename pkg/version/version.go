package version

import (
	"fmt"
	"regexp"
	"runtime"

	apimachineryversion "k8s.io/apimachinery/pkg/version"
)

var (
	versionRegexp = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)`)
)

// Get returns the overall codebase version. It's for detecting
// what code a binary was built from.
func Get() apimachineryversion.Info {
	gitMajor := ""
	gitMinor := ""
	subMatches := versionRegexp.FindStringSubmatch(gitVersion)
	if len(subMatches) > 3 {
		gitMajor = subMatches[1]
		gitMinor = subMatches[2]
	}
	// These variables typically come from -ldflags settings and in
	// their absence fallback to the settings in pkg/version/base.go
	return apimachineryversion.Info{
		Major:        gitMajor,
		Minor:        gitMinor,
		GitVersion:   gitVersion,
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		BuildDate:    buildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
