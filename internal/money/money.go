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

func ParseAmount(amount string, minorUnits uint8) (uint64, error) {
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

	wholeUint, err := strconv.ParseUint(whole, 10, 64)
	if err != nil {
		return 0, err
	}

	fractionUint := uint64(0)
	if fraction != "" {
		fractionUint, err = strconv.ParseUint(fraction, 10, 64)
		if err != nil {
			return 0, err
		}
	}

	if wholeUint > (math.MaxUint64-fractionUint)/multiplier {
		return 0, strconv.ErrRange
	}

	return (wholeUint * multiplier) + fractionUint, nil
}

func FormatAmount(amount uint64, minorUnits uint8, positive bool) string {
	var str string
	if minorUnits != 0 && amount > 0 {
		divisor := pow10(minorUnits)
		whole := amount / divisor
		fraction := amount % divisor
		str = strconv.FormatUint(whole, 10) + "," + fmt.Sprintf("%0*d", minorUnits, fraction)
	} else {
		str = strconv.FormatUint(amount, 10)
	}
	if !positive && amount > 0 {
		str = "-" + str
	}
	return str
}

func pow10(n uint8) uint64 {
	powered := uint64(1)
	for range n {
		powered *= 10
	}
	return powered
}
