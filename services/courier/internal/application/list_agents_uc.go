package application

import (
	"context"

	"github.com/isaacwallace123/cortex/services/courier/internal/domain"
)

type ListAgentsUseCase struct {
	registry *Registry
}

func NewListAgentsUseCase(registry *Registry) *ListAgentsUseCase {
	return &ListAgentsUseCase{registry: registry}
}

func (uc *ListAgentsUseCase) Execute(_ context.Context) []*domain.Agent {
	return uc.registry.List()
}
