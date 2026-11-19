package main

import (
	"runtime/debug"
	"strings"
)

// version is set by -X main.version in release builds (scripts/build-release.sh).
var version = "dev"

func currentVersion() string {
	info, _ := debug.ReadBuildInfo()
	return resolveVersion(version, info)
}

// resolveVersion prefers the release version set via -X, falling back to the
// module version Go stamps into binaries built by go install or go build. The
// leading "v" is dropped so every build prints the same format as a release.
func resolveVersion(linked string, info *debug.BuildInfo) string {
	if linked != "dev" {
		return linked
	}
	if info == nil {
		return "dev"
	}
	switch v := info.Main.Version; v {
	case "", "(devel)":
		return "dev"
	default:
		return strings.TrimPrefix(v, "v")
	}
}
