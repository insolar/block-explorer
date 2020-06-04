// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package apitests

import (
	"testing"

	"github.com/insolar/block-explorer/testutils/connectionmanager"
	"github.com/stretchr/testify/suite"
)

type dbIntegrationSuite struct {
	suite.Suite
	c connectionmanager.ConnectionManager
}

func (a *dbIntegrationSuite) BeforeTest() {
	a.c.Start(a.T())
	a.c.StartDB(a.T())
}

func (a *dbIntegrationSuite) AfterTest() {
	a.c.Stop()
}

func TestAll(t *testing.T) {
	suite.Run(t, new(dbIntegrationSuite))
}

func (a *dbIntegrationSuite) TestApi_ObjectLifeline() {

}
