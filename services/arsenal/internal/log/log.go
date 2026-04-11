// Package log exposes pre-tagged loggers for each Arsenal subsystem.
package log

import (
	"log/slog"
	"os"
)

func Arsenal() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("subsystem", "Arsenal")
}
