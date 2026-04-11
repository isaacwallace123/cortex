package log

import (
	"log/slog"
	"os"
)

const SubsystemCourier = "Courier"

func New(subsystem string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(slog.String("subsystem", subsystem))
}

func Courier() *slog.Logger { return New(SubsystemCourier) }
