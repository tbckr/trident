package pap

import (
	"fmt"
	"strings"
)

type Level int

const (
	Red Level = iota
	Amber
	Green
	White
)

func (l Level) String() string {
	switch l {
	case Red:
		return "red"
	case Amber:
		return "amber"
	case Green:
		return "green"
	case White:
		return "white"
	default:
		return "unknown"
	}
}

func ParseLevel(lvl string) (Level, error) {
	switch strings.ToLower(lvl) {
	case "red":
		return Red, nil
	case "amber":
		return Amber, nil
	case "green":
		return Green, nil
	case "white":
		return White, nil
	default:
		return White, fmt.Errorf("invalid PAP level: %s", lvl)
	}
}

func Enforce(commandLevel, userLimit Level) error {
	if commandLevel > userLimit {
		return fmt.Errorf("command PAP level (%s) exceeds user limit (%s)", commandLevel, userLimit)
	}
	return nil
}

func IsDefangOutput(userLimit Level, overrideDefang, enableDefang bool) bool {
	// Never defang if overridden
	if overrideDefang {
		return false
	}
	// Always defang if enabled
	if enableDefang {
		return true
	}
	// Defang if user PAP limit is Amber or Red
	return userLimit <= Amber
}
