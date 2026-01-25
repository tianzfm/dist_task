package logger

import (
	"os"

	"dist_task/internal/config"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func Init(cfg *config.LogConfig) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro

	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.Format == "json" {
		log = zerolog.New(os.Stdout)
	} else {
		log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout})
	}
}

func Get() zerolog.Logger {
	return log
}

func Info() *zerolog.Event {
	return log.Info()
}

func Error() *zerolog.Event {
	return log.Error()
}

func Debug() *zerolog.Event {
	return log.Debug()
}

func Warn() *zerolog.Event {
	return log.Warn()
}

func Fatal() *zerolog.Event {
	return log.Fatal()
}
