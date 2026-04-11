// Package log provides subsystem-tagged loggers for the brain service.
// All output must identify the correct subsystem: Prism, Axiom, Vector, or brain.
package log

import (
	"log/slog"
	"os"
)

// Subsystem identities for the brain service.
const (
	SubsystemPrism  = "Prism"  // input understanding / intent parsing
	SubsystemAxiom  = "Axiom"  // reasoning / planning
	SubsystemVector = "Vector" // routing / orchestration
	SubsystemNerva  = "Nerva"  // telemetry / event emission
	SubsystemBrain  = "brain"  // bootstrap / transport (service-level)
)

// New returns a JSON structured logger tagged with the given subsystem.
func New(subsystem string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(slog.String("subsystem", subsystem))
}

// Prism returns a logger tagged [Prism] for input understanding operations.
func Prism() *slog.Logger { return New(SubsystemPrism) }

// Axiom returns a logger tagged [Axiom] for reasoning and planning operations.
func Axiom() *slog.Logger { return New(SubsystemAxiom) }

// Vector returns a logger tagged [Vector] for routing and orchestration.
func Vector() *slog.Logger { return New(SubsystemVector) }

// Nerva returns a logger tagged [Nerva] for telemetry event emission.
func Nerva() *slog.Logger { return New(SubsystemNerva) }

// Brain returns a logger tagged [brain] for service-level bootstrap/transport logs.
func Brain() *slog.Logger { return New(SubsystemBrain) }
