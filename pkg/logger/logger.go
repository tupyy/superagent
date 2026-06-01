package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Init(format string, logLevel string) *zap.Logger {
	lvl := zapcore.DebugLevel
	level, err := zapcore.ParseLevel(logLevel)
	if err == nil {
		lvl = level
	}

	loggerCfg := &zap.Config{
		Level:    zap.NewAtomicLevelAt(lvl),
		Encoding: format,
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "severity",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeDuration: zapcore.MillisDurationEncoder, EncodeCaller: zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	plain, err := loggerCfg.Build(zap.AddStacktrace(zap.DPanicLevel))
	if err != nil {
		panic(err)
	}

	return plain
}
