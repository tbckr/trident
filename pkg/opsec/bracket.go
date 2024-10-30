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
