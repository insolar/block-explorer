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

func (c *BEApiClient) Pulses(t *testing.T, localVarOptionals *client.PulsesOpts) (response client.GetPulsesResponse200, err error) {
	response, rawResponse, err := c.client.PulseApi.Pulses(context.Background(), localVarOptionals)
	LogHTTP(t, rawResponse, nil, response)
	return response, err
}

func (c *BEApiClient) Pulse(t *testing.T, pulseNumber int64) (response client.PulseResponse200, err error) {
	response, rawResponse, err := c.client.PulseApi.Pulse(context.Background(), pulseNumber)
	LogHTTP(t, rawResponse, nil, response)
	return response, err
}

func (c *BEApiClient) JetDropsByPulseNumber(t *testing.T, pulseNumber int64, localVarOptionals *client.JetDropsByPulseNumberOpts) (response client.JetDropsByJetIdResponse200, err error) {
	response, rawResponse, err := c.client.JetDropApi.JetDropsByPulseNumber(context.Background(), pulseNumber, localVarOptionals)
	LogHTTP(t, rawResponse, nil, response)
	return response, err
}

func (c *BEApiClient) JetDropsByID(t *testing.T, jetDropID string) (response client.JetDropByIdResponse200, err error) {
	response, rawResponse, err := c.client.JetDropApi.JetDropByID(context.Background(), jetDropID)
	LogHTTP(t, rawResponse, nil, response)
	return response, err
}

func (c *BEApiClient) Search(t *testing.T, value string) (response client.SearchResponse200, err error) {
	response, rawResponse, err := c.client.SearchApi.Search(context.Background(), value)
	LogHTTP(t, rawResponse, nil, response)
	return response, err
}

func (c *BEApiClient) JetDropRecords(t *testing.T, jetDropId string, localVarOptionals *client.JetDropRecordsOpts) (response client.ObjectLifelineResponse200, err error) {
	response, rawResponse, err := c.client.RecordApi.JetDropRecords(context.Background(), jetDropId, localVarOptionals)
	LogHTTP(t, rawResponse, nil, response)
	return response, err
}
