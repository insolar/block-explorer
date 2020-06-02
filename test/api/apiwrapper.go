// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

import (
	"testing"

	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
	"github.com/stretchr/testify/require"
)

type BEApiClient struct {
	t *testing.T
	client client.APIClient
}

func NewBeApiClient(t *testing.T, basePath string) *BEApiClient {
	cfg := client.Configuration{
		BasePath:      basePath,
		Host:          "",
		Scheme:        "",
		DefaultHeader: nil,
		UserAgent:     "",
		Debug:         false,
		Servers:       nil,
		HTTPClient:    nil,
	}
	return &BEApiClient{
		t:      t,
		client: *client.NewAPIClient(&cfg),
	}
}

func (c *BEApiClient) ObjectLifeline(objectRef string, localVarOptionals *client.ObjectLifelineOpts) client.JetDropRecordsResponse200 {
	response, rawResponse, err := c.client.RecordApi.ObjectLifeline(nil, objectRef, localVarOptionals)
	LogHttp(c.t, rawResponse, nil, response)
	require.NoError(c.t, err, "Error after executing http request")
	validateResponse(c.t, rawResponse)
	return response
}
