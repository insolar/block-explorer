package converter

import (
	"strconv"
	"strings"
	"time"

	"github.com/insolar/insolar/insolar"
)

// NanosecondsInSecond contains a count of the nanoseconds in one second
const NanosecondsInSecond = int64(time.Second / time.Nanosecond)

// JetIDToString returns the string representation of JetID
func JetIDToString(id insolar.JetID) string {
	depth, prefix := id.Depth(), id.Prefix()
	if depth == 0 {
		return ""
	}
	res := strings.Builder{}
	for i := uint8(0); i < depth; i++ {
		bytePos, bitPos := i/8, 7-i%8

		byteValue := prefix[bytePos]
		bitValue := byteValue >> uint(bitPos) & 0x01
		bitString := strconv.Itoa(int(bitValue))
		res.WriteString(bitString)
	}
	return res.String()
}

// SecondsToNanos convert the seconds to a nanosecond
func SecondsToNanos(seconds uint32) int64 {
	return int64(seconds) * NanosecondsInSecond
}

// NanosToSeconds convert the nanosecond to a seconds
func NanosToSeconds(nanoseconds int64) int64 {
	return nanoseconds / NanosecondsInSecond
}
