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

package pap_test

import (
	"testing"

	"github.com/tbckr/trident/pkg/pap"
)

func TestIsAllowed(t *testing.T) {
	type args struct {
		environmentPapLevel pap.PapLevel
		pluginPapLevel      pap.PapLevel
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "environment RED, plugin RED, success",
			args: args{
				environmentPapLevel: pap.LevelRed,
				pluginPapLevel:      pap.LevelRed,
			},
			want: true,
		},
		{
			name: "environment RED, plugin AMBER, failure",
			args: args{
				environmentPapLevel: pap.LevelRed,
				pluginPapLevel:      pap.LevelAmber,
			},
			want: false,
		},
		{
			name: "environment AMBER, plugin RED, success",
			args: args{
				environmentPapLevel: pap.LevelAmber,
				pluginPapLevel:      pap.LevelRed,
			},
			want: true,
		},
		{
			name: "environment AMBER, plugin WHITE, failure",
			args: args{
				environmentPapLevel: pap.LevelAmber,
				pluginPapLevel:      pap.LevelWhite,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pap.IsAllowed(tt.args.environmentPapLevel, tt.args.pluginPapLevel); got != tt.want {
				t.Errorf("IsAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsEscapeData(t *testing.T) {
	type args struct {
		environmentPapLevel pap.PapLevel
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "environment RED, escape data",
			args: args{
				environmentPapLevel: pap.LevelRed,
			},
			want: true,
		},
		{
			name: "environment AMBER, escape data",
			args: args{
				environmentPapLevel: pap.LevelAmber,
			},
			want: true,
		},
		{
			name: "environment GREEN, do not escape data",
			args: args{
				environmentPapLevel: pap.LevelGreen,
			},
			want: false,
		},
		{
			name: "environment CLEAR, do not escape data",
			args: args{
				environmentPapLevel: pap.LevelClear,
			},
			want: false,
		},
		{
			name: "environment WHITE, do not escape data",
			args: args{
				environmentPapLevel: pap.LevelWhite,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pap.IsEscapeData(tt.args.environmentPapLevel); got != tt.want {
				t.Errorf("IsEscapeData() = %v, want %v", got, tt.want)
			}
		})
	}
}
