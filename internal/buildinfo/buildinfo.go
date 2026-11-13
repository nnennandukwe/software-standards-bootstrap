// Package buildinfo holds release identity injected by the build.
package buildinfo

// Version and Commit default to local build identity.
var (
	Version = "dev"
	Commit  = "unknown"
)
