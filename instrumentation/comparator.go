// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

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
