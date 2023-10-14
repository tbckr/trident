// Copyright (c) 2023 Tim <tbckr>
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

package dnsutils

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"strings"

	"github.com/miekg/dns"
)

const GoogleDNS = "8.8.8.8:53"

type Report struct {
	Domain string
	MX     []MXServer
}

type MXServer struct {
	Host     string
	Priority uint16
}

func (r Report) String() string {
	var s string
	for _, mx := range r.MX {
		s += mx.Host + "\n"
	}
	return s
}

func Retrieve(domains io.Reader) []Report {
	var reports []Report

	c := new(dns.Client)

	sc := bufio.NewScanner(domains)
	for sc.Scan() {
		domain := strings.ToLower(sc.Text())

		report := new(Report)
		report.Domain = domain

		if err := queryRecordType(c, domain, dns.TypeMX, report); err != nil {
			slog.Error("Failed to query record", "domain", domain, "type", "MX", "error", err)
		}

		reports = append(reports, *report)
	}
	return reports
}

func queryRecordType(c *dns.Client, domain string, dnsRecordType uint16, report *Report) error {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dnsRecordType)
	m.RecursionDesired = true

	r, _, err := c.Exchange(m, GoogleDNS)
	if err != nil {
		return err
	}

	switch r.Rcode {
	case dns.RcodeSuccess:
		// do nothing
	case dns.RcodeNameError:
		return errors.New("no such domain")
	default:
		// do nothing
	}

	for _, answer := range r.Answer {
		switch record := answer.(type) {
		case *dns.MX:
			server := MXServer{
				Host:     record.Mx,
				Priority: record.Preference,
			}
			report.MX = append(report.MX, server)
		default:
			// TODO
		}
	}
	return nil
}
