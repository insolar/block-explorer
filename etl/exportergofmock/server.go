// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package exportergofmock

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/insolar/block-explorer/etl/exporter"
)

type ExporterServerMemory struct {
	Listen     string
	grpcServer *grpc.Server
	Data       *DataMock
}

func NewExporterMock(initPulse int64) *ExporterServerMemory {
	s := grpc.NewServer()
	d := NewDataMock(initPulse)
	fp, err := GetFreePort()
	if err != nil {
		log.Fatal(err)
	}
	exporter.RegisterRecordExporterServer(s, NewRecordServerMock(d))
	exporter.RegisterPulseExporterServer(s, NewPulseServerMock(d))
	m := &ExporterServerMemory{
		Listen:     fmt.Sprintf("0.0.0.0:%d", fp),
		grpcServer: s,
		Data:       d,
	}
	if err := m.Start(); err != nil {
		log.Fatal(err)
	}
	return m
}

func (s *ExporterServerMemory) Start() error {
	if s.grpcServer == nil {
		return errors.New("gRPC server is required")
	}
	l, err := net.Listen("tcp", s.Listen)
	if err != nil {
		return errors.Wrapf(err, "failed to start gPRC server on %s", s.Listen)
	}
	s.run(l)
	time.Sleep(1 * time.Second)
	return nil
}

func (s *ExporterServerMemory) Stop() error {
	s.grpcServer.GracefulStop()
	return nil
}

func (s *ExporterServerMemory) run(l net.Listener) {
	go func() {
		err := s.grpcServer.Serve(l)
		if err != nil {
			log.Fatal(err)
		}
	}()
}
