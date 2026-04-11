package log

import (
	"log/slog"
	"os"
)

func Beacon() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).
		With(slog.String("service", "beacon"))
}
