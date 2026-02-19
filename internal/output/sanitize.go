package output

import "regexp"

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape sequences from external data before terminal output.
func StripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}
