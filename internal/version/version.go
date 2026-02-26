package version

import (
	"runtime/debug"
	"strings"
)

// Build-time variables injected via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func init() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	applyBuildInfo(bi)
}

// applyBuildInfo overwrites package vars from bi only when they still hold
// their default (ldflags-unset) values. ldflags always win.
func applyBuildInfo(bi *debug.BuildInfo) {
	if Version == "dev" {
		v := bi.Main.Version
		if v != "" && v != "(devel)" {
			Version = strings.TrimPrefix(v, "v")
		}
	}

	var revision, vcsTime string
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.time":
			vcsTime = s.Value
		}
	}

	if Commit == "none" && revision != "" {
		if len(revision) > 7 {
			revision = revision[:7]
		}
		Commit = revision
	}

	if Date == "unknown" && vcsTime != "" {
		Date = vcsTime
	}
}
