package cmd

import "fmt"

// ErrConflictingFlags is returned when mutually exclusive flags are used together.
func ErrConflictingFlags(msg string) error {
	return fmt.Errorf("conflicting flags: %s", msg)
}

// ErrInvalidOutputFormat is returned when an unsupported output format is specified.
func ErrInvalidOutputFormat(format string) error {
	return fmt.Errorf("invalid output format %q: must be one of: text, json, plain", format)
}

// ErrInvalidPAPLimit is returned when an invalid PAP limit is specified.
func ErrInvalidPAPLimit(limit string) error {
	return fmt.Errorf("invalid PAP limit %q: must be one of: red, amber, green, white", limit)
}

// ErrInvalidConcurrency is returned when concurrency is set to an invalid value.
func ErrInvalidConcurrency(value int) error {
	return fmt.Errorf("invalid concurrency value %d: must be at least 1", value)
}
