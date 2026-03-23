package service

import "testing"

func TestParseWindowsList(t *testing.T) {
	t.Parallel()

	input := `
TaskName: \IITJ-LAN-AutoLogin
Status: Running
Last Result: 267009
Schedule Type: At log on
`

	got := parseWindowsList(input)
	if got["TaskName"] != `\IITJ-LAN-AutoLogin` {
		t.Fatalf("TaskName = %q", got["TaskName"])
	}
	if got["Status"] != "Running" {
		t.Fatalf("Status = %q", got["Status"])
	}
	if got["Last Result"] != "267009" {
		t.Fatalf("Last Result = %q", got["Last Result"])
	}
}

func TestTrimLogLines(t *testing.T) {
	t.Parallel()

	input := "\nline-1\n\nline-2\nline-3\n"
	got := trimLogLines(input, 2)

	if len(got) != 2 {
		t.Fatalf("len(trimLogLines()) = %d, want 2", len(got))
	}
	if got[0] != "line-2" || got[1] != "line-3" {
		t.Fatalf("trimLogLines() = %#v, want [line-2 line-3]", got)
	}
}

func TestShouldShowWindowsLastResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		lastResult string
		running    bool
		want       bool
	}{
		{name: "empty", lastResult: "", running: false, want: false},
		{name: "success code", lastResult: "0", running: false, want: false},
		{name: "success message", lastResult: "The operation completed successfully.", running: false, want: false},
		{name: "running task scheduler code decimal", lastResult: "267009", running: true, want: false},
		{name: "running task scheduler code hex", lastResult: "0x41301", running: true, want: false},
		{name: "show real failure", lastResult: "2147942402", running: false, want: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := shouldShowWindowsLastResult(tt.lastResult, tt.running); got != tt.want {
				t.Fatalf("shouldShowWindowsLastResult(%q, %t) = %t, want %t", tt.lastResult, tt.running, got, tt.want)
			}
		})
	}
}
