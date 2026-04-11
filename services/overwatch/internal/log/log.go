// Package log provides the subsystem-tagged logger for the Overwatch analysis engine.
package log

import (
	"log/slog"
	"os"
)

func New(subsystem string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(slog.String("subsystem", subsystem))
}

func Overwatch() *slog.Logger { return New("Overwatch") }
