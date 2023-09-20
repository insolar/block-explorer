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
