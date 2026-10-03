package main

import (
	"runtime/debug"
	"strings"
)

// resolveVersion returns the version and commit to report. GoReleaser sets
// both with -ldflags, but go install builds from the module without them,
// which left every installed binary reporting "dev". The build information
// Go embeds fills the gap: the module version for go install pkg@version,
// and the VCS revision for a build from a checkout.
func resolveVersion(version, commit string, info *debug.BuildInfo, ok bool) (string, string) {
	if !ok || info == nil {
		return version, commit
	}

	if version == "dev" {
		// "(devel)" is what a build from the main module reports, which
		// says no more than "dev" does.
		if v := info.Main.Version; v != "" && v != "(devel)" {
			version = strings.TrimPrefix(v, "v")
		}
	}

	if commit == "unknown" {
		var revision string
		modified := false
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				revision = s.Value
			case "vcs.modified":
				modified = s.Value == "true"
			}
		}
		if revision != "" {
			commit = revision
			if modified {
				commit += "-dirty"
			}
		}
	}

	return version, commit
}
