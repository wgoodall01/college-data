package main

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func TrimFormattedNumber(value string) string {
	value = strings.Replace(value, " ", "", -1)
	value = strings.Replace(value, "$", "", -1)
	value = strings.Replace(value, ",", "", -1)
	value = strings.Replace(value, "%", "", -1)
	return value
}

func MustParseInt(value string) int {
	valInt64, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		panic(errors.Wrap(err, "couldn't parse float:"))
	}

	return int(valInt64)
}

func MustParseFloat64(value string) float64 {
	valFloat, err := strconv.ParseFloat(value, 64)
	if err != nil {
		panic(errors.Wrap(err, "couldn't parse float:"))
	}

	return valFloat
}
