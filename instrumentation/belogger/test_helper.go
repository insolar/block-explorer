package belogger

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
)

func NewTestLogger(target logcommon.TestingRedirectTarget) log.Logger {
	return NewTestLoggerExt(target, "")
}

func NewTestLoggerExt(target logcommon.TestingRedirectTarget, adapter string) log.Logger {
	if target == nil {
		panic("illegal value")
	}
	logCfg := defaultLogConfig()
	if adapter != "" {
		logCfg.Adapter = adapter
	}

	l, err := newLogger(logCfg)
	if err != nil {
		panic(err)
	}
	return l.WithMetrics(logcommon.LogMetricsResetMode).
		WithFormat(logcommon.TextFormat).
		WithCaller(logcommon.CallerField).
		WithOutput(logcommon.TestingLoggerOutput{Target: target}).
		MustBuild()
}

func SetTestOutput(target logcommon.TestingRedirectTarget) {
	global.SetLogger(NewTestLogger(target))
}
