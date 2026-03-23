package installer

import "testing"

func TestParseYesNo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		answer     string
		defaultYes bool
		want       bool
	}{
		{name: "empty uses yes default", answer: "", defaultYes: true, want: true},
		{name: "empty uses no default", answer: "", defaultYes: false, want: false},
		{name: "explicit yes", answer: "yes", defaultYes: false, want: true},
		{name: "explicit no", answer: "n", defaultYes: true, want: false},
		{name: "mixed case", answer: "Y", defaultYes: false, want: true},
		{name: "unknown falls back to default yes", answer: "maybe", defaultYes: true, want: true},
		{name: "unknown falls back to default no", answer: "maybe", defaultYes: false, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := parseYesNo(tt.answer, tt.defaultYes); got != tt.want {
				t.Fatalf("parseYesNo(%q, %t) = %t, want %t", tt.answer, tt.defaultYes, got, tt.want)
			}
		})
	}
}
