// Package log provides subsystem-tagged loggers for the memory service.
// All output must identify the Echo subsystem.
package log

import (
	"log/slog"
	"os"
)

const SubsystemEcho = "Echo" // memory / session persistence

func New(subsystem string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(slog.String("subsystem", subsystem))
}

// Echo returns a logger tagged [Echo] for all memory service operations.
func Echo() *slog.Logger { return New(SubsystemEcho) }
