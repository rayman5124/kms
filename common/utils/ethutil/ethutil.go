package ethutil

import (
	"bytes"
	"log"
	"math/big"
	"regexp"
	"strings"
)

func PadLeftTo32Bytes(buffer []byte) []byte {
	buffer = bytes.TrimLeft(buffer, "\x00")
	for len(buffer) < 32 {
		buffer = append([]byte{0}, buffer...)
	}
	return buffer
}

func ParseUnit(val string, decimal uint8) *big.Int {
	trimed := regexp.MustCompile("[^0-9.-]+").ReplaceAllString(val, "")
	if trimed == "" {
		log.Println("not numeric value")
		return nil
	}

	pad := decimal
	if lastInd := strings.LastIndex(trimed, "."); lastInd != -1 {
		pad -= uint8(lastInd)
	}
	padded := strings.Replace(trimed, ".", "", 1) + strings.Repeat("0", int(pad))

	parsed := new(big.Int)
	parsed.SetString(padded, 10)
	return parsed
}
