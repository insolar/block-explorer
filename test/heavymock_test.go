// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build heavy_mock_integration

package test

import (
	"testing"
)

func TestFailNowHeavymock(t *testing.T) {
	panic("oops!")
}
