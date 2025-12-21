package defang

import (
	"strings"
)

func Defang(input string, defangEnabled bool) string {
	if !defangEnabled {
		return input
	}

	// Order matters: URLs first, then domains/IPs
	res := input
	res = strings.ReplaceAll(res, "http://", "hxxp://")
	res = strings.ReplaceAll(res, "https://", "hxxps://")

	// Defang dots in middle of strings (domains/IPs)
	// Simple approach: replace all "." with "[.]"
	// More sophisticated logic would use regex to match boundaries
	res = strings.ReplaceAll(res, ".", "[.]")

	return res
}
