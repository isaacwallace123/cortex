package log

import (
	"log/slog"
	"os"
)

func Catalyst() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}
