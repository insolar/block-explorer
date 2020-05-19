// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package dbconn

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/block-explorer/configuration"
)

func TestConnect(t *testing.T) {
	cfg := configuration.DB{
		URL:      "postgres://postgres@localhost/postgres?sslmode=disable",
		PoolSize: 100,
	}
	db, err := Connect(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)
}

func TestConnect_WrongURL(t *testing.T) {
	cfg := configuration.DB{
		URL: "wrong_url",
	}
	db, err := Connect(cfg)
	require.Error(t, err)
	require.Nil(t, db)
}
