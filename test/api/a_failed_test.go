// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package api

import (
	"testing"
)

func TestSuccess(t *testing.T) {
	t.Log("hello success")
}

func TestFail(t *testing.T) {
	t.Log("hello fail")
	t.Fatalf("oops!")
}
