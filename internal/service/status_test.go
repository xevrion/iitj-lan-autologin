package service

import (
	"testing"

	"github.com/iitj/iitj-lan-autologin/internal/detect"
)

func TestFallback(t *testing.T) {
	t.Parallel()

	if got := fallback("value", "alt"); got != "value" {
		t.Fatalf("fallback returned %q, want %q", got, "value")
	}
	if got := fallback("   ", "alt"); got != "alt" {
		t.Fatalf("fallback returned %q, want %q", got, "alt")
	}
}

func TestRenderPlatform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		platform detect.Platform
		want     string
	}{
		{
			name:     "linux with distro",
			platform: detect.Platform{OS: "linux", Distro: "fedora", Arch: "amd64"},
			want:     "linux / fedora / amd64",
		},
		{
			name:     "windows without distro",
			platform: detect.Platform{OS: "windows", Arch: "arm64"},
			want:     "windows / arm64",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := renderPlatform(tt.platform); got != tt.want {
				t.Fatalf("renderPlatform() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLogLineColoring(t *testing.T) {
	t.Parallel()

	ui := newPalette(false)
	tests := []struct {
		name string
		line string
		want string
	}{
		{name: "plain line", line: "already authenticated", want: "already authenticated"},
		{name: "error line", line: "error: trigger check failed", want: "error: trigger check failed"},
		{name: "warning line", line: "warning: retrying", want: "warning: retrying"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ui.logLine(tt.line); got != tt.want {
				t.Fatalf("logLine() = %q, want %q", got, tt.want)
			}
		})
	}
}
