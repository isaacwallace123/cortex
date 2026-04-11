package domain

import "time"

type AgentStatus string

const (
	StatusOnline   AgentStatus = "online"
	StatusDegraded AgentStatus = "degraded"
)

type Agent struct {
	AgentID      string
	AgentType    string
	Version      string
	DeviceID     string
	DeviceName   string
	Hostname     string
	OS           string
	Arch         string
	Capabilities []string
	LocalIPs     []string
	Status       AgentStatus
	ConnectedAt  time.Time
	LastSeen     time.Time
}

func NewAgent(
	agentID, agentType, version, deviceID, deviceName, hostname, os, arch string,
	capabilities, localIPs []string,
) *Agent {
	now := time.Now()
	return &Agent{
		AgentID:      agentID,
		AgentType:    agentType,
		Version:      version,
		DeviceID:     deviceID,
		DeviceName:   deviceName,
		Hostname:     hostname,
		OS:           os,
		Arch:         arch,
		Capabilities: capabilities,
		LocalIPs:     localIPs,
		Status:       StatusOnline,
		ConnectedAt:  now,
		LastSeen:     now,
	}
}

func (a *Agent) UpdateHeartbeat(cpuPct, memPct, _ float32) {
	a.LastSeen = time.Now()
	if cpuPct > 90 || memPct > 90 {
		a.Status = StatusDegraded
	} else {
		a.Status = StatusOnline
	}
}
