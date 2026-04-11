// Package log provides subsystem-tagged loggers for the shell executor service.
package log

import (
	"log/slog"
	"os"
)

const SubsystemShell = "Shell" // local shell executor

func New(subsystem string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(slog.String("subsystem", subsystem))
}

// Shell returns a logger tagged [Shell] for all executor operations.
func Shell() *slog.Logger { return New(SubsystemShell) }
