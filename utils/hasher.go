// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package utils

import (
	"bytes"
	"encoding/gob"
)

func Compare(a, b []byte) bool {
	a = append(a, b...)
	c := 0
	for _, x := range a {
		c ^= int(x)
	}
	return c == 0
}

func Hash(o interface{}) ([]byte, error) {
	var b bytes.Buffer
	err := gob.NewEncoder(&b).Encode(o)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
