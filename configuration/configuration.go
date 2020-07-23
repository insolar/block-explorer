// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package configuration

import (
	"time"

	"go.opencensus.io/stats/view"
)

func init() {
	// todo fix problem with importing two loggers PENV-344
	view.Unregister(&view.View{Name: "log_write_delays"})
}

type BlockExplorer struct {
	Log        Log
	DB         DB
	Replicator Replicator
	Controller Controller
	Processor  Processor
}

type API struct {
	Listen       string        `insconfig:":0| API starts on this address"`
	ReadTimeout  time.Duration `insconfig:"60s| The maximum duration for reading the entire request, including the body"`
	WriteTimeout time.Duration `insconfig:"60s| The maximum duration before timing out writes of the response"`
	DB           DB
	Log          Log
}

type DB struct {
	URL             string        `insconfig:"postgres://postgres:secret@localhost:5432/postgres?sslmode=disable| Path to postgres db"`
	MaxOpenConns    int           `insconfig:"100| The maximum number of open connections to the database"`
	MaxIdleConns    int           `insconfig:"100| The maximum number of connections in the idle"`
	ConnMaxLifetime time.Duration `insconfig:"600s| The maximum amount of time a connection may be reused"`
}

type TestDB struct {
	URL          string `insconfig:"postgres://postgres@localhost/postgres?sslmode=disable| Path to postgres db"`
	PoolSize     int    `insconfig:"100| Maximum number of socket connections"`
	TestPulses   int    `insconfig:"100| amount of generated pulses"`
	TestJetDrops int    `insconfig:"1000| amount of generated jet drops"`
	TestRecords  int    `insconfig:"1000| amount of generated records"`
}

// Replicator represents a configuration of the Platform connection
type Replicator struct {
	Addr                                      string `insconfig:"127.0.0.1:5678| The gRPC server address"`
	MaxTransportMsg                           int    `insconfig:"1073741824| Maximum message size the client can send"`
	ContinuousPulseRetrievingHalfPulseSeconds uint32 `insconfig:"5| Half pulse in seconds"`
	ParallelConnections                       uint32 `insconfig:"100| Maximum parallel pulse retrievers"`
	Auth                                      Auth
}

// Auth represents the authentication of the Platform
type Auth struct {
	// warning: set false only for testing purpose within secured environment
	// if true then have to authorize
	Required      bool          `insconfig:"false| Required authorization or not"`
	URL           string        `insconfig:"https://{heavy.url}/auth/token | URL to auth endpoint"`
	Login         string        `insconfig:"login| Authorization login"`
	Password      string        `insconfig:"password| Authorization password"`
	RefreshOffset int64         `insconfig:"60| Number of seconds remain of token expiration to start token refreshing"`
	Timeout       time.Duration `insconfig:"15s| Timeout specifies a time limit for requests made by Client"`
	// warning: set true only for testing purpose within secured environment
	InsecureTLS bool `insconfig:"false| Transport layer security"`
}

// Log holds configuration for logging
type Log struct {
	// Default level for logger
	Level string `insconfig:"debug| Default level for logger"`
	// Logging adapter - only zerolog by now
	Adapter string `insconfig:"zerolog| Logging adapter - only zerolog by now"`
	// Log output format - e.g. json or text
	Formatter string `insconfig:"text| Log output format - e.g. json or text"`
	// Log output type - e.g. stderr, syslog
	OutputType string `insconfig:"stderr| Log output type - e.g. stderr, syslog"`
	// Write-parallel limit for the output
	OutputParallelLimit string `insconfig:"| Write-parallel limit for the output"`
	// Parameter for output - depends on OutputType
	OutputParams string `insconfig:"| Parameter for output - depends on OutputType"`
	// Number of regular log events that can be buffered, =0 to disable
	BufferSize int `insconfig:"0| Number of regular log events that can be buffered, =0 to disable"`
	// Number of low-latency log events that can be buffered, =-1 to disable, =0 - default size
	LLBufferSize int `insconfig:"0| Number of low-latency log events that can be buffered, =-1 to disable, =0 - default size"`
}

type Controller struct {
	PulsePeriod      int `insconfig:"10| Seconds between pulse completion tries"`
	SequentialPeriod int `insconfig:"1| Seconds between pulse sequential tries"`
	// recommend to use 20 minutes because of PENV-447
	ReloadPeriod      int `insconfig:"1200| Seconds between reloading data for same pulse tries"`
	ReloadCleanPeriod int `insconfig:"1| Seconds between launching cleaning for reloaded data map"`
}

// Processor represents for processing layer
type Processor struct {
	Workers int `insconfig:"200| The count of workers for processing transformed data"`
}

// NewLog creates new default configuration for logging
func NewLog() Log {
	return Log{
		Level:      "Info",
		Adapter:    "zerolog",
		Formatter:  "json",
		OutputType: "stderr",
		BufferSize: 0,
	}
}
