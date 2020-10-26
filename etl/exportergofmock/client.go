// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package exportergofmock

import (
	"io"
	"log"

	"google.golang.org/grpc"

	"github.com/insolar/block-explorer/etl/exporter"
)

type Client struct {
	conn *grpc.ClientConn
	exporter.PulseExporterClient
	exporter.RecordExporterClient
}

func NewClient(target string) *Client {
	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	return &Client{
		conn:                 conn,
		PulseExporterClient:  exporter.NewPulseExporterClient(conn),
		RecordExporterClient: exporter.NewRecordExporterClient(conn),
	}
}

func (c *Client) ReadAllPulses(stream exporter.PulseExporter_GetNextPulseClient) []*exporter.GetNextPulseResponse {
	pulseResponses := make([]*exporter.GetNextPulseResponse, 0)
	for {
		pulseResp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		pulseResponses = append(pulseResponses, pulseResp)
	}
	return pulseResponses
}

func (c *Client) ReadAllRecords(stream exporter.RecordExporter_GetRecordsClient) []*exporter.GetRecordsResponse {
	recordsResponses := make([]*exporter.GetRecordsResponse, 0)
	for {
		pulseResp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		recordsResponses = append(recordsResponses, pulseResp)
	}
	return recordsResponses
}
