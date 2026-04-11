// Package agent implements the main Relay Agent loop: connect â†’ register â†’
// heartbeat â†’ execute tasks â†’ auto-reconnect.
package agent

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/isaacwallace123/cortex/apps/relay/internal/executor"
	"github.com/isaacwallace123/cortex/apps/relay/internal/sysinfo"
	courierv1 "github.com/isaacwallace123/cortex/services/courier/gen/courier/v1"
)

const (
	heartbeatInterval = 30 * time.Second
	maxBackoff        = 2 * time.Minute
	version           = "0.1.0"
)

// Config holds runtime configuration for the relay agent.
type Config struct {
	AgentID     string
	AgentName   string
	DeviceID    string
	CourierAddr string
}

// Run connects to Courier and runs the agent event loop with auto-reconnect.
// It blocks until ctx is canceled.
func Run(ctx context.Context, cfg Config, log *slog.Logger) {
	info := sysinfo.Gather()

	backoff := time.Second
	attempt := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if attempt > 0 {
			log.Info("[Relay] reconnecting",
				slog.Int("attempt", attempt),
				slog.Duration("backoff", backoff),
			)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			// Exponential backoff: 1s, 2s, 4s, â€¦ capped at maxBackoff
			backoff = time.Duration(math.Min(
				float64(backoff*2),
				float64(maxBackoff),
			))
		}
		attempt++

		if err := runSession(ctx, cfg, info, log); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Warn("[Relay] session ended",
				slog.String("error", err.Error()),
				slog.Int("attempt", attempt),
			)
		}
	}
}

// runSession opens one gRPC stream to Courier, handles the session, and returns
// when the stream closes or ctx is done.
func runSession(ctx context.Context, cfg Config, info sysinfo.Static, log *slog.Logger) error {
	conn, err := grpc.NewClient(cfg.CourierAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := courierv1.NewCourierServiceClient(conn)

	stream, err := client.Connect(ctx)
	if err != nil {
		return err
	}

	agentName := cfg.AgentName
	if agentName == "" {
		agentName = info.Hostname
	}

	if err := stream.Send(&courierv1.AgentMessage{
		Payload: &courierv1.AgentMessage_Register{
			Register: &courierv1.RegisterRequest{
				AgentId:      cfg.AgentID,
				AgentType:    "relay",
				Version:      version,
				DeviceId:     cfg.DeviceID,
				DeviceName:   agentName,
				Hostname:     info.Hostname,
				Os:           info.OS,
				Arch:         info.Arch,
				Capabilities: []string{"terminal"},
				LocalIps:     info.LocalIPs,
			},
		},
	}); err != nil {
		return err
	}

	// Wait for the registration ack before starting the main loop.
	ackMsg, err := stream.Recv()
	if err != nil {
		return err
	}
	ack := ackMsg.GetRegisterAck()
	if ack == nil || !ack.GetAccepted() {
		msg := "rejected by core"
		if ack != nil && ack.GetMessage() != "" {
			msg = ack.GetMessage()
		}
		log.Error("[Relay] registration rejected", slog.String("reason", msg))
		return nil
	}

	log.Info("[Relay] registered with Core",
		slog.String("agent_id", cfg.AgentID),
		slog.String("courier", cfg.CourierAddr),
		slog.String("device_name", agentName),
	)

	// Start the heartbeat goroutine. It runs until the session ends.
	hbCtx, hbCancel := context.WithCancel(ctx)
	defer hbCancel()

	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-ticker.C:
				_ = stream.Send(&courierv1.AgentMessage{
					Payload: &courierv1.AgentMessage_Heartbeat{
						Heartbeat: &courierv1.Heartbeat{
							AgentId: cfg.AgentID,
							// cpu/mem/disk: 0 for now â€” gopsutil can be added later
						},
					},
				})
			}
		}
	}()

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF || ctx.Err() != nil {
				return nil
			}
			return err
		}

		switch p := msg.Payload.(type) {
		case *courierv1.CoreMessage_TaskAssignment:
			ta := p.TaskAssignment
			log.Info("[Relay] executing task",
				slog.String("task_id", ta.GetTaskId()),
				slog.String("command", ta.GetCommand()),
			)

			result := executor.Run(ctx, ta.GetCommand(), ta.GetTimeoutMs())

			if err := stream.Send(&courierv1.AgentMessage{
				Payload: &courierv1.AgentMessage_TaskResult{
					TaskResult: &courierv1.TaskResult{
						TaskId:     ta.GetTaskId(),
						Stdout:     result.Stdout,
						Stderr:     result.Stderr,
						ExitCode:   result.ExitCode,
						DurationMs: result.DurationMs,
					},
				},
			}); err != nil {
				return err
			}

			log.Info("[Relay] task result sent",
				slog.String("task_id", ta.GetTaskId()),
				slog.Int("exit_code", int(result.ExitCode)),
				slog.Int64("duration_ms", result.DurationMs),
			)

		case *courierv1.CoreMessage_Ping:
			// Pong is implicit â€” stream activity keeps connection alive.
			log.Debug("[Relay] ping received")

		case *courierv1.CoreMessage_RegisterAck:
			// Unexpected after initial handshake â€” ignore.
		}
	}
}

// PersistentID loads an agent ID from a file, generating and saving one if absent.
func PersistentID(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		id := string(data)
		if len(id) > 0 {
			return id, nil
		}
	}

	id, err := generateID()
	if err != nil {
		return "", err
	}
	if writeErr := os.WriteFile(path, []byte(id), 0600); writeErr != nil {
		// Non-fatal â€” agent still works, just won't persist across restarts.
		slog.Warn("[Relay] could not persist agent ID",
			slog.String("path", path),
			slog.String("error", writeErr.Error()),
		)
	}
	return id, nil
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

