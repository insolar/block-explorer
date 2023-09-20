package instrumentation

import (
	"bytes"
)

// IsEmpty checks if the given bytes are empty or filled zeroes
func IsEmpty(b []byte) bool {
	emptyByte := make([]byte, len(b))
	return bytes.Equal(b, []byte{}) ||
		bytes.Equal(b, emptyByte)
}
