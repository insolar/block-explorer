// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package exporter

type PulseServer struct {
}

func NewPulseServer() *PulseServer {
	return &PulseServer{}
}

func (s *PulseServer) GetNextPulse(*GetNextPulseRequest, PulseExporter_GetNextPulseServer) error {
	return nil
}
