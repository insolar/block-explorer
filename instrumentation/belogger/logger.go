package belogger

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
	"go.opencensus.io/stats/view"

	"github.com/insolar/block-explorer/configuration"
)

const TimestampFormat = "2006-01-02T15:04:05.000000000Z07:00"

const bePrefix = "github.com/insolar/block-explorer/"
const skipFrameBaselineAdjustment = 0

func init() {
	// todo fix problem with importing two loggers PENV-344
	view.Unregister(&view.View{Name: "log_write_delays"})
	initBilog()
	initZlog()

	// NB! initialize adapters' globals before the next call
	global.TrySetDefaultInitializer(func() (log.LoggerBuilder, error) {
		return newLogger(defaultLogConfig())
	})
}

func defaultLogConfig() configuration.Log {
	logCfg := configuration.NewLog()

	// enforce buffer-less for a non-configured logger
	logCfg.BufferSize = 0
	logCfg.LLBufferSize = -1
	return logCfg
}

func fileLineMarshaller(file string, line int) string {
	var skip = 0
	if idx := strings.Index(file, bePrefix); idx != -1 {
		skip = idx + len(bePrefix)
	}
	return file[skip:] + ":" + strconv.Itoa(line)
}

func newLogger(cfg configuration.Log) (log.LoggerBuilder, error) {
	defaults := DefaultLoggerSettings()
	pCfg, err := ParseLogConfigWithDefaults(cfg, defaults)
	if err != nil {
		return log.LoggerBuilder{}, err
	}

	var logBuilder log.LoggerBuilder

	pCfg.SkipFrameBaselineAdjustment = skipFrameBaselineAdjustment

	msgFmt := logfmt.GetDefaultLogMsgFormatter()
	msgFmt.TimeFmt = TimestampFormat

	switch strings.ToLower(cfg.Adapter) {
	case "zerolog":
		logBuilder, err = newZerologAdapter(pCfg, msgFmt)
	case "bilog":
		logBuilder, err = newBilogAdapter(pCfg, msgFmt)
	default:
		return log.LoggerBuilder{}, errors.New("invalid logger config, unknown adapter")
	}

	switch {
	case err != nil:
		return log.LoggerBuilder{}, err
	case logBuilder.IsZero():
		return log.LoggerBuilder{}, errors.New("logger was not initialized")
	default:
		return logBuilder, nil
	}
}

// newLog creates a new logger with the given configuration
func NewLog(cfg configuration.Log) (logger log.Logger, err error) {
	var b log.LoggerBuilder
	b, err = newLogger(cfg)
	if err == nil {
		logger, err = b.Build()
		if err == nil {
			return logger, nil
		}
	}
	return log.Logger{}, err
}

var loggerKey = struct{}{}

func InitLogger(ctx context.Context, cfg configuration.Log, serviceName string) (context.Context, log.Logger) {
	belog, err := NewLog(cfg)
	if err != nil {
		panic(err)
	}

	fields := map[string]interface{}{"loginstance": "node"}
	if serviceName != "" {
		fields["service_name"] = serviceName
	}
	belog = belog.WithFields(fields)

	ctx = SetLogger(ctx, belog)
	global.SetLogger(belog)

	return ctx, belog
}

// FromContext returns logger from context.
func FromContext(ctx context.Context) log.Logger {
	return getLogger(ctx)
}

// SetLogger returns context with provided insolar.Logger,
func SetLogger(ctx context.Context, l log.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

func UpdateLogger(ctx context.Context, fn func(log.Logger) (log.Logger, error)) context.Context {
	lOrig := FromContext(ctx)
	lNew, err := fn(lOrig)
	if err != nil {
		panic(err)
	}
	if lOrig == lNew {
		return ctx
	}
	return SetLogger(ctx, lNew)
}

// SetLoggerLevel and set logLevel on logger and returns context with the new logger
func WithLoggerLevel(ctx context.Context, logLevel log.Level) context.Context {
	if logLevel == log.NoLevel {
		return ctx
	}
	oldLogger := FromContext(ctx)
	b := oldLogger.Copy()
	if b.GetLogLevel() == logLevel {
		return ctx
	}
	logCopy, err := b.WithLevel(logLevel).Build()
	if err != nil {
		oldLogger.Error("failed to set log level: ", err.Error())
		return ctx
	}
	return SetLogger(ctx, logCopy)
}

// WithField returns context with logger initialized with provided field's key value and logger itself.
func WithField(ctx context.Context, key string, value string) (context.Context, log.Logger) {
	l := getLogger(ctx).WithField(key, value)
	return SetLogger(ctx, l), l
}

// WithFields returns context with logger initialized with provided fields map.
func WithFields(ctx context.Context, fields map[string]interface{}) (context.Context, log.Logger) {
	l := getLogger(ctx).WithFields(fields)
	return SetLogger(ctx, l), l
}

func getLogger(ctx context.Context) log.Logger {
	val := ctx.Value(loggerKey)
	if val == nil {
		return global.CopyForContext()
	}
	return val.(log.Logger)
}

// TestContext returns context with initalized log field "testname" equal t.Name() value.
func TestContext(t *testing.T) context.Context {
	ctx, _ := WithField(context.Background(), "testname", t.Name())
	return ctx
}

func GetLoggerLevel(ctx context.Context) log.Level {
	return getLogger(ctx).Copy().GetLogLevel()
}
