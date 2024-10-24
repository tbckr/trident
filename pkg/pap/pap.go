package pap

import (
	"errors"
	"fmt"
	"strings"
)

type PapLevel int

const (
	LevelRed PapLevel = iota
	LevelAmber
	LevelGreen
	LevelClear
	LevelWhite

	DefaultPapLevelString = "WHITE"
)

var (
	ErrInvalidPapLevel = errors.New("invalid PAP level provided")
)

type LevelConstraintError struct {
	Environment PapLevel
	Plugin      PapLevel
}

func NewPapLevelConstraintError(environment, plugin PapLevel) *LevelConstraintError {
	return &LevelConstraintError{
		Environment: environment,
		Plugin:      plugin,
	}
}

func (e *LevelConstraintError) Error() string {
	return fmt.Sprintf("PAP level constraint error: (Environment PAP Level)=%s, (Plugin PAP Level)=%s", e.Environment, e.Plugin)
}

func (l PapLevel) String() string {
	switch l {
	case LevelRed:
		return "RED"
	case LevelAmber:
		return "AMBER"
	case LevelGreen:
		return "GREEN"
	case LevelClear:
		return "CLEAR"
	case LevelWhite:
		return "WHITE"
	default:
		return "UNSET"
	}
}

func GetLevel(stringPapLevel string) (PapLevel, error) {
	switch strings.ToLower(stringPapLevel) {
	case "red":
		return LevelRed, nil
	case "amber":
		return LevelAmber, nil
	case "green":
		return LevelGreen, nil
	case "clear":
		return LevelClear, nil
	case "white":
		return LevelWhite, nil
	default:
		return LevelWhite, ErrInvalidPapLevel
	}
}

func IsAllowed(target, actual PapLevel) bool {
	return actual <= target
}

func IsEscapeData(environmentPapLevel PapLevel) bool {
	return environmentPapLevel < LevelGreen
}
