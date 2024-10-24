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
