// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

/*
Package belogger contains context helpers for log

Examples:

	// initialize base context with default logger with provided trace id
	ctx, log := belogger.WithTraceField(context.Background(), "TraceID")
	log.Warn("warn")

	// get logger from context
	log := belogger.FromContext(ctx)

	// initalize logger (SomeNewLogger() should return insolar.Logger)
	belogger.SetLogger(ctx, SomeNewLogger())

*/
package belogger
