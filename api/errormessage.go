// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

type ErrorMessage struct {
	Error []string `json:"error"`
}

func NewSingleMessageError(err string) ErrorMessage {
	return ErrorMessage{Error: []string{err}}
}
