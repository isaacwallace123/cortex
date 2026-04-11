package log

import (
	"log/slog"
	"os"
)

func New(subsystem string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).
		With(slog.String("subsystem", subsystem))
}

func Atlas() *slog.Logger { return New("Atlas") }
