package pap

import (
	"fmt"
	"strings"
)

// Level represents the PAP activity level of a service or the user's configured limit.
// Higher numeric values indicate more active (potentially target-facing) operations.
type Level int

const (
	// RED — non-detectable operations (offline lookups, local databases).
	// Most restrictive when used as a limit.
	RED Level = iota
	// AMBER — detectable but not directly attributable to the target (3rd-party APIs).
	AMBER
	// GREEN — active operations with direct interaction with the target
	// (e.g. direct DNS resolution, port scanning, HTTP crawling).
	GREEN
	// WHITE — unrestricted; all services are permitted.
	// Most permissive when used as a limit.
	WHITE
)

// Parse converts a case-insensitive string ("red", "amber", "green", "white") to a Level.
func Parse(s string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "red":
		return RED, nil
	case "amber":
		return AMBER, nil
	case "green":
		return GREEN, nil
	case "white":
		return WHITE, nil
	default:
		return WHITE, fmt.Errorf("unknown PAP level %q: must be one of red, amber, green, white", s)
	}
}

// String returns the lowercase string representation of a Level.
func (l Level) String() string {
	switch l {
	case RED:
		return "red"
	case AMBER:
		return "amber"
	case GREEN:
		return "green"
	case WHITE:
		return "white"
	default:
		return fmt.Sprintf("level(%d)", int(l))
	}
}

// MustParse is like Parse but panics if s is not a valid PAP level.
// Only call this when the input has already been validated (e.g., after buildDeps).
func MustParse(s string) Level {
	level, err := Parse(s)
	if err != nil {
		panic(fmt.Sprintf("pap.MustParse: %v", err))
	}
	return level
}

// Allows reports whether the user-specified limit permits a service at the given level.
// A service is allowed when its level does not exceed the limit.
//
// Examples:
//
//	Allows(WHITE, GREEN) == true   // unrestricted limit allows everything
//	Allows(AMBER, GREEN) == false  // AMBER limit blocks direct-interaction services
//	Allows(GREEN, AMBER) == true   // GREEN limit allows 3rd-party API services
func Allows(limit, service Level) bool {
	return service <= limit
}
