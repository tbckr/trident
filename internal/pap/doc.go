// Package pap implements the Permissible Actions Protocol (PAP) classification
// system for controlling how actively a service interacts with or exposes a target.
//
// Levels in ascending order of activity:
//
//	RED   — non-detectable (offline / local only)
//	AMBER — detectable but not directly attributable (3rd-party APIs)
//	GREEN — active, direct interaction with the target (DNS, port scan, HTTP crawl)
//	WHITE — unrestricted
//
// The user sets a --pap-limit; a service is blocked when its level exceeds the limit.
package pap
