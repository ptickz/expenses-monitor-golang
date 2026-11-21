package money

import (
	"errors"
	"math"
	"strconv"
	"testing"
)

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		amount     uint64
		minorUnits uint8
		positive   bool
		want       string
	}{
		{amount: 1050, minorUnits: 2, positive: true, want: "10,50"},
		{amount: 1005, minorUnits: 2, positive: true, want: "10,05"},
		{amount: 1005, minorUnits: 2, positive: false, want: "-10,05"},
		{amount: 1005, minorUnits: 0, positive: true, want: "1005"},
		{amount: 10000001, minorUnits: 4, positive: false, want: "-1000,0001"},
		{amount: 0, minorUnits: 4, positive: false, want: "0"},
		{amount: 0, minorUnits: 0, positive: false, want: "0"},
		{amount: 0, minorUnits: 2, positive: true, want: "0"},
		{amount: 3, minorUnits: 2, positive: true, want: "0,03"},
		{amount: 3, minorUnits: 1, positive: true, want: "0,3"},
	}

	for _, tt := range tests {
		got := FormatAmount(tt.amount, tt.minorUnits, tt.positive)
		if got != tt.want {
			t.Errorf("formatAmount(%d, %d, %v) = %q, want %q",
				tt.amount, tt.minorUnits, tt.positive, got, tt.want)
		}
	}
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		amount     string
		minorUnits uint8
		want       uint64
		wantError  error
	}{
		{amount: "22,53", minorUnits: 2, want: 2253, wantError: nil},
		{amount: "0.53", minorUnits: 2, want: 53, wantError: nil},
		{amount: "0.53", minorUnits: 1, want: 0, wantError: ErrWrongFraction},
		{amount: "0.53", minorUnits: 0, want: 0, wantError: ErrWrongFraction},
		{amount: "1sdfsdf00.53", minorUnits: 0, want: 0, wantError: ErrWrongFormat},
		{amount: "20", minorUnits: 2, want: 2000, wantError: nil},
		{amount: "20,5", minorUnits: 2, want: 2050, wantError: nil},
		{amount: "184467440737095516", minorUnits: 2, want: 18446744073709551600, wantError: nil},
		{amount: "184467440737095517", minorUnits: 2, want: 0, wantError: strconv.ErrRange},
		{amount: "184467440737095516,15", minorUnits: 2, want: math.MaxUint64, wantError: nil},
		{amount: "184467440737095516,16", minorUnits: 2, want: 0, wantError: strconv.ErrRange},
		{amount: "99999999999999999999", minorUnits: 0, want: 0, wantError: strconv.ErrRange},
		{amount: "22", minorUnits: 0, want: 22, wantError: nil},
		{amount: "1 000,5", minorUnits: 2, want: 100050, wantError: nil},
		{amount: "22,", minorUnits: 2, want: 0, wantError: ErrWrongFormat},
		{amount: ",5", minorUnits: 2, want: 0, wantError: ErrWrongFormat},
		{amount: "", minorUnits: 2, want: 0, wantError: ErrWrongFormat},
		{amount: "-5", minorUnits: 2, want: 0, wantError: ErrWrongFormat},
	}

	for _, tt := range tests {
		got, err := ParseAmount(tt.amount, tt.minorUnits)
		if !errors.Is(err, tt.wantError) || (got != tt.want) {
			t.Errorf("parseAmount(%q, %d) = %v, %v, want %v, %v",
				tt.amount, tt.minorUnits, got, err, tt.want, tt.wantError)
		}
	}
}
