// Package log provides the subsystem-tagged logger for the Nerva event bus.
package log

import (
	"log/slog"
	"os"
)

const SubsystemNerva = "Nerva"

func New(subsystem string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(slog.String("subsystem", subsystem))
}

// Nerva returns a logger tagged [Nerva] for all event bus operations.
func Nerva() *slog.Logger { return New(SubsystemNerva) }
