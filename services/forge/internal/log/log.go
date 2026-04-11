// Package log provides the subsystem-tagged logger for the Forge service.
package log

import (
	"log/slog"
	"os"
)

const SubsystemForge = "Forge"

func New(subsystem string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(slog.String("subsystem", subsystem))
}

// Forge returns a logger tagged [Forge] for filesystem executor operations.
func Forge() *slog.Logger { return New(SubsystemForge) }
