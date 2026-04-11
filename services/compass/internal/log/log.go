package log

import (
	"log/slog"
	"os"
)

func Compass() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("subsystem", "Compass")
}
