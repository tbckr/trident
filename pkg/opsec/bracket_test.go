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

package opsec_test

import (
	"testing"

	"github.com/tbckr/trident/pkg/opsec"
)

func TestBracketDomain(t *testing.T) {
	type args struct {
		domain string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "domain without dots",
			args: args{
				domain: "com",
			},
			want: "com",
		},
		{
			name: "domain with one dot",
			args: args{
				domain: "example.com",
			},
			want: "example[.]com",
		},
		{
			name: "domain with two dots",
			args: args{
				domain: "www.example.com",
			},
			want: "www.example[.]com",
		},
		{
			name: "domain with three dots",
			args: args{
				domain: "www.example.co.uk",
			},
			want: "www.example.co[.]uk",
		},
		{
			name: "domain one dot and already bracked",
			args: args{
				domain: "example[.]com",
			},
			want: "example[.]com",
		},
		{
			name: "domain two dots and already bracked",
			args: args{
				domain: "www.example[.]com",
			},
			want: "www.example[.]com",
		},
		{
			name: "domain three dots and already bracked",
			args: args{
				domain: "www.example.co[.]uk",
			},
			want: "www.example.co[.]uk",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := opsec.BracketDomain(tt.args.domain); got != tt.want {
				t.Errorf("BracketDomain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnbracketDomain(t *testing.T) {
	type args struct {
		domain string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "domain without dots",
			args: args{
				domain: "com",
			},
			want: "com",
		},
		{
			name: "domain with one dot",
			args: args{
				domain: "example[.]com",
			},
			want: "example.com",
		},
		{
			name: "domain with two dots",
			args: args{
				domain: "www.example[.]com",
			},
			want: "www.example.com",
		},
		{
			name: "domain with three dots",
			args: args{
				domain: "www.example.co[.]uk",
			},
			want: "www.example.co.uk",
		},
		{
			name: "domain one dot and already unbracked",
			args: args{
				domain: "example.com",
			},
			want: "example.com",
		},
		{
			name: "domain two dots and already unbracked",
			args: args{
				domain: "www.example.com",
			},
			want: "www.example.com",
		},
		{
			name: "domain three dots and already unbracked",
			args: args{
				domain: "www.example.co.uk",
			},
			want: "www.example.co.uk",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := opsec.UnbracketDomain(tt.args.domain); got != tt.want {
				t.Errorf("UnbracketDomain() = %v, want %v", got, tt.want)
			}
		})
	}
}
