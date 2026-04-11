package log

import (
	"log/slog"
	"os"
)

func Sovereign() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("subsystem", "Sovereign")
}
