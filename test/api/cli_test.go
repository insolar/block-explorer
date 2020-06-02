// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test123(t *testing.T) {
	c := NewBeApiClient(t, "localhost")
	resp := c.ObjectLifeline("", nil)
	require.NotEmpty(t, resp.Result)
	require.NotEmpty(t, resp.Result[0].PulseNumber)
}
