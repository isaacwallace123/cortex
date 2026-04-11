package application

import (
	"fmt"
	"log/slog"

	courierv1 "github.com/isaacwallace123/cortex/services/courier/gen/courier/v1"
	"github.com/isaacwallace123/cortex/services/courier/internal/domain"
)

// EventPublisher lets the connect use-case emit agent lifecycle events without
// depending on a concrete Nerva adapter.
type EventPublisher interface {
	Publish(subsystem, eventName, payload string) error
}

// ConnectUseCase handles the full lifecycle of a single agent connection.
// It is called once per Connect stream and blocks until the stream closes.
type ConnectUseCase struct {
	registry  *Registry
	publisher EventPublisher
	log       *slog.Logger
}

func NewConnectUseCase(registry *Registry, publisher EventPublisher, log *slog.Logger) *ConnectUseCase {
	return &ConnectUseCase{registry: registry, publisher: publisher, log: log}
}

// Handle runs the agent session: register â†’ event loop â†’ cleanup.
func (uc *ConnectUseCase) Handle(stream courierv1.CourierService_ConnectServer) error {
	// First message must be a RegisterRequest — anything else is a protocol error.
	msg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("courier.Connect: recv register: %w", err)
	}

	reg := msg.GetRegister()
	if reg == nil {
		return fmt.Errorf("courier.Connect: first message must be RegisterRequest")
	}

	agent := domain.NewAgent(
		reg.GetAgentId(), reg.GetAgentType(), reg.GetVersion(),
		reg.GetDeviceId(), reg.GetDeviceName(), reg.GetHostname(),
		reg.GetOs(), reg.GetArch(),
		reg.GetCapabilities(), reg.GetLocalIps(),
	)

	conn := newAgentConn(agent, stream)
	uc.registry.Register(conn)
	defer uc.registry.Unregister(agent.AgentID)

	uc.log.Info("[Courier] agent connected",
		slog.String("agent_id", agent.AgentID),
		slog.String("device_name", agent.DeviceName),
		slog.String("hostname", agent.Hostname),
		slog.String("os", agent.OS),
		slog.String("arch", agent.Arch),
	)

	_ = uc.publisher.Publish("Courier", "agent.registered", fmt.Sprintf(
		`{"agent_id":%q,"device_name":%q,"hostname":%q,"os":%q,"arch":%q}`,
		agent.AgentID, agent.DeviceName, agent.Hostname, agent.OS, agent.Arch,
	))

	// Acknowledge the registration so the agent knows it was accepted.
	if err := stream.Send(&courierv1.CoreMessage{
		Payload: &courierv1.CoreMessage_RegisterAck{
			RegisterAck: &courierv1.RegisterAck{
				AgentId:  agent.AgentID,
				Accepted: true,
			},
		},
	}); err != nil {
		return fmt.Errorf("courier.Connect: send ack: %w", err)
	}

	for {
		msg, err := stream.Recv()
		if err != nil {
			break
		}

		switch p := msg.Payload.(type) {
		case *courierv1.AgentMessage_Heartbeat:
			hb := p.Heartbeat
			uc.registry.UpdateHeartbeat(agent.AgentID, hb.GetCpuPercent(), hb.GetMemoryPercent(), hb.GetDiskPercent())
			uc.log.Debug("[Courier] heartbeat",
				slog.String("agent_id", agent.AgentID),
				slog.Float64("cpu", float64(hb.GetCpuPercent())),
				slog.Float64("mem", float64(hb.GetMemoryPercent())),
			)
			_ = uc.publisher.Publish("Courier", "agent.heartbeat", fmt.Sprintf(
				`{"agent_id":%q,"cpu":%.1f,"mem":%.1f,"disk":%.1f}`,
				agent.AgentID, hb.GetCpuPercent(), hb.GetMemoryPercent(), hb.GetDiskPercent(),
			))

		case *courierv1.AgentMessage_TaskResult:
			tr := p.TaskResult
			result := &domain.TaskResult{
				TaskID:     tr.GetTaskId(),
				Stdout:     tr.GetStdout(),
				Stderr:     tr.GetStderr(),
				ExitCode:   tr.GetExitCode(),
				DurationMs: tr.GetDurationMs(),
			}
			if !conn.DeliverResult(result) {
				uc.log.Warn("[Courier] received result for unknown task",
					slog.String("task_id", tr.GetTaskId()),
					slog.String("agent_id", agent.AgentID),
				)
			}
			uc.log.Info("[Courier] task completed",
				slog.String("task_id", tr.GetTaskId()),
				slog.String("agent_id", agent.AgentID),
				slog.Int("exit_code", int(tr.GetExitCode())),
				slog.Int64("duration_ms", tr.GetDurationMs()),
			)
		}
	}

	uc.log.Info("[Courier] agent disconnected", slog.String("agent_id", agent.AgentID))
	_ = uc.publisher.Publish("Courier", "agent.disconnected", fmt.Sprintf(
		`{"agent_id":%q,"device_name":%q}`, agent.AgentID, agent.DeviceName,
	))
	return nil
}

