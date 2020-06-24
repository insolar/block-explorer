// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetJetIdParents(t *testing.T) {
	tests := map[string]struct {
		input  string
		output []string
	}{
		"empty input": {"", []string{}},
		"0":           {"0", []string{"0"}},
		"01":          {"01", []string{"0", "01"}},
		"010":         {"010", []string{"0", "01", "010"}},
		"0010":        {"0010", []string{"0", "00", "001", "0010"}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			parents := GetJetIdParents(test.input)
			for i, output := range test.output {
				require.Equal(t, output, parents[i])
			}
			require.EqualValues(t, test.output, parents)
		})
	}
}
