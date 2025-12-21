package validation

import (
	"fmt"
	"net"
	"regexp"
)

var (
	domainRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{1,61}[a-zA-Z0-9]\.[a-zA-Z]{2,}$`)
	asnRegex    = regexp.MustCompile(`^AS\d+$`)
)

func ValidateDomain(input string) error {
	if !domainRegex.MatchString(input) {
		return fmt.Errorf("invalid domain format: %s", input)
	}
	return nil
}

func ValidateIP(input string) error {
	if net.ParseIP(input) == nil {
		return fmt.Errorf("invalid IP format: %s", input)
	}
	return nil
}

func ValidateASN(input string) error {
	if !asnRegex.MatchString(input) {
		return fmt.Errorf("invalid ASN format: %s", input)
	}
	return nil
}
