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
