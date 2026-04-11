package log

import (
	"log/slog"
	"os"
)

func Workspace() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).
		With(slog.String("service", "workspace"))
}
