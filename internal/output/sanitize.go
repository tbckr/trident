package output

import (
	"regexp"
	"strings"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape sequences from external data before terminal output.
func StripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

var (
	// oscSequence matches ESC ] ... terminated by BEL or ESC \ (ST).
	oscSequence = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)
	// csiSequence matches ESC [ parameter bytes, intermediate bytes, final byte (ECMA-48).
	csiSequence = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
	// escPair matches any remaining two-byte ESC x sequence, including the
	// introducer of an OSC or CSI sequence that was cut off.
	escPair = regexp.MustCompile(`\x1b[ -~]`)
)

// SanitizeTerminal removes from s what a terminal would act on instead of
// print: whole OSC and CSI sequences, every other two-byte escape sequence,
// then every remaining C0 control character (except newline and tab), DEL and
// C1 control character. Use it on text that carries external data into the
// terminal but is not produced by a service result, such as error messages
// that quote a proxy's reply. Only OSC and CSI sequences are removed whole:
// of a DCS, SOS, PM or APC string (ESC P, ESC X, ESC ^, ESC _) only the
// introducer and the terminator go, and its payload stays as visible text, as
// does the payload of a truncated OSC or CSI sequence. Invalid UTF-8 bytes
// become U+FFFD, so a raw 0x9b cannot act as an 8-bit CSI.
func SanitizeTerminal(s string) string {
	s = oscSequence.ReplaceAllString(s, "")
	s = csiSequence.ReplaceAllString(s, "")
	s = escPair.ReplaceAllString(s, "")
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\t':
			return r
		case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f:
			return -1
		}
		return r
	}, s)
}
