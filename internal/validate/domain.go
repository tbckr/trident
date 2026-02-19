// Package validate provides shared input validation helpers.
package validate

import "regexp"

// domainRegexp validates RFC-compliant hostnames.
var domainRegexp = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

// IsDomain reports whether s is a valid RFC-compliant hostname.
func IsDomain(s string) bool {
	return domainRegexp.MatchString(s)
}
