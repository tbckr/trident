// Package version holds build-time version variables injected via ldflags.
// When ldflags are not set (e.g. go install), an init function reads
// runtime/debug.BuildInfo as a fallback so the binary reports the correct
// module version and VCS metadata instead of the default placeholder values.
package version
