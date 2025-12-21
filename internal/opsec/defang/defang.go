package defang

import (
	"strings"
)

func Defang(input string, defangEnabled bool) string {
	if !defangEnabled {
		return input
	}

	// 1. Defang protocol
	res := strings.ReplaceAll(input, "http://", "hxxp://")
	res = strings.ReplaceAll(res, "https://", "hxxps://")

	// 2. Defang only the LAST dot/separator to avoid over-defanging
	// For IPs: 1.2.3.4 -> 1.2.3[.]4
	// For domains: example.com -> example[.]com
	lastDot := strings.LastIndex(res, ".")
	// If there is no dot in the domain, return the domain as is
	if lastDot == -1 {
		return res
	}
	// If dot is already bracketed, return the domain as is
	if lastDot > 0 && lastDot < len(res)-1 && res[lastDot-1] == '[' && res[lastDot+1] == ']' {
		return res
	}
	// Replace the dot with [.]
	return res[:lastDot] + "[.]" + res[lastDot+1:]
}
