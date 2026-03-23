package login

import "testing"

func TestExtractToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "fortigate javascript redirect",
			body: `window.location="https://gateway.iitj.ac.in:1003/fgtauth?abc123token"`,
			want: "abc123token",
		},
		{
			name: "single quoted redirect",
			body: `location.href='https://gateway.iitj.ac.in:1003/fgtauth?token-456'`,
			want: "token-456",
		},
		{
			name: "token ends at whitespace",
			body: "fgtauth?token789 \nnext",
			want: "token789",
		},
		{
			name: "missing token marker",
			body: "<html>already authenticated</html>",
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := extractToken(tt.body); got != tt.want {
				t.Fatalf("extractToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractMagic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "hidden magic input",
			html: `<input type="hidden" name="magic" value="real-magic-value">`,
			want: "real-magic-value",
		},
		{
			name: "magic among other fields",
			html: `<form><input name="username"><input name="magic" value="abc123"><input name="password"></form>`,
			want: "abc123",
		},
		{
			name: "missing magic value",
			html: `<input type="hidden" name="token" value="abc">`,
			want: "",
		},
		{
			name: "missing closing quote",
			html: `<input name="magic" value="broken`,
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := extractMagic(tt.html); got != tt.want {
				t.Fatalf("extractMagic() = %q, want %q", got, tt.want)
			}
		})
	}
}
