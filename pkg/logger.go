package pkg

import (
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	globalLogger *zap.Logger
	once         sync.Once
)

const (
	DebugLevel = "debug"
	InfoLevel  = "info"
	WarnLevel  = "warn"
	ErrorLevel = "error"
	FatalLevel = "fatal"
)

type Options struct {
	Level       string
	OutputPath  string
	ErrorPath   string
	Development bool
}

func DefaultOptions() Options {
	return Options{
		Level:       InfoLevel,
		OutputPath:  "stdout",
		ErrorPath:   "stderr",
		Development: false,
	}
}

func InitLogger(options Options) error {
	var err error
	once.Do(func() {
		globalLogger, err = buildLogger(options)
	})
	return err
}

func buildLogger(options Options) (*zap.Logger, error) {
	level := zap.InfoLevel
	switch options.Level {
	case DebugLevel:
		level = zap.DebugLevel
	case WarnLevel:
		level = zap.WarnLevel
	case ErrorLevel:
		level = zap.ErrorLevel
	case FatalLevel:
		level = zap.FatalLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02T15:04:05.000Z0700"))
	}
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	encoder := zapcore.NewJSONEncoder(encoderConfig)
	if options.Development {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	writers := []zapcore.WriteSyncer{zapcore.Lock(os.Stdout)}
	if options.ErrorPath != "" && options.ErrorPath != "stderr" {
		errWriter, _, err := zap.Open(options.ErrorPath)
		if err != nil {
			return nil, err
		}
		writers = append(writers, errWriter)
	}

	core := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(writers...), level)
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel)), nil
}

func GetLogger() *zap.Logger {
	if globalLogger == nil {
		InitLogger(DefaultOptions())
	}
	return globalLogger
}

func Debug(msg string, fields ...zap.Field) { GetLogger().Debug(msg, fields...) }
func Info(msg string, fields ...zap.Field)  { GetLogger().Info(msg, fields...) }
func Warn(msg string, fields ...zap.Field)  { GetLogger().Warn(msg, fields...) }
func Error(msg string, fields ...zap.Field) { GetLogger().Error(msg, fields...) }
func Fatal(msg string, fields ...zap.Field) { GetLogger().Fatal(msg, fields...) }
func Sync() error                           { return GetLogger().Sync() }
func With(fields ...zap.Field) *zap.Logger  { return GetLogger().With(fields...) }
func WithError(err error) *zap.Logger       { return GetLogger().With(zap.Error(err)) }