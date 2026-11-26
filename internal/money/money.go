package money

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrWrongFormat   = errors.New("wrong amount format")
	ErrWrongFraction = errors.New("wrong fraction part size")
	amountRegexp     = regexp.MustCompile(`^(\d+)(?:[.,](\d+))?$`)
)

func ParseAmount(amount string, minorUnits int16) (int64, error) {
	amount = strings.Join(strings.Fields(amount), "")
	matches := amountRegexp.FindStringSubmatch(amount)
	if matches == nil {
		return 0, ErrWrongFormat
	}

	whole, fraction := matches[1], matches[2]
	multiplier := pow10(minorUnits)

	if len(fraction) > int(minorUnits) {
		return 0, ErrWrongFraction
	}
	fraction += strings.Repeat("0", int(minorUnits)-len(fraction))

	wholeInt, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, err
	}

	fractionInt := int64(0)
	if fraction != "" {
		fractionInt, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, err
		}
	}

	if wholeInt > (math.MaxInt64-fractionInt)/multiplier {
		return 0, strconv.ErrRange
	}

	return wholeInt*multiplier + fractionInt, nil
}

func FormatAmount(amount int64, minorUnits int16) string {
	if minorUnits == 0 || amount == 0 {
		return strconv.FormatInt(amount, 10)
	}

	divisor := pow10(minorUnits)
	whole := amount / divisor
	fraction := amount % divisor
	sign := ""
	if amount < 0 {
		sign = "-"
		whole = -whole
		fraction = -fraction
	}

	return sign + strconv.FormatInt(whole, 10) + "," + fmt.Sprintf("%0*d", int(minorUnits), fraction)
}

func pow10(n int16) int64 {
	powered := int64(1)
	for range n {
		powered *= 10
	}
	return powered
}
