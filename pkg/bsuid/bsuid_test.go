package bsuid

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantCC      string
		wantID      string
		wantErr     bool
	}{
		{name: "valid BR", input: "BR.abc123def456", wantCC: "BR", wantID: "abc123def456"},
		{name: "valid US", input: "US.xyz789ghi012", wantCC: "US", wantID: "xyz789ghi012"},
		{name: "valid DE", input: "DE.ABCDEFGH12", wantCC: "DE", wantID: "ABCDEFGH12"},
		// Invalid
		{name: "empty", input: "", wantErr: true},
		{name: "no dot", input: "BRabc123def456", wantErr: true},
		{name: "lowercase country code", input: "br.abc123def456", wantErr: true},
		{name: "mixed case country code", input: "Br.abc123def456", wantErr: true},
		{name: "ID exactly 8 chars", input: "BR.abcd1234", wantCC: "BR", wantID: "abcd1234"}, // 8 chars — valid
		{name: "ID 7 chars too short", input: "BR.abc1234"[:len("BR.")+7], wantErr: true},
		{name: "ID 6 chars too short", input: "BR.abc123", wantErr: true}, // 6 chars
		{name: "ID only 7 chars", input: "BR.abcde12"[:len("BR.")+7], wantErr: true},
		{name: "special chars in ID", input: "BR.abc-123456", wantErr: true},
		{name: "dot in ID", input: "BR.abc.123456", wantErr: true},
		{name: "too long", input: "BR." + string(make([]byte, 62)), wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("Parse(%q) = %+v, want error", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("Parse(%q) unexpected error: %v", tc.input, err)
				return
			}
			if got.CountryCode != tc.wantCC {
				t.Errorf("Parse(%q).CountryCode = %q, want %q", tc.input, got.CountryCode, tc.wantCC)
			}
			if got.ID != tc.wantID {
				t.Errorf("Parse(%q).ID = %q, want %q", tc.input, got.ID, tc.wantID)
			}
			if got.Raw != tc.input {
				t.Errorf("Parse(%q).Raw = %q, want %q", tc.input, got.Raw, tc.input)
			}
		})
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"BR.abc123def456", true},
		{"US.xyz789ghi012", true},
		{"br.abc123def456", false},
		{"BRabc123def456", false},
		{"BR.abc123", false}, // 6 chars < 8
		{"BR.abc-1234567", false},
		{"", false},
	}
	for _, tc := range tests {
		if got := IsValid(tc.input); got != tc.want {
			t.Errorf("IsValid(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestIsBSUID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
		desc  string
	}{
		{"BR.abc123def456", true, "valid BSUID"},
		{"US.xyz789ghi012", true, "valid BSUID US"},
		{"+5511999990000", false, "phone number"},
		{"5511999990000", false, "digits only"},
		{"5511999990000@s.whatsapp.net", false, "JID"},
		{"BR.abc123", false, "too short ID"},
		{"br.abc123def456", false, "lowercase CC"},
		{"", false, "empty"},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			if got := IsBSUID(tc.input); got != tc.want {
				t.Errorf("IsBSUID(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
