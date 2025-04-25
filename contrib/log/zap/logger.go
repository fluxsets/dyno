package zap

import (
	"github.com/fluxsets/fleet"
	slogzap "github.com/samber/slog-zap/v2"
	"go.uber.org/zap"
	"log/slog"
)

func NewLogger(ft fleet.Fleet, logLevel string) *slog.Logger {
	level := slog.LevelDebug
	atomicLevel := zap.NewAtomicLevel()
	if logLevel == "" {
		logLevel = ft.Option().LogLevel
	}

	zapLevel := zap.DebugLevel
	if logLevel != "" {
		_ = level.UnmarshalText([]byte(logLevel))
		_ = zapLevel.UnmarshalText([]byte(logLevel))
	}
	atomicLevel.SetLevel(zapLevel)

	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = atomicLevel
	//zapConfig.EncoderConfig.EncodeTime= zap.En
	zapLogger, _ := zapConfig.Build()
	slog.SetLogLoggerLevel(level)
	logger := slog.New(slogzap.Option{Level: level, Logger: zapLogger}.NewZapHandler())
	logger = logger.With("version", ft.Option().Version, "service_name", ft.Option().Name, "service_id", ft.Option().ID)
	return logger
}
