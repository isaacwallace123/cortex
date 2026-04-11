// Package log provides subsystem-tagged loggers for the inference service.
package log

import (
	"log/slog"
	"os"
)

const SubsystemInference = "inference"

func New(subsystem string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(slog.String("subsystem", subsystem))
}

func Inference() *slog.Logger { return New(SubsystemInference) }
