package buildinfo

type BuildInfo struct {
	GitTag           string
	GitCommit        string
	BuildDate        string
	BuildTags        string
	ClientIdentifier string // main program name: eg: 'aquabootnode'
}

var binfo BuildInfo

// Set once by main package
func SetBuildInfo(info BuildInfo) {
	binfo = info
}

// GetBuildInfo returns the build information or a default value if not set
func GetBuildInfo() BuildInfo {
	if binfo == (BuildInfo{}) {
		return BuildInfo{
			GitTag:           "v0.0.1-unknown",
			GitCommit:        "a0b0c0d0",
			BuildDate:        "unknown", // TODO: this one should error somewhere
			BuildTags:        "unknown",
			ClientIdentifier: "unknown",
		}
	}
	return binfo
}
