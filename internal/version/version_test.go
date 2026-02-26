package version

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func saveRestore(t *testing.T) {
	t.Helper()
	origVersion, origCommit, origDate := Version, Commit, Date
	t.Cleanup(func() {
		Version = origVersion
		Commit = origCommit
		Date = origDate
	})
}

func bi(mainVersion string, settings map[string]string) *debug.BuildInfo {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: mainVersion},
	}
	for k, v := range settings {
		info.Settings = append(info.Settings, debug.BuildSetting{Key: k, Value: v})
	}
	return info
}

func TestApplyBuildInfo(t *testing.T) {
	tests := []struct {
		name        string
		setup       func()
		buildInfo   *debug.BuildInfo
		wantVersion string
		wantCommit  string
		wantDate    string
	}{
		{
			name: "ldflags already set — no override",
			setup: func() {
				Version = "1.2.3"
				Commit = "abc1234"
				Date = "2025-01-01T00:00:00Z"
			},
			buildInfo:   bi("v0.5.0", map[string]string{"vcs.revision": "deadbeefcafe", "vcs.time": "2024-06-01T00:00:00Z"}),
			wantVersion: "1.2.3",
			wantCommit:  "abc1234",
			wantDate:    "2025-01-01T00:00:00Z",
		},
		{
			name:        "go install @latest — module version only",
			buildInfo:   bi("v0.5.0", nil),
			wantVersion: "0.5.0",
			wantCommit:  "none",
			wantDate:    "unknown",
		},
		{
			name:        "local build — devel version with VCS settings",
			buildInfo:   bi("(devel)", map[string]string{"vcs.revision": "deadbeefcafe123", "vcs.time": "2024-06-01T12:00:00Z"}),
			wantVersion: "dev",
			wantCommit:  "deadbee",
			wantDate:    "2024-06-01T12:00:00Z",
		},
		{
			name:        "full VCS info — module version and VCS",
			buildInfo:   bi("v1.0.0", map[string]string{"vcs.revision": "aabbccdd1122334", "vcs.time": "2025-03-15T08:00:00Z"}),
			wantVersion: "1.0.0",
			wantCommit:  "aabbccd",
			wantDate:    "2025-03-15T08:00:00Z",
		},
		{
			name:        "empty BuildInfo — no changes",
			buildInfo:   &debug.BuildInfo{},
			wantVersion: "dev",
			wantCommit:  "none",
			wantDate:    "unknown",
		},
		{
			name:        "short revision — not truncated",
			buildInfo:   bi("(devel)", map[string]string{"vcs.revision": "abc"}),
			wantVersion: "dev",
			wantCommit:  "abc",
			wantDate:    "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveRestore(t)
			Version = "dev"
			Commit = "none"
			Date = "unknown"
			if tt.setup != nil {
				tt.setup()
			}

			applyBuildInfo(tt.buildInfo)

			assert.Equal(t, tt.wantVersion, Version, "Version")
			assert.Equal(t, tt.wantCommit, Commit, "Commit")
			assert.Equal(t, tt.wantDate, Date, "Date")
		})
	}
}
