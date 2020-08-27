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
	"github.com/stretchr/testify/require"
)

type BEApiClient struct {
	Client client.APIClient
}

func NewBeAPIClient(basePath string) *BEApiClient {
	cfg := client.Configuration{
		BasePath:   basePath,
		HTTPClient: http.DefaultClient,
	}
	return &BEApiClient{
		Client: *client.NewAPIClient(&cfg),
	}
}

func (c *BEApiClient) ObjectLifeline(t *testing.T, objectRef string, localVarOptionals *client.ObjectLifelineOpts) (response client.ObjectLifelineResponse200) {
	response, rawResponse, err := c.Client.RecordApi.ObjectLifeline(context.Background(), objectRef, localVarOptionals)
	require.NoError(t, err)
	LogHTTP(t, rawResponse, nil, response)
	return response
}

func (c *BEApiClient) Pulses(t *testing.T, localVarOptionals *client.PulsesOpts) (response client.GetPulsesResponse200) {
	response, rawResponse, err := c.Client.PulseApi.Pulses(context.Background(), localVarOptionals)
	require.NoError(t, err)
	LogHTTP(t, rawResponse, nil, response)
	return response
}

func (c *BEApiClient) PulsesWithError(t *testing.T, localVarOptionals *client.PulsesOpts, expError string) (response client.GetPulsesResponse200) {
	response, rawResponse, err := c.Client.PulseApi.Pulses(context.Background(), localVarOptionals)
	require.Error(t, err)
	require.Equal(t, expError, err.Error())
	LogHTTP(t, rawResponse, nil, response)
	return response
}

func (c *BEApiClient) Pulse(t *testing.T, pulseNumber int64) (response client.PulseResponse200) {
	response, rawResponse, err := c.Client.PulseApi.Pulse(context.Background(), pulseNumber)
	require.NoError(t, err)
	LogHTTP(t, rawResponse, nil, response)
	return response
}

func (c *BEApiClient) PulseWithError(t *testing.T, pulseNumber int64, expError string) (response client.PulseResponse200) {
	response, rawResponse, err := c.Client.PulseApi.Pulse(context.Background(), pulseNumber)
	require.Error(t, err)
	require.Equal(t, expError, err.Error())
	LogHTTP(t, rawResponse, nil, response)
	return response
}

func (c *BEApiClient) JetDropsByPulseNumber(t *testing.T, pulseNumber int64, localVarOptionals *client.JetDropsByPulseNumberOpts) (response client.JetDropsByJetIdResponse200) {
	response, rawResponse, err := c.Client.JetDropApi.JetDropsByPulseNumber(context.Background(), pulseNumber, localVarOptionals)
	require.NoError(t, err)
	LogHTTP(t, rawResponse, nil, response)
	return response
}

func (c *BEApiClient) JetDropsByPulseNumberWithError(t *testing.T, pulseNumber int64, localVarOptionals *client.JetDropsByPulseNumberOpts, expError string) (response client.JetDropsByJetIdResponse200) {
	response, rawResponse, err := c.Client.JetDropApi.JetDropsByPulseNumber(context.Background(), pulseNumber, localVarOptionals)
	require.Error(t, err)
	require.Equal(t, expError, err.Error())
	LogHTTP(t, rawResponse, nil, response)
	return response
}

func (c *BEApiClient) JetDropsByID(t *testing.T, jetDropID string) (response client.JetDropByIdResponse200) {
	response, rawResponse, err := c.Client.JetDropApi.JetDropByID(context.Background(), jetDropID)
	require.NoError(t, err)
	LogHTTP(t, rawResponse, nil, response)
	return response
}

func (c *BEApiClient) JetDropsByIDWithError(t *testing.T, jetDropID string, expError string) (response client.JetDropByIdResponse200) {
	response, rawResponse, err := c.Client.JetDropApi.JetDropByID(context.Background(), jetDropID)
	require.Error(t, err)
	require.Equal(t, expError, err.Error())
	LogHTTP(t, rawResponse, nil, response)
	return response
}

func (c *BEApiClient) JetDropsByJetID(t *testing.T, jetID string, opts *client.JetDropsByJetIDOpts) (response client.JetDropsByJetIdResponse200) {
	response, rawResponse, err := c.Client.JetDropApi.JetDropsByJetID(context.Background(), jetID, opts)
	require.NoError(t, err)
	LogHTTP(t, rawResponse, nil, response)
	return response
}

func (c *BEApiClient) JetDropsByJetIDWithError(t *testing.T, jetID string, opts *client.JetDropsByJetIDOpts, expError string) (response client.JetDropsByJetIdResponse200) {
	response, rawResponse, err := c.Client.JetDropApi.JetDropsByJetID(context.Background(), jetID, opts)
	require.Error(t, err)
	require.Equal(t, expError, err.Error())
	LogHTTP(t, rawResponse, nil, response)
	return response
}

func (c *BEApiClient) Search(t *testing.T, value string) (response client.SearchResponse200) {
	response, rawResponse, err := c.Client.SearchApi.Search(context.Background(), value)
	require.NoError(t, err)
	LogHTTP(t, rawResponse, nil, response)
	return response
}

func (c *BEApiClient) SearchWithError(t *testing.T, value string, expError string) (response client.SearchResponse200) {
	response, rawResponse, err := c.Client.SearchApi.Search(context.Background(), value)
	require.Error(t, err)
	require.Equal(t, expError, err.Error())
	LogHTTP(t, rawResponse, nil, response)
	return response
}

func (c *BEApiClient) JetDropRecords(t *testing.T, jetDropID string, localVarOptionals *client.JetDropRecordsOpts) (response client.ObjectLifelineResponse200) {
	response, rawResponse, err := c.Client.RecordApi.JetDropRecords(context.Background(), jetDropID, localVarOptionals)
	require.NoError(t, err)
	LogHTTP(t, rawResponse, nil, response)
	return response
}

func (c *BEApiClient) JetDropRecordsWithError(t *testing.T, jetDropID string, localVarOptionals *client.JetDropRecordsOpts, expError string) (response client.ObjectLifelineResponse200) {
	response, rawResponse, err := c.Client.RecordApi.JetDropRecords(context.Background(), jetDropID, localVarOptionals)
	require.Error(t, err)
	require.Equal(t, expError, err.Error())
	LogHTTP(t, rawResponse, nil, response)
	return response
}
