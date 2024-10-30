// Copyright (c) 2024 Tim <tbckr>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
//
// SPDX-License-Identifier: MIT

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
