package opsec

import (
	"strings"
)

func BracketDomain(domain string) string {
	// Get last index of dot
	lastDot := strings.LastIndex(domain, ".")
	// If there is no dot in the domain, return the domain as is
	if lastDot == -1 {
		return domain
	}
	// If dot is already bracketed, return the domain as is
	if domain[lastDot-1] == '[' && domain[lastDot+1] == ']' {
		return domain
	}
	// Bracket the last dot
	return domain[:lastDot] + "[.]" + domain[lastDot+1:]
}

func UnbracketDomain(domain string) string {
	if !strings.Contains(domain, "[.]") {
		return domain
	}
	return strings.ReplaceAll(domain, "[.]", ".")
}
