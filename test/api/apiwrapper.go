// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/insolar/spec-insolar-block-explorer-api/v1/client"
)

type BEApiClient struct {
	client client.APIClient
}

func NewBeAPIClient(basePath string) *BEApiClient {
	cfg := client.Configuration{
		BasePath:   basePath,
		HTTPClient: http.DefaultClient,
	}
	return &BEApiClient{
		client: *client.NewAPIClient(&cfg),
	}
}

func (c *BEApiClient) ObjectLifeline(t *testing.T, objectRef string, localVarOptionals *client.ObjectLifelineOpts) (response client.ObjectLifelineResponse200, err error) {
	response, rawResponse, err := c.client.RecordApi.ObjectLifeline(context.Background(), objectRef, localVarOptionals)
	LogHTTP(t, rawResponse, nil, response)
	return response, err
}

func (c *BEApiClient) Pulses(t *testing.T, localVarOptionals *client.PulsesOpts) (response client.PulsesResponse200, err error) {
	response, rawResponse, err := c.client.PulseApi.Pulses(context.Background(), localVarOptionals)
	LogHTTP(t, rawResponse, nil, response)
	return response, err
}
