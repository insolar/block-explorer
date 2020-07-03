// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package converter

import (
	"strconv"
	"strings"

	"github.com/insolar/insolar/insolar"
)

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
