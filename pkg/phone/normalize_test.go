package phone

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		defaultRegion string
		want          string
		wantErr       bool
	}{
		// Brazilian numbers with full country code (no region needed)
		{name: "BR formatted", input: "+55 11 99999-0000", defaultRegion: "", want: "+5511999990000"},
		{name: "BR digits only", input: "5511999990000", defaultRegion: "", want: "+5511999990000"},
		{name: "BR with plus", input: "+5511999990000", defaultRegion: "", want: "+5511999990000"},
		// US numbers (must be real valid numbers; 555 numbers are not valid)
		{name: "US formatted", input: "+1 (650) 253-0000", defaultRegion: "", want: "+16502530000"},
		{name: "US digits only", input: "16502530000", defaultRegion: "", want: "+16502530000"},
		// WhatsApp JIDs
		{name: "JID s.whatsapp.net", input: "5511999990000@s.whatsapp.net", defaultRegion: "", want: "+5511999990000"},
		{name: "JID c.us", input: "5511999990000@c.us", defaultRegion: "", want: "+5511999990000"},
		// Edge cases
		{name: "extra spaces and dashes", input: "  +55 11 9 9999-0000  ", defaultRegion: "", want: "+5511999990000"},
		{name: "dots as separators", input: "+55.11.99999.0000", defaultRegion: "", want: "+5511999990000"},
		// Invalid
		{name: "empty", input: "", defaultRegion: "", wantErr: true},
		{name: "letters only", input: "abcdef", defaultRegion: "", wantErr: true},
		{name: "too short", input: "+1234", defaultRegion: "", wantErr: true},

		// Local Brazilian numbers using defaultRegion="BR"
		{name: "BR local formatted parens", input: "(62) 98576-4545", defaultRegion: "BR", want: "+5562985764545"},
		{name: "BR local digits only", input: "62985764545", defaultRegion: "BR", want: "+5562985764545"},
		{name: "BR local space separated", input: "62 98576-4545", defaultRegion: "BR", want: "+5562985764545"},
		{name: "BR local SP cell", input: "11999990000", defaultRegion: "BR", want: "+5511999990000"},
		{name: "BR non-numeric prefix", input: "Whats: 62 98576-4545", defaultRegion: "BR", want: "+5562985764545"},

		// Local number without region should fail
		{name: "BR local no region", input: "62985764545", defaultRegion: "", wantErr: true},

		// US local numbers with region
		{name: "US local", input: "6502530000", defaultRegion: "US", want: "+16502530000"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Normalize(tc.input, tc.defaultRegion)
			if tc.wantErr {
				if err == nil {
					t.Errorf("Normalize(%q, %q) = %q, want error", tc.input, tc.defaultRegion, got)
				}
				return
			}
			if err != nil {
				t.Errorf("Normalize(%q, %q) unexpected error: %v", tc.input, tc.defaultRegion, err)
				return
			}
			if got != tc.want {
				t.Errorf("Normalize(%q, %q) = %q, want %q", tc.input, tc.defaultRegion, got, tc.want)
			}
		})
	}
}

func TestIsE164(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"+5511999990000", true},
		{"+15551234567", true},
		{"5511999990000", false},
		{"+1", false},
		{"", false},
		{"+abc", false},
	}
	for _, tc := range tests {
		if got := IsE164(tc.input); got != tc.want {
			t.Errorf("IsE164(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestStripJID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"5511999990000@s.whatsapp.net", "5511999990000"},
		{"5511999990000@c.us", "5511999990000"},
		{"5511999990000@g.us", "5511999990000"},
		{"5511999990000", "5511999990000"},
		{"+5511999990000", "+5511999990000"},
	}
	for _, tc := range tests {
		if got := StripJID(tc.input); got != tc.want {
			t.Errorf("StripJID(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
