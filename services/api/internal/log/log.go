// Package log provides subsystem-tagged loggers for the api gateway service.
package log

import (
	"log/slog"
	"os"
)

const SubsystemAPI = "api"

func New(subsystem string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(slog.String("subsystem", subsystem))
}

func API() *slog.Logger { return New(SubsystemAPI) }
