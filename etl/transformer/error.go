// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package transformer

type constError string

func (err constError) Error() string {
	return string(err)
}

const (
	UnsupportedRecordTypeError = constError("Record does not support")
)
