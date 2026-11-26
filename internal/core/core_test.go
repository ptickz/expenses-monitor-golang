package core

import (
	"strings"
	"testing"
)

func TestParseTimeZone(t *testing.T) {
	tests := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{in: "Asia/Tbilisi", want: "Asia/Tbilisi", wantOK: true},
		{in: "America/Argentina/Buenos_Aires", want: "America/Argentina/Buenos_Aires", wantOK: true},
		{in: "UTC", want: "UTC", wantOK: true},
		{in: "", wantOK: false},
		{in: "Local", wantOK: false},
		{in: "asia/tbilisi", wantOK: false},
		{in: "Mars/Olympus", wantOK: false},
		{in: "../etc/passwd", wantOK: false},
	}
	for _, tt := range tests {
		loc, ok := ParseTimeZone(tt.in)
		if ok != tt.wantOK {
			t.Errorf("ParseTimeZone(%q) ok = %t, want %t", tt.in, ok, tt.wantOK)
			continue
		}
		if !ok {
			if loc != nil {
				t.Errorf("ParseTimeZone(%q) loc = %v, want nil", tt.in, loc)
			}
			continue
		}
		if loc.String() != tt.want {
			t.Errorf("ParseTimeZone(%q) = %q, want %q", tt.in, loc.String(), tt.want)
		}
	}
}

func TestValidateAccountName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "latin", in: "Revolut", want: true},
		{name: "cyrillic", in: "Сбер", want: true},
		{name: "surrounding spaces", in: "  TBC  ", want: true},
		{name: "empty", in: "", want: false},
		{name: "only spaces", in: " \t\n ", want: false},
		{name: "savings", in: SavingsAccountName, want: false},
		{name: "savings other case", in: "СБЕРЕГАТЕЛЬНЫЙ", want: false},
		{name: "savings with spaces", in: " сберегательный ", want: false},
		{name: "64 cyrillic runes", in: strings.Repeat("я", 64), want: true},
		{name: "65 cyrillic runes", in: strings.Repeat("я", 65), want: false},
		{name: "64 runes after trim", in: " " + strings.Repeat("a", 64) + " ", want: true},
	}
	for _, tt := range tests {
		if got := ValidateAccountName(tt.in); got != tt.want {
			t.Errorf("%s: ValidateAccountName(%q) = %t, want %t", tt.name, tt.in, got, tt.want)
		}
	}
}
