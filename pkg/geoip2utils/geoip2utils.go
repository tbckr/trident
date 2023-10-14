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

package geoip2utils

import (
	"net"
	"strconv"

	"github.com/oschwald/geoip2-golang"
)

const CityDBPath = "/usr/share/GeoIP/GeoLite2-City.mmdb"
const CountryDBPath = "/usr/share/GeoIP/GeoLite2-Country.mmdb"
const ASNDBPath = "/usr/share/GeoIP/GeoLite2-ASN.mmdb"

type IPInfo struct {
	IP string

	ASNumber uint
	ASName   string

	Country    string
	City       string
	PostalCode string
	Latitude   float64
	Longitude  float64
}

func (i IPInfo) String() string {
	return "IP: " + i.IP + "\n" +
		"Country: " + i.Country + "\n" +
		"City: " + i.City + "\n" +
		"PostalCode: " + i.PostalCode + "\n" +
		"Latitude: " + strconv.FormatFloat(i.Latitude, 'f', -1, 64) + "\n" +
		"Longitude: " + strconv.FormatFloat(i.Longitude, 'f', -1, 64) + "\n" +
		"ASNumber: " + strconv.Itoa(int(i.ASNumber)) + "\n" +
		"ASName: " + i.ASName + "\n"
}

func Info(ipAdr string) (*IPInfo, error) {
	ipInfo := new(IPInfo)
	ipInfo.IP = ipAdr

	ip := net.ParseIP(ipAdr)

	if err := queryCityDB(ip, ipInfo); err != nil {
		return &IPInfo{}, err
	}

	if err := queryASNDB(ip, ipInfo); err != nil {
		return &IPInfo{}, err
	}

	return ipInfo, nil
}

func queryCityDB(ip net.IP, ipInfo *IPInfo) error {
	db, err := geoip2.Open(CityDBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// City
	var city *geoip2.City
	city, err = db.City(ip)
	if err != nil {
		return err
	}
	ipInfo.Country = city.Country.Names["en"]
	ipInfo.City = city.City.Names["en"]
	ipInfo.PostalCode = city.Postal.Code
	ipInfo.Latitude = city.Location.Latitude
	ipInfo.Longitude = city.Location.Longitude

	return nil
}

func queryASNDB(ip net.IP, ipInfo *IPInfo) error {
	db, err := geoip2.Open(ASNDBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// ASN
	var asn *geoip2.ASN
	asn, err = db.ASN(ip)
	if err != nil {
		return err
	}
	ipInfo.ASNumber = asn.AutonomousSystemNumber
	ipInfo.ASName = asn.AutonomousSystemOrganization

	return nil
}
