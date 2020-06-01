// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

// +build unit

package belogger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"runtime"
	"strconv"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"

	"github.com/insolar/block-explorer/configuration"
	"github.com/insolar/block-explorer/instrumentation/belogger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Beware, test results there depends on test file and package names!

const (
	pkgRelName   = "instrumentation/belogger/"
	testFileName = "logger_ext_test.go"
	callerRe     = "^" + pkgRelName + testFileName + ":"
)

type loggerField struct {
	Caller string
	Func   string
}

func logFields(t *testing.T, b []byte) loggerField {
	var lf loggerField
	err := json.Unmarshal(b, &lf)
	require.NoErrorf(t, err, "failed decode: '%v'", string(b))
	return lf
}

func TestExt_Global(t *testing.T) {

	l := belogger.FromContext(context.Background())

	var b bytes.Buffer
	l, err := l.Copy().WithOutput(&b).WithCaller(logcommon.CallerField).WithLevel(log.InfoLevel).Build()
	require.NoError(t, err)

	_, _, line, _ := runtime.Caller(0)
	l.Info("test")

	lf := logFields(t, b.Bytes())
	assert.Regexp(t, callerRe+strconv.Itoa(line+1), lf.Caller, "log contains call place")
	assert.Equal(t, "", lf.Func, "log not contains func name")
}

func TestExt_Global_WithFunc(t *testing.T) {
	l := belogger.FromContext(context.Background())
	var b bytes.Buffer

	l, err := l.Copy().WithOutput(&b).WithCaller(logcommon.CallerFieldWithFuncName).WithLevel(log.InfoLevel).Build()
	require.NoError(t, err)

	_, _, line, _ := runtime.Caller(0)
	l.Info("test")

	lf := logFields(t, b.Bytes())
	assert.Regexp(t, callerRe+strconv.Itoa(line+1), lf.Caller, "log contains call place")
	assert.Equal(t, t.Name(), lf.Func, "log not contains func name")
}

func TestExt_Log(t *testing.T) {
	logPut, err := belogger.NewLog(configuration.Log{
		Level:     "info",
		Adapter:   "zerolog",
		Formatter: "json",
	})
	require.NoError(t, err, "log creation")
	ctx := belogger.SetLogger(context.TODO(), logPut)

	l := belogger.FromContext(ctx)
	var b bytes.Buffer

	l, err = l.Copy().WithOutput(&b).WithCaller(logcommon.CallerField).Build()
	require.NoError(t, err)

	_, _, line, _ := runtime.Caller(0)
	l.Info("test")

	lf := logFields(t, b.Bytes())
	assert.Regexp(t, callerRe+strconv.Itoa(line+1), lf.Caller, "log contains call place")
	assert.Equal(t, "", lf.Func, "log not contains func name")
}

func TestExt_Log_WithFunc(t *testing.T) {
	logPut, err := belogger.NewLog(configuration.Log{
		Level:     "info",
		Adapter:   "zerolog",
		Formatter: "json",
	})
	require.NoError(t, err, "log creation")
	ctx := belogger.SetLogger(context.TODO(), logPut)

	l := belogger.FromContext(ctx)
	var b bytes.Buffer

	l, err = l.Copy().WithOutput(&b).WithCaller(logcommon.CallerFieldWithFuncName).Build()
	require.NoError(t, err)

	_, _, line, _ := runtime.Caller(0)
	l.Info("test")

	lf := logFields(t, b.Bytes())
	assert.Regexp(t, callerRe+strconv.Itoa(line+1), lf.Caller,
		"log contains call place")
	assert.Equal(t, t.Name(), lf.Func, "log not contains func name")
}

func TestExt_Log_SubCall(t *testing.T) {
	logPut, err := belogger.NewLog(configuration.Log{
		Level:     "info",
		Adapter:   "zerolog",
		Formatter: "json",
	})
	require.NoError(t, err, "log creation")
	logPut, err = logPut.Copy().WithCaller(logcommon.CallerFieldWithFuncName).Build()
	require.NoError(t, err)

	ctx := belogger.SetLogger(context.TODO(), logPut)

	lf, line := logCaller(ctx, t)
	assert.Regexp(t, callerRe+line, lf.Caller, "log contains call place")
	assert.Equal(t, "logCaller", lf.Func, "log not contains func name")
}

func logCaller(ctx context.Context, t *testing.T) (loggerField, string) {
	l := belogger.FromContext(ctx)
	var b bytes.Buffer

	var err error
	l, err = l.Copy().WithOutput(&b).WithCaller(logcommon.CallerFieldWithFuncName).Build()
	require.NoError(t, err)

	_, _, line, _ := runtime.Caller(0)
	l.Info("test")

	return logFields(t, b.Bytes()), strconv.Itoa(line + 1)
}

func TestExt_Global_SubCall(t *testing.T) {
	lf, line := logCallerGlobal(context.Background(), t)
	assert.Regexp(t, callerRe+line, lf.Caller, "log contains call place")
}

func logCallerGlobal(ctx context.Context, t *testing.T) (loggerField, string) {
	l := belogger.FromContext(ctx)

	var b bytes.Buffer
	var err error
	l, err = l.Copy().WithOutput(&b).WithCaller(logcommon.CallerFieldWithFuncName).WithLevel(log.InfoLevel).Build()
	require.NoError(t, err)

	_, _, line, _ := runtime.Caller(0)
	l.Info("test")
	return logFields(t, b.Bytes()), strconv.Itoa(line + 1)
}

func TestExt_Check_LoggerProxy_DoesntLoop(t *testing.T) {
	l, err := global.Logger().Copy().WithFormat(logcommon.JSONFormat).WithLevel(log.DebugLevel).Build()
	if err != nil {
		panic(err)
	}
	global.SetLogger(l.Copy().WithLevel(log.InfoLevel).MustBuild()) // enforce different instance

	l.Info("test") // here will be a stack overflow if logger proxy doesn't handle self-setting
}
