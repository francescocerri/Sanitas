package config

import "testing"

func TestUnquote(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "double-quoted value with a # inside (the real-world case that broke INVITE_URL_BASE)",
			input: `"http://localhost:5173/#/user-activation"`,
			want:  "http://localhost:5173/#/user-activation",
		},
		{name: "single-quoted value", input: `'value'`, want: "value"},
		{name: "unquoted value is left as-is", input: "plain-value", want: "plain-value"},
		{name: "mismatched quotes are left as-is", input: `"mismatched'`, want: `"mismatched'`},
		{name: "empty value", input: "", want: ""},
		{name: "single quote character alone is left as-is", input: `"`, want: `"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := unquote(tt.input); got != tt.want {
				t.Errorf("unquote(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
