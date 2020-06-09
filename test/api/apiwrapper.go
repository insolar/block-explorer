// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

import (
	"net/http"
	"testing"

	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
)

type BEApiClient struct {
	t      *testing.T
	client client.APIClient
}

func NewBeApiClient(t *testing.T, basePath string) *BEApiClient {
	cfg := client.Configuration{
		BasePath:   basePath,
		HTTPClient: http.DefaultClient,
	}
	return &BEApiClient{
		t:      t,
		client: *client.NewAPIClient(&cfg),
	}
}

func (c *BEApiClient) ObjectLifeline(objectRef string, localVarOptionals *client.ObjectLifelineOpts) (response client.JetDropRecordsResponse200, err error) {
	response, rawResponse, err := c.client.RecordApi.ObjectLifeline(nil, objectRef, localVarOptionals)
	LogHttp(c.t, rawResponse, nil, response)
	return response, err
}
