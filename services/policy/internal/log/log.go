// Package log provides subsystem-tagged loggers for the policy service.
// All output must identify the Aegis subsystem.
package log

import (
	"log/slog"
	"os"
)

const SubsystemAegis = "Aegis" // policy / approval gates

func New(subsystem string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(slog.String("subsystem", subsystem))
}

// Aegis returns a logger tagged [Aegis] for all policy service operations.
func Aegis() *slog.Logger { return New(SubsystemAegis) }
